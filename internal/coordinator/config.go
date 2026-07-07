package coordinator

import (
	"log/slog"
)

// Config holds static configuration for a PeerHost.
type Config struct {
	Logger *slog.Logger

	DataDir            string
	GrpcListenAddr     string // e.g. 0.0.0.0:50051
	HTTPListenAddr     string // e.g. 0.0.0.0:8080
	BootstrapURL       string // e.g. http://127.0.0.1:9099
	ReplicationFactor  int
}
