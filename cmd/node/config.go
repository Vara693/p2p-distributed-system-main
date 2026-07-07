package main

import (
	"flag"
)

// ServeConfig is populated from flags for `node serve`.
type ServeConfig struct {
	DataDir            string
	GRPCAddr           string
	HTTPAddr           string
	AdvertiseGRPC      string
	AdvertiseHTTP      string
	BootstrapURL       string
	ReplicationFactor  int
}

func ParseServeFlags(args []string) (ServeConfig, error) {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	cfg := ServeConfig{}
	fs.StringVar(&cfg.DataDir, "data", "storage/node1", "per-node data directory")
	fs.StringVar(&cfg.GRPCAddr, "grpc", "127.0.0.1:50051", "gRPC listen address")
	fs.StringVar(&cfg.HTTPAddr, "http", "127.0.0.1:8080", "HTTP API listen address")
	fs.StringVar(&cfg.AdvertiseGRPC, "advertise-grpc", "", "gRPC advertise address for external peers (e.g. ngrok tcp)")
	fs.StringVar(&cfg.AdvertiseHTTP, "advertise-http", "", "HTTP API advertise address (e.g. ngrok http)")
	fs.StringVar(&cfg.BootstrapURL, "bootstrap", "", "bootstrap HTTP base URL, e.g. http://127.0.0.1:9099")
	fs.IntVar(&cfg.ReplicationFactor, "replication", 3, "number of remote replicas per block (best-effort)")
	if err := fs.Parse(args); err != nil {
		return cfg, err
	}
	return cfg, nil
}
