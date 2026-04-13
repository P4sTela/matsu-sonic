package distribution

import (
	"context"
	"fmt"

	"github.com/P4sTela/matsu-sonic/internal/config"
)

// Manager creates and manages distribution targets.
type Manager struct {
	targets map[string]Target
}

// NewManager creates a Manager from the given config targets.
func NewManager(configs []config.DistTargetConf) *Manager {
	m := &Manager{targets: make(map[string]Target)}
	for _, c := range configs {
		m.targets[c.Name] = newTarget(c)
	}
	return m
}

// Get returns a target by name.
func (m *Manager) Get(name string) (Target, error) {
	t, ok := m.targets[name]
	if !ok {
		return nil, fmt.Errorf("target %q not found", name)
	}
	return t, nil
}

// Distribute copies a file to the named target.
func (m *Manager) Distribute(ctx context.Context, targetName, src, destRelative string) (string, error) {
	t, err := m.Get(targetName)
	if err != nil {
		return "", err
	}
	return t.Distribute(ctx, src, destRelative)
}

// Reload rebuilds targets from updated config.
func (m *Manager) Reload(configs []config.DistTargetConf) {
	m.targets = make(map[string]Target)
	for _, c := range configs {
		m.targets[c.Name] = newTarget(c)
	}
}

func newTarget(c config.DistTargetConf) Target {
	switch c.Type {
	case "smb":
		return &SMBTarget{
			Server:   c.Server,
			Share:    c.Share,
			Username: c.Username,
			Password: c.Password,
			Domain:   c.Domain,
		}
	default:
		return &LocalTarget{BasePath: c.Path}
	}
}
