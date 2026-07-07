package dht

import (
	"sync"

	nodev1 "edi_sem2/internal/gen"
)

// ProviderRecords maps content CID (hex) to providers (peer endpoints).
type ProviderRecords struct {
	mu sync.RWMutex
	// cid -> peerID -> endpoint
	m map[string]map[string]*nodev1.PeerEndpoint
}

func NewProviderRecords() *ProviderRecords {
	return &ProviderRecords{m: make(map[string]map[string]*nodev1.PeerEndpoint)}
}

func (p *ProviderRecords) Add(cidHex string, ep *nodev1.PeerEndpoint) {
	if p == nil || ep == nil || ep.PeerId == "" {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.m[cidHex] == nil {
		p.m[cidHex] = make(map[string]*nodev1.PeerEndpoint)
	}
	p.m[cidHex][ep.PeerId] = ep
}

func (p *ProviderRecords) Get(cidHex string) []*nodev1.PeerEndpoint {
	p.mu.RLock()
	defer p.mu.RUnlock()
	mp := p.m[cidHex]
	if len(mp) == 0 {
		return nil
	}
	out := make([]*nodev1.PeerEndpoint, 0, len(mp))
	for _, ep := range mp {
		out = append(out, ep)
	}
	return out
}

func (p *ProviderRecords) Merge(cidHex string, eps []*nodev1.PeerEndpoint) {
	for _, ep := range eps {
		p.Add(cidHex, ep)
	}
}

func (p *ProviderRecords) RemovePeer(peerID string) {
	if p == nil || peerID == "" {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	for cidHex, mp := range p.m {
		delete(mp, peerID)
		if len(mp) == 0 {
			delete(p.m, cidHex)
		}
	}
}
