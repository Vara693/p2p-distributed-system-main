package network

import (
	"context"
	"sync"

	"google.golang.org/grpc"
)

// Pool reuses gRPC client connections keyed by target address ("host:port").
type Pool struct {
	mu    sync.Mutex
	conns map[string]*grpc.ClientConn
}

func NewPool() *Pool {
	return &Pool{conns: make(map[string]*grpc.ClientConn)}
}

func (p *Pool) Conn(ctx context.Context, addr string) (*grpc.ClientConn, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if c, ok := p.conns[addr]; ok {
		return c, nil
	}
	c, err := DialPeer(ctx, addr)
	if err != nil {
		return nil, err
	}
	p.conns[addr] = c
	return c, nil
}

func (p *Pool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	var firstErr error
	for addr, c := range p.conns {
		if err := c.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		delete(p.conns, addr)
	}
	return firstErr
}
