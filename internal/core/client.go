package core

import (
	"context"
	"sync"

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

	pauseMu sync.RWMutex
	paused  map[protocol.InfoHash]bool
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
		paused:    make(map[protocol.InfoHash]bool),
	}
}

func (c *Client) IsPaused(ih protocol.InfoHash) bool {
	c.pauseMu.RLock()
	defer c.pauseMu.RUnlock()
	return c.paused[ih]
}

func (c *Client) PauseSwarm(ih protocol.InfoHash) bool {
	swarm, ok := c.Swarms[ih]
	if !ok {
		return false
	}

	c.pauseMu.Lock()
	c.paused[ih] = true
	c.pauseMu.Unlock()

	if swarm != nil {
		swarm.DisconnectAll(c.Sessions)
	}

	return true
}

func (c *Client) UnpauseSwarm(ih protocol.InfoHash) {
	c.pauseMu.Lock()
	delete(c.paused, ih)
	c.pauseMu.Unlock()
}

func (c *Client) RemoveSwarm(ih protocol.InfoHash) (*Swarm, bool) {
	swarm, ok := c.Swarms[ih]
	if !ok {
		return nil, false
	}

	delete(c.Swarms, ih)
	c.UnpauseSwarm(ih)

	return swarm, true
}
