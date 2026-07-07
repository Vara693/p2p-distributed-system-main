package coordinator

import (
	"context"
	"time"

	nodev1 "edi_sem2/internal/gen"
	"edi_sem2/internal/peer"
	"edi_sem2/internal/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type grpcService struct {
	nodev1.UnimplementedNodeServer
	h *PeerHost
}

// NewGRPCService returns a gRPC service implementation backed by h.
func NewGRPCService(h *PeerHost) nodev1.NodeServer {
	return &grpcService{h: h}
}

func (s *grpcService) Ping(ctx context.Context, req *nodev1.PingRequest) (*nodev1.PingResponse, error) {
	_ = ctx
	if req != nil && req.PeerId != "" {
		s.h.reg.Touch(req.PeerId)
	}
	return &nodev1.PingResponse{Ok: true}, nil
}

func (s *grpcService) StoreBlock(ctx context.Context, req *nodev1.StoreBlockRequest) (*nodev1.StoreBlockResponse, error) {
	_ = ctx
	if req == nil || req.Cid == "" {
		return &nodev1.StoreBlockResponse{Ok: false, ErrorMessage: "missing cid"}, nil
	}
	c := storage.CID(req.Cid)
	if err := c.Validate(); err != nil {
		return &nodev1.StoreBlockResponse{Ok: false, ErrorMessage: err.Error()}, nil
	}
	if _, err := s.h.store.Put(req.Data); err != nil {
		return &nodev1.StoreBlockResponse{Ok: false, ErrorMessage: err.Error()}, nil
	}
	s.h.providers.Add(req.Cid, s.h.SelfEndpoint())
	return &nodev1.StoreBlockResponse{Ok: true}, nil
}

func (s *grpcService) GetBlock(ctx context.Context, req *nodev1.GetBlockRequest) (*nodev1.GetBlockResponse, error) {
	_ = ctx
	if req == nil || req.Cid == "" {
		return nil, status.Error(codes.InvalidArgument, "missing cid")
	}
	c := storage.CID(req.Cid)
	b, err := s.h.store.Get(c)
	if err != nil {
		return &nodev1.GetBlockResponse{ErrorMessage: err.Error()}, nil
	}
	return &nodev1.GetBlockResponse{Data: b}, nil
}

func (s *grpcService) FindProvider(ctx context.Context, req *nodev1.FindProviderRequest) (*nodev1.FindProviderResponse, error) {
	_ = ctx
	if req == nil || req.Cid == "" {
		return &nodev1.FindProviderResponse{}, nil
	}
	return &nodev1.FindProviderResponse{Providers: s.h.providers.Get(req.Cid)}, nil
}

func (s *grpcService) JoinNetwork(ctx context.Context, req *nodev1.JoinNetworkRequest) (*nodev1.JoinNetworkResponse, error) {
	_ = ctx
	if req == nil || req.Self == nil {
		return &nodev1.JoinNetworkResponse{Accepted: false, ErrorMessage: "missing self"}, nil
	}
	rec := &peer.Record{
		ID:        req.Self.PeerId,
		GrpcAddr:  req.Self.GrpcAddr,
		HttpAddr:  req.Self.HttpAddr,
		State:     peer.StateActive,
		LastSeen:  time.Now(),
	}
	if !s.h.reg.Upsert(rec) {
		return &nodev1.JoinNetworkResponse{Accepted: false, ErrorMessage: "network full"}, nil
	}
	var eps []*nodev1.PeerEndpoint
	for _, p := range s.h.reg.Snapshot() {
		eps = append(eps, &nodev1.PeerEndpoint{
			PeerId:   p.ID,
			GrpcAddr: p.GrpcAddr,
			HttpAddr: p.HttpAddr,
		})
	}
	return &nodev1.JoinNetworkResponse{Accepted: true, KnownPeers: eps}, nil
}

func (s *grpcService) LeaveNetwork(ctx context.Context, req *nodev1.LeaveNetworkRequest) (*nodev1.LeaveNetworkResponse, error) {
	_ = ctx
	if req == nil || req.PeerId == "" {
		return &nodev1.LeaveNetworkResponse{Ok: false}, nil
	}
	s.h.rtm.Leave(req.PeerId)
	return &nodev1.LeaveNetworkResponse{Ok: true}, nil
}

func (s *grpcService) Heartbeat(ctx context.Context, req *nodev1.HeartbeatRequest) (*nodev1.HeartbeatResponse, error) {
	_ = ctx
	if req != nil && req.PeerId != "" {
		s.h.reg.Touch(req.PeerId)
	}
	return &nodev1.HeartbeatResponse{Ok: true}, nil
}

func (s *grpcService) AddProvider(ctx context.Context, req *nodev1.AddProviderRequest) (*nodev1.AddProviderResponse, error) {
	_ = ctx
	if req == nil || req.Cid == "" || req.Provider == nil {
		return &nodev1.AddProviderResponse{Ok: false}, nil
	}
	s.h.providers.Add(req.Cid, req.Provider)
	return &nodev1.AddProviderResponse{Ok: true}, nil
}
