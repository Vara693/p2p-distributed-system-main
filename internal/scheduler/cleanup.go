package scheduler

import (
	"context"
	"time"

	"edi_sem2/internal/dht"
	"edi_sem2/internal/peer"
)

// RunCleanup periodically sweeps the peer registry for long-term inactive peers,
// transitions them to StateLeft, and purges their stale provider records.
func RunCleanup(ctx context.Context, reg *peer.Registry, provs *dht.ProviderRecords) {
	t := time.NewTicker(10 * time.Second)
	defer t.Stop()

	staleTimeout := 60 * time.Second

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if reg == nil || provs == nil {
				continue
			}
			now := time.Now()
			for _, p := range reg.Snapshot() {
				if p.ID == reg.SelfID() {
					continue
				}
				if p.State == peer.StateInactive && now.Sub(p.LastSeen) > staleTimeout {
					reg.MarkState(p.ID, peer.StateLeft)
					provs.RemovePeer(p.ID)
				}
			}
		}
	}
}
