package dht

import (
	"sort"

	"edi_sem2/internal/peer"
)

// RoutingTable selects peers by XOR distance to a 32-byte target key.
type RoutingTable struct {
	reg *peer.Registry
}

func NewRoutingTable(reg *peer.Registry) *RoutingTable {
	return &RoutingTable{reg: reg}
}

// ClosestPeerIDs returns up to k peer IDs closest to targetKey, excluding excludeID.
// If allowedStates is non-nil, only peers whose state is true in the map are eligible.
func (t *RoutingTable) ClosestPeerIDs(targetKey []byte, k int, excludeID string, allowedStates map[peer.State]bool) []string {
	if t == nil || t.reg == nil || k <= 0 {
		return nil
	}
	var candidates []string
	for _, rec := range t.reg.Snapshot() {
		if rec.ID == excludeID {
			continue
		}
		if allowedStates != nil && !allowedStates[rec.State] {
			continue
		}
		if _, err := KeyFromHex(rec.ID); err != nil {
			continue
		}
		candidates = append(candidates, rec.ID)
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		ki, _ := KeyFromHex(candidates[i])
		kj, _ := KeyFromHex(candidates[j])
		return Closer(targetKey, ki, kj)
	})
	if len(candidates) > k {
		candidates = candidates[:k]
	}
	return candidates
}
