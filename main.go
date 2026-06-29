package main

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/P4sTela/matsu-sonic/internal/config"
	"github.com/P4sTela/matsu-sonic/internal/distribution"
	"github.com/P4sTela/matsu-sonic/internal/drive"
	"github.com/P4sTela/matsu-sonic/internal/handler"
	"github.com/P4sTela/matsu-sonic/internal/server"
	"github.com/P4sTela/matsu-sonic/internal/store"
	msync "github.com/P4sTela/matsu-sonic/internal/sync"
)

var version = "dev"

// defaultConfigPath returns the default location of config.json. The app is
// portable by default: state (config, database, token) lives in a .gdrive-sync
// folder next to the executable, so the whole folder can be moved around
// without writing into the user's home/config directories.
//
// If the executable directory is not writable (e.g. installed under
// Program Files / /Applications, a read-only mount, or macOS app
// translocation), it falls back to the per-user config directory so the app
// still works instead of failing to start.
func defaultConfigPath() string {
	if dir, ok := portableBaseDir(); ok {
		return filepath.Join(dir, ".gdrive-sync", "config.json")
	}
	if ucd, err := os.UserConfigDir(); err == nil && ucd != "" {
		return filepath.Join(ucd, "matsu-sonic", "config.json")
	}
	return filepath.Join(".gdrive-sync", "config.json")
}

// portableBaseDir returns the executable's directory if it is writable.
func portableBaseDir() (string, bool) {
	exe, err := os.Executable()
	if err != nil {
		return "", false
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	dir := filepath.Dir(exe)
	f, err := os.CreateTemp(dir, ".write-test-*")
	if err != nil {
		return "", false
	}
	name := f.Name()
	f.Close()
	os.Remove(name)
	return dir, true
}

func main() {
	var (
		port        int
		configPath  string
		showVersion bool
	)

	flag.IntVar(&port, "port", 8765, "server port")
	flag.StringVar(&configPath, "config", defaultConfigPath(),
		"config file path (default: .gdrive-sync next to the executable)")
	flag.BoolVar(&showVersion, "version", false, "show version and exit")
	flag.Parse()

	if showVersion {
		fmt.Println("gdrive-sync", version)
		return
	}

	// Load config
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// Init DB
	dbDir := filepath.Dir(configPath)
	dbPath := filepath.Join(dbDir, "gdrive-sync.db")
	db, err := store.New(dbPath)
	if err != nil {
		log.Fatalf("init database: %v", err)
	}
	defer db.Close()

	// Init Drive client (may fail if not configured yet — that's OK)
	ctx := context.Background()
	drv, driveErr := drive.NewDriveClient(ctx, &cfg)
	if driveErr != nil {
		log.Printf("Drive client not available: %v (configure credentials first)", driveErr)
	}

	// WebSocket hub
	hub := server.NewHub()
	go hub.Run()

	// Sync engine
	engine := msync.NewSyncEngine(&cfg, drv, db, hub)

	// Distribution manager
	distMgr := distribution.NewManager(cfg.DistTargets)

	// HTTP server
	srv := server.New(hub)

	// Register API routes
	h := &handler.Handler{
		Config:      &cfg,
		ConfigPath:  configPath,
		Store:       db,
		Drive:       drv,
		Engine:      engine,
		DistManager: distMgr,
		Hub:         hub,
	}
	h.RegisterRoutes(srv.Router)

	// Mount embedded frontend
	frontendSub, err := fs.Sub(frontendFS, "frontend/dist")
	if err != nil {
		log.Printf("frontend not embedded (dev mode): %v", err)
	} else {
		srv.MountSPA(frontendSub)
	}

	// Graceful shutdown
	sigCtx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	httpServer := &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", port),
		Handler: srv.Router,
	}

	go func() {
		log.Printf("Starting server on http://127.0.0.1:%d", port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-sigCtx.Done()
	log.Println("Shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	httpServer.Shutdown(shutdownCtx)
}
