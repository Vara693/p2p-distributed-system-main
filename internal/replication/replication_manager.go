package replication

import (
	"context"
	"fmt"
	"time"

	nodev1 "edi_sem2/internal/gen"
	"edi_sem2/internal/network"
	"edi_sem2/internal/storage"
)

// Manager pushes block bytes to remote peers via gRPC StoreBlock.
type Manager struct {
	Pool *network.Pool
}

func (m *Manager) StoreOnPeer(ctx context.Context, grpcAddr string, c storage.CID, data []byte) error {
	if m == nil || m.Pool == nil {
		return fmt.Errorf("replication manager not configured")
	}
	return network.WithTimeout(ctx, 15*time.Second, func(cctx context.Context) error {
		cli, err := network.NodeClient(cctx, m.Pool, grpcAddr)
		if err != nil {
			return err
		}
		resp, err := cli.StoreBlock(cctx, &nodev1.StoreBlockRequest{Cid: string(c), Data: data})
		if err != nil {
			return err
		}
		if resp == nil || !resp.Ok {
			if resp != nil && resp.ErrorMessage != "" {
				return fmt.Errorf("store_block: %s", resp.ErrorMessage)
			}
			return fmt.Errorf("store_block failed")
		}
		return nil
	})
}

// ReplicateToAddrs sends the same block to each gRPC address (best-effort: collects last error).
func (m *Manager) ReplicateToAddrs(ctx context.Context, addrs []string, c storage.CID, data []byte) error {
	var lastErr error
	for _, a := range addrs {
		if err := m.StoreOnPeer(ctx, a, c, data); err != nil {
			lastErr = err
		}
	}
	return lastErr
}
