// connect.go
package core

import (
	"context"
	"log"
	"time"

	"github.com/baoswarm/baobun/pkg/protocol"
)

func (c *Client) ConnectPeer(swarm *Swarm, peerKey protocol.NodeKey) {
	if c.IsPaused(swarm.InfoHash) {
		return
	}

	// Use the enhanced ConnectPeer with timeout
	_, err := c.Sessions.ConnectPeer(swarm, peerKey, 10*time.Second)
	if err != nil {
		log.Printf("connect peer failed (%s): %v", peerKey, err)
		return
	}

	log.Printf("Successfully connected to peer %s for swarm %s", peerKey, swarm.InfoHash)
}

func (c *Client) AnnounceSwarm(
	ctx context.Context,
	ih protocol.InfoHash,
	event protocol.AnnounceEvent,
) {
	swarm, ok := c.Swarms[ih]
	if !ok {
		log.Printf("swarm not found: %s", ih)
		return
	}
	if c.IsPaused(ih) {
		return
	}

	req := protocol.AnnounceRequest{
		InfoHash:   ih,
		Event:      event,
		Uploaded:   swarm.Uploaded,
		Downloaded: swarm.Downloaded,
		Left:       swarm.CalcLeft(),
		Timestamp:  uint64(time.Now().Unix()),
	}

	// Track successful connections for logging
	successfulConnections := 0

	for _, tracker := range swarm.File.Trackers {
		resp, err := c.Transport.Announce(ctx, tracker, req)
		if err != nil {
			log.Printf("announce failed (%s): %v", tracker, err)
			continue
		}

		for _, peer := range resp.Peers {
			if peer.NodeKey == protocol.NodeKey(c.NodeKey) {
				continue
			}

			// Check if peer already exists
			swarm.mu.RLock()
			_, exists := swarm.Peers[peer.NodeKey]
			swarm.mu.RUnlock()

			if exists {
				continue
			}

			// Connect in background with proper synchronization
			go func(p protocol.NodeKey) {
				c.ConnectPeer(swarm, p)
				successfulConnections++
			}(peer.NodeKey)
		}

		log.Printf(
			"announced to %s → %d new peers",
			tracker,
			len(resp.Peers),
		)
	}

	// Log overall connection success
	log.Printf("Initiated connections to %d peers for swarm %s", successfulConnections, ih)
}

func (c *Client) ReannounceAllSwarms(
	ctx context.Context,
) {
	for _, swarm := range c.Swarms {
		if c.IsPaused(swarm.InfoHash) {
			continue
		}

		req := protocol.AnnounceRequest{
			InfoHash:   swarm.InfoHash,
			Event:      "",
			Uploaded:   swarm.Uploaded,
			Downloaded: swarm.Downloaded,
			Left:       swarm.CalcLeft(),
			Timestamp:  uint64(time.Now().Unix()),
		}

		for _, tracker := range swarm.File.Trackers {
			resp, err := c.Transport.Announce(ctx, tracker, req)
			if err != nil {
				log.Printf("announce failed (%s): %v", tracker, err)
				continue
			}
			for _, peer := range resp.Peers {
				if peer.NodeKey == protocol.NodeKey(c.NodeKey) {
					continue
				}

				// Check if peer already exists
				swarm.mu.RLock()
				ph, exists := swarm.Peers[peer.NodeKey]
				swarm.mu.RUnlock()

				if exists && ph.state != protocol.StateClosed {
					continue
				}

				// Connect in background with proper synchronization
				go func(p protocol.NodeKey) {
					c.ConnectPeer(swarm, p)
				}(peer.NodeKey)
			}

			log.Printf(
				"announced to %s → %d new peers",
				tracker,
				len(resp.Peers),
			)
		}
	}
}
