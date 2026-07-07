package peer

import (
	"time"

	"edi_sem2/internal/utils"
)

// Manager applies timeouts and state transitions for the local registry view.
type Manager struct {
	Registry      *Registry
	InactiveAfter time.Duration
}

func NewManager(reg *Registry) *Manager {
	return &Manager{
		Registry:      reg,
		InactiveAfter: time.Duration(utils.DefaultHeartbeatSec*3) * time.Second,
	}
}

// Sweep marks peers inactive if they have not been seen; does not touch self.
func (m *Manager) Sweep(now time.Time) {
	if m == nil || m.Registry == nil {
		return
	}
	for _, p := range m.Registry.Snapshot() {
		if p.ID == m.Registry.SelfID() {
			continue
		}
		if p.State == StateLeft {
			continue
		}
		if now.Sub(p.LastSeen) > m.InactiveAfter {
			m.Registry.MarkState(p.ID, StateInactive)
		}
	}
}

func (m *Manager) Leave(id string) {
	if m == nil || m.Registry == nil {
		return
	}
	m.Registry.MarkState(id, StateLeft)
	m.Registry.Remove(id)
}
