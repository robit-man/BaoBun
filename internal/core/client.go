package core

import (
	"context"

	"github.com/baoswarm/baobun/pkg/protocol"
)

/*
Client represents a running node.
It owns swarms and tracker communication,
but peer traffic flows through SessionManager.
*/
type Client struct {
	NodeKey   string // NKN public key
	Transport TrackerTransport
	Sessions  *SessionManager

	Swarms map[protocol.InfoHash]*Swarm
}

type TrackerTransport interface {
	Announce(
		ctx context.Context,
		trackerKey string,
		req protocol.AnnounceRequest,
	) (protocol.AnnounceResponse, error)

	Close()
}

func NewClient(
	nodeKey string,
	transport TrackerTransport,
	sessions *SessionManager,
) *Client {

	return &Client{
		NodeKey:   nodeKey,
		Transport: transport,
		Sessions:  sessions,
		Swarms:    make(map[protocol.InfoHash]*Swarm),
	}
}
