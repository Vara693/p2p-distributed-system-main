package heartbeat

import "time"

// DefaultInterval between local health sweeps when not configured elsewhere.
func DefaultInterval() time.Duration {
	return 3 * time.Second
}
