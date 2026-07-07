package utils

// Shared defaults for HTTP/gRPC (flags live in cmd/node/config.go).
type NetDefaults struct {
	GRPCTimeoutSec int
	RetryAttempts  int
}

func DefaultNet() NetDefaults {
	return NetDefaults{GRPCTimeoutSec: DefaultGRPCTimeoutSec, RetryAttempts: 2}
}
