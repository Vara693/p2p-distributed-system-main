package replication

import (
	"context"
	"fmt"
	"time"

	nodev1 "edi_sem2/internal/gen"
	"edi_sem2/internal/network"
	"edi_sem2/internal/storage"
)

// FetchBlock tries each provider gRPC address until GetBlock succeeds.
func FetchBlock(ctx context.Context, pool *network.Pool, cid storage.CID, providers []*nodev1.PeerEndpoint) ([]byte, error) {
	var lastErr error
	for _, ep := range providers {
		if ep == nil || ep.GrpcAddr == "" {
			continue
		}
		var data []byte
		err := network.WithTimeout(ctx, 15*time.Second, func(cctx context.Context) error {
			cli, err := network.NodeClient(cctx, pool, ep.GrpcAddr)
			if err != nil {
				return err
			}
			resp, err := cli.GetBlock(cctx, &nodev1.GetBlockRequest{Cid: string(cid)})
			if err != nil {
				return err
			}
			if resp == nil || len(resp.Data) == 0 {
				if resp != nil && resp.ErrorMessage != "" {
					return fmt.Errorf("%s", resp.ErrorMessage)
				}
				return fmt.Errorf("empty block")
			}
			data = resp.Data
			return nil
		})
		if err == nil {
			if err := storage.VerifyCID(cid, data); err != nil {
				lastErr = err
				continue
			}
			return data, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no providers for %s", cid)
	}
	return nil, lastErr
}
