package converter

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	gosync "sync"
	"time"

	"github.com/P4sTela/matsu-sonic/internal/config"
	"github.com/P4sTela/matsu-sonic/internal/store"
)

// Broadcaster is the interface for sending messages to WebSocket clients.
type Broadcaster interface {
	Broadcast(data []byte)
}

// Manager manages converter instances and runs conversion jobs.
type Manager struct {
	cfg *config.Config
	st  *store.DB
	hub Broadcaster

	mu         gosync.Mutex
	running    map[string]bool // job id -> running
	workerSem  chan struct{}
}

// NewManager creates a conversion manager.
func NewManager(cfg *config.Config, st *store.DB, hub Broadcaster) *Manager {
	workers := cfg.ConverterWorkers
	if workers <= 0 {
		workers = 1
	}
	return &Manager{
		cfg:       cfg,
		st:        st,
		hub:       hub,
		running:   make(map[string]bool),
		workerSem: make(chan struct{}, workers),
	}
}

// Run starts a conversion for the given file using the named converter.
// It returns the conversion record ID.
func (m *Manager) Run(fileID, converterName string) (*store.Conversion, error) {
	// Resolve converter config
	var convConf *config.ConverterConf
	for i := range m.cfg.Converters {
		if m.cfg.Converters[i].Name == converterName && m.cfg.Converters[i].Enabled {
			convConf = &m.cfg.Converters[i]
			break
		}
	}
	if convConf == nil {
		return nil, fmt.Errorf("converter %q not found or disabled", converterName)
	}

	// Resolve the synced file
	sf, err := m.st.GetFile(fileID)
	if err != nil {
		return nil, fmt.Errorf("file %q not found: %w", fileID, err)
	}
	if sf.IsFolder {
		return nil, fmt.Errorf("cannot convert a folder")
	}

	// Validate input pattern match
	if !matchPattern(convConf.InputPattern, sf.Name) {
		return nil, fmt.Errorf("file %q does not match converter pattern %q", sf.Name, convConf.InputPattern)
	}

	// Build output path
	inPath := sf.LocalPath
	outPath, err := outputPath(m.cfg.LocalSyncDir, convConf, sf)
	if err != nil {
		return nil, err
	}

	// Create or update conversion record
	conv, err := m.st.InsertConversion(fileID, converterName, inPath)
	if err != nil {
		return nil, fmt.Errorf("insert conversion: %w", err)
	}

	// Run asynchronously
	go m.runConversion(conv, *convConf, inPath, outPath)

	return conv, nil
}

func (m *Manager) runConversion(conv *store.Conversion, cfg config.ConverterConf, inPath, outPath string) {
	m.workerSem <- struct{}{}
	defer func() { <-m.workerSem }()

	m.mu.Lock()
	m.running[conv.ID] = true
	m.mu.Unlock()
	defer func() {
		m.mu.Lock()
		delete(m.running, conv.ID)
		m.mu.Unlock()
	}()

	log.Printf("[converter] starting %s via %s: %s -> %s", conv.FileID, conv.Converter, inPath, outPath)

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		m.fail(conv.ID, fmt.Sprintf("mkdir: %v", err))
		return
	}

	// Record original file state before conversion
	origSize, origMod := fileStat(inPath)

	// Build command
	cmdTokens := expandCommand(cfg.Command, inPath, outPath, cfg.OutputDir)
	if len(cmdTokens) == 0 {
		m.fail(conv.ID, "empty command after expansion")
		return
	}

	// Broadcast start
	m.broadcast(conv.ID, "convert_start", map[string]any{
		"file_id":   conv.FileID,
		"converter": conv.Converter,
		"input":     inPath,
		"output":    outPath,
	})

	// Run with timeout
	timeout := 30 * time.Minute
	if err := runCmd(timeout, cmdTokens, cfg.Env, func(progress float64) {
		m.broadcast(conv.ID, "convert_progress", map[string]any{
			"file_id":   conv.FileID,
			"converter": conv.Converter,
			"progress":  progress,
		})
	}); err != nil {
		m.fail(conv.ID, err.Error())
		m.broadcast(conv.ID, "convert_error", map[string]any{
			"file_id":   conv.FileID,
			"converter": conv.Converter,
			"error":     err.Error(),
		})
		return
	}

	// Verify output file exists
	if _, err := os.Stat(outPath); os.IsNotExist(err) {
		m.fail(conv.ID, "output file was not created")
		m.broadcast(conv.ID, "convert_error", map[string]any{
			"file_id":   conv.FileID,
			"converter": conv.Converter,
			"error":     "output file was not created",
		})
		return
	}

	// Record original file state used for this conversion
	if origSize == 0 && origMod == "" {
		origSize, origMod = fileStat(inPath)
	}

	// Mark completed
	if err := m.st.FinishConversion(conv.ID, outPath, origSize, origMod); err != nil {
		log.Printf("[converter] finish db error: %v", err)
	}

	// Broadcast completion
	outInfo, _ := os.Stat(outPath)
	outSize := int64(0)
	if outInfo != nil {
		outSize = outInfo.Size()
	}
	m.broadcast(conv.ID, "convert_complete", map[string]any{
		"file_id":    conv.FileID,
		"converter":  conv.Converter,
		"output":     outPath,
		"output_size": outSize,
	})

	log.Printf("[converter] completed %s via %s: %s -> %s (%d bytes)", conv.FileID, conv.Converter, inPath, outPath, outSize)
}

func (m *Manager) fail(id, errMsg string) {
	log.Printf("[converter] failed %s: %s", id, errMsg)
	if err := m.st.FailConversion(id, errMsg); err != nil {
		log.Printf("[converter] fail db error: %v", err)
	}
}

func (m *Manager) broadcast(jobID, typ string, data map[string]any) {
	if m.hub == nil {
		return
	}
	msg := map[string]any{"type": typ, "job_id": jobID, "data": data}
	b, err := json.Marshal(msg)
	if err != nil {
		return
	}
	m.hub.Broadcast(b)
}

// outputPath builds the absolute output path for a conversion.
func outputPath(syncRoot string, c *config.ConverterConf, sf *store.SyncedFile) (string, error) {
	base := filepath.Clean(syncRoot)
	rel, err := filepath.Rel(base, sf.LocalPath)
	if err != nil {
		return "", fmt.Errorf("cannot determine relative path: %w", err)
	}
	stem := strings.TrimSuffix(filepath.Base(rel), filepath.Ext(rel))

	var dir string
	if c.OutputDir != "" {
		dir = filepath.Join(base, c.OutputDir, filepath.Dir(rel))
	} else {
		dir = filepath.Join(base, filepath.Dir(rel))
	}
	return filepath.Join(dir, stem+c.OutputExtension), nil
}

// matchPattern checks whether name matches the given glob pattern.
func matchPattern(pattern, name string) bool {
	matched, _ := filepath.Match(pattern, name)
	return matched
}

// fileStat returns the size and modification time of path.
func fileStat(path string) (int64, string) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, ""
	}
	return info.Size(), info.ModTime().UTC().Format(time.RFC3339Nano)
}
