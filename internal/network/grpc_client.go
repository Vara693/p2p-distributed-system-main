package network

import (
	"context"
	"time"

	nodev1 "edi_sem2/internal/gen"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NodeClient returns a NodeClient for addr using the pool.
func NodeClient(ctx context.Context, pool *Pool, addr string) (nodev1.NodeClient, error) {
	c, err := pool.Conn(ctx, addr)
	if err != nil {
		return nil, err
	}
	return nodev1.NewNodeClient(c), nil
}

// WithTimeout runs fn with a deadline on ctx.
func WithTimeout(parent context.Context, d time.Duration, fn func(context.Context) error) error {
	ctx, cancel := context.WithTimeout(parent, d)
	defer cancel()
	return fn(ctx)
}

// IsUnavailable returns true for common gRPC transport failures.
func IsUnavailable(err error) bool {
	if err == nil {
		return false
	}
	s, ok := status.FromError(err)
	if !ok {
		return true
	}
	switch s.Code() {
	case codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted:
		return true
	default:
		return false
	}
}
