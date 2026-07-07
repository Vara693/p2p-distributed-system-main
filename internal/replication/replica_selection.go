package replication

import (
	"edi_sem2/internal/dht"
	"edi_sem2/internal/peer"
)

// SelectReplicaPeerIDs picks up to k active peers closest to cidHex (XOR), excluding self.
func SelectReplicaPeerIDs(rt *dht.RoutingTable, cidHex string, k int, selfID string) ([]string, error) {
	key, err := dht.KeyFromHex(cidHex)
	if err != nil {
		return nil, err
	}
	st := map[peer.State]bool{peer.StateActive: true}
	return rt.ClosestPeerIDs(key, k, selfID, st), nil
}
