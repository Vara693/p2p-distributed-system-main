package dht

import (
	"context"
	"sync"
	"time"

	nodev1 "edi_sem2/internal/gen"
	"edi_sem2/internal/network"
	"edi_sem2/internal/peer"
	"edi_sem2/internal/utils"
)

// LookupProviders merges local records with FindProvider RPC fan-out to active peers.
func LookupProviders(
	ctx context.Context,
	pool *network.Pool,
	local *ProviderRecords,
	selfGrpc string,
	peers []*peer.Record,
	cidHex string,
) []*nodev1.PeerEndpoint {
	seen := make(map[string]struct{})
	var out []*nodev1.PeerEndpoint
	add := func(ep *nodev1.PeerEndpoint) {
		if ep == nil || ep.PeerId == "" {
			return
		}
		if _, ok := seen[ep.PeerId]; ok {
			return
		}
		seen[ep.PeerId] = struct{}{}
		out = append(out, ep)
	}
	for _, ep := range local.Get(cidHex) {
		add(ep)
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	timeout := time.Duration(utils.DefaultNet().GRPCTimeoutSec) * time.Second
	for _, pr := range peers {
		if pr.State != peer.StateActive {
			continue
		}
		if pr.GrpcAddr == "" || pr.GrpcAddr == selfGrpc {
			continue
		}
		wg.Add(1)
		go func(addr string) {
			defer wg.Done()
			_ = network.WithTimeout(ctx, timeout, func(cctx context.Context) error {
				cli, err := network.NodeClient(cctx, pool, addr)
				if err != nil {
					return err
				}
				resp, err := cli.FindProvider(cctx, &nodev1.FindProviderRequest{Cid: cidHex})
				if err != nil || resp == nil {
					return err
				}
				mu.Lock()
				for _, p := range resp.Providers {
					add(p)
				}
				mu.Unlock()
				return nil
			})
		}(pr.GrpcAddr)
	}
	wg.Wait()
	return out
}
