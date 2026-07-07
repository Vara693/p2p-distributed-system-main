package coordinator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	nodev1 "edi_sem2/internal/gen"
	"edi_sem2/internal/network"
	"edi_sem2/internal/peer"
)

// BootstrapPeer is the JSON shape exchanged with the bootstrap HTTP service.
type BootstrapPeer struct {
	PeerID   string `json:"peer_id"`
	GrpcAddr string `json:"grpc_addr"`
	HttpAddr string `json:"http_addr"`
}

func (h *PeerHost) registerBootstrap() error {
	if h.Bootstrap == "" {
		return nil
	}
	body, err := json.Marshal(BootstrapPeer{
		PeerID:   h.SelfID,
		GrpcAddr: h.GrpcAddr,
		HttpAddr: h.HTTPAddr,
	})
	if err != nil {
		return err
	}
	u := strings.TrimRight(h.Bootstrap, "/") + "/v1/register"
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("register: status %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

func (h *PeerHost) mergeBootstrapPeers(ctx context.Context) error {
	if h.Bootstrap == "" {
		return nil
	}
	u := strings.TrimRight(h.Bootstrap, "/") + "/v1/peers"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("peers: status %d: %s", resp.StatusCode, string(b))
	}
	var list []BootstrapPeer
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return err
	}
	for _, bp := range list {
		if bp.PeerID == "" || bp.PeerID == h.SelfID {
			continue
		}
		rec := &peer.Record{
			ID:        bp.PeerID,
			GrpcAddr:  bp.GrpcAddr,
			HttpAddr:  bp.HttpAddr,
			State:     peer.StateActive,
			LastSeen:  time.Now(),
		}
		if !h.reg.Upsert(rec) {
			continue
		}
		if bp.GrpcAddr == "" {
			continue
		}
		_ = network.WithTimeout(ctx, 8*time.Second, func(cctx context.Context) error {
			cli, err := network.NodeClient(cctx, h.pool, bp.GrpcAddr)
			if err != nil {
				return err
			}
			joinResp, err := cli.JoinNetwork(cctx, &nodev1.JoinNetworkRequest{Self: h.SelfEndpoint()})
			if err != nil || joinResp == nil || !joinResp.Accepted {
				return fmt.Errorf("join rejected")
			}
			for _, ep := range joinResp.KnownPeers {
				if ep == nil || ep.PeerId == "" {
					continue
				}
				h.reg.Upsert(&peer.Record{
					ID:        ep.PeerId,
					GrpcAddr:  ep.GrpcAddr,
					HttpAddr:  ep.HttpAddr,
					State:     peer.StateActive,
					LastSeen:  time.Now(),
				})
			}
			return nil
		})
	}
	return nil
}
