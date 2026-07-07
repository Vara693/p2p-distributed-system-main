package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"edi_sem2/internal/api"
	"edi_sem2/internal/cli"
	"edi_sem2/internal/coordinator"
	nodev1 "edi_sem2/internal/gen"
	"google.golang.org/grpc"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "serve" {
		if err := runServe(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		return
	}
	ctx := context.Background()
	if err := cli.Run(ctx, os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func runServe(args []string) error {
	cfg, err := ParseServeFlags(args)
	if err != nil {
		return err
	}
	if cfg.DataDir == "" {
		return errors.New("-data required")
	}

	advGrpc := cfg.GRPCAddr
	if cfg.AdvertiseGRPC != "" {
		advGrpc = cfg.AdvertiseGRPC
	}
	advHttp := cfg.HTTPAddr
	if cfg.AdvertiseHTTP != "" {
		advHttp = cfg.AdvertiseHTTP
	}

	host, err := coordinator.NewPeerHost(coordinator.Config{
		DataDir:           cfg.DataDir,
		GrpcListenAddr:    advGrpc,
		HTTPListenAddr:    advHttp,
		BootstrapURL:      cfg.BootstrapURL,
		ReplicationFactor: cfg.ReplicationFactor,
	})
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	host.RunLoops(ctx)

	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		return fmt.Errorf("grpc listen: %w", err)
	}
	grpcSrv := grpc.NewServer()
	nodev1.RegisterNodeServer(grpcSrv, coordinator.NewGRPCService(host))
	go func() {
		if err := grpcSrv.Serve(lis); err != nil {
			fmt.Fprintln(os.Stderr, "grpc:", err)
		}
	}()

	mux := http.NewServeMux()
	api.RegisterRoutes(mux, host)
	httpSrv := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: api.CORSMiddleware(mux),
	}
	go func() {
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintln(os.Stderr, "http:", err)
		}
	}()

	<-ctx.Done()
	shCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(shCtx)
	grpcSrv.GracefulStop()
	_ = host.Close()
	return nil
}
