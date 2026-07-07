package api

import (
	"context"
	"io"

	"edi_sem2/internal/dht"
	"edi_sem2/internal/network"
	"edi_sem2/internal/peer"
	"edi_sem2/internal/storage"
)

// Host is the minimal surface the HTTP API needs from a running peer node.
type Host interface {
	AddFile(ctx context.Context, path string) (storage.CID, error)
	GetFile(ctx context.Context, root storage.CID, w io.Writer) error
	InspectCID(c storage.CID) (string, error)
	Registry() *peer.Registry
	Providers() *dht.ProviderRecords
	Pool() *network.Pool
	// DialGRPC is the gRPC address other peers use to reach this node (host:port).
	DialGRPC() string
	ListenHTTP() string
	PeerID() string
	UpdateGlobalCatalog(ctx context.Context, name, cid string, size int64, ext string) (storage.CID, error)
	LocalCatalogCid() string
}
