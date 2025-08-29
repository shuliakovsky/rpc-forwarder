package bootstrap

import "github.com/shuliakovsky/rpc-forwarder/pkg/peers"

type AnnounceRequest struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	InternalAddr string `json:"internal_addr"`
	Timestamp    int64  `json:"timestamp"`
	Signature    string `json:"signature"`
}

type AnnounceResponse struct {
	Peers []peers.Peer `json:"peers"`
}
