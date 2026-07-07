package heartbeat

import "time"

// InactiveAfter is the default duration without contact before marking a peer inactive.
func InactiveAfter() time.Duration {
	return 9 * time.Second
}
