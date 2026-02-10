package core

import (
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/baoswarm/baobun/pkg/protocol"
)

type TransferUnitManager struct {
	swarm             *Swarm
	transferUnits     []*TransferUnit
	transferUnitCount uint64

	// Active requests tracking
	activeRequests map[uint64]*transferUnitRequest // transferUnit index -> request
	peerRequests   map[protocol.NodeKey][]uint64   // peer -> slice of transferUnit indices

	transferUnitCompleteChan chan transferUnitCompleteEvent

	mu sync.RWMutex
}

func NewTransferUnitManager(swarm *Swarm, numTransferUnits uint64) *TransferUnitManager {
	pm := &TransferUnitManager{
		swarm:                    swarm,
		transferUnits:            make([]*TransferUnit, numTransferUnits),
		transferUnitCount:        numTransferUnits,
		activeRequests:           make(map[uint64]*transferUnitRequest),
		peerRequests:             make(map[protocol.NodeKey][]uint64),
		transferUnitCompleteChan: make(chan transferUnitCompleteEvent, 100),
	}

	for i := uint64(0); i < numTransferUnits; i++ {
		state := TransferUnitStateMissing
		if swarm.FileIO.haveUnits.Has(i) {
			state = TransferUnitStateComplete
		}

		pm.transferUnits[i] = &TransferUnit{
			Index: i,
			State: state,
		}
	}

	go pm.run()
	return pm
}

func (pm *TransferUnitManager) run() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case event := <-pm.transferUnitCompleteChan:
			pm.handleTransferUnitComplete(event.transferUnit)

		case <-ticker.C:
			pm.checkTimeouts()
			pm.scheduleDownloads()
		}
	}
}

func (pm *TransferUnitManager) MarkTransferUnitComplete(index uint64) {
	pm.transferUnitCompleteChan <- transferUnitCompleteEvent{
		transferUnit: index,
	}
}

func (pm *TransferUnitManager) handleTransferUnitComplete(index uint64) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if index >= pm.transferUnitCount {
		return
	}

	unit := pm.transferUnits[index]
	unit.State = TransferUnitStateComplete

	if req, exists := pm.activeRequests[index]; exists {
		pm.cleanupRequest(index, req.From)
	}

	log.Printf("TransferUnit %d download complete", index)
}

func (pm *TransferUnitManager) checkTimeouts() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	now := time.Now()
	timeout := 30 * time.Second

	for idx, req := range pm.activeRequests {
		if now.Sub(req.SentAt) > timeout {
			log.Printf("Request for transferUnit %d from %s timed out", idx, req.From)

			pm.cleanupRequest(idx, req.From)

			unit := pm.transferUnits[idx]
			if unit.State == TransferUnitStateDownloading {
				unit.State = TransferUnitStateMissing
			}

			req.Attempts++
		}
	}
}

func (pm *TransferUnitManager) cleanupRequest(index uint64, peer protocol.NodeKey) {
	delete(pm.activeRequests, index)

	if reqs, ok := pm.peerRequests[peer]; ok {
		filtered := reqs[:0]
		for _, r := range reqs {
			if r != index {
				filtered = append(filtered, r)
			}
		}
		if len(filtered) == 0 {
			delete(pm.peerRequests, peer)
		} else {
			pm.peerRequests[peer] = filtered
		}
	}
}

func (pm *TransferUnitManager) scheduleDownloads() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	const (
		maxActiveRequests  = 64
		maxRequestsPerPeer = 8
	)

	if len(pm.activeRequests) >= maxActiveRequests {
		return
	}

	//TODO: rarest first mode
	//NOT YET IMPLEMENTED

	//TODO: shuffle mode
	var candidates []uint64
	for unitIdx := uint64(0); unitIdx < pm.transferUnitCount; unitIdx++ {
		unit := pm.transferUnits[unitIdx]

		if unit.State != TransferUnitStateMissing {
			continue
		}

		if _, active := pm.activeRequests[unitIdx]; active {
			continue
		}

		candidates = append(candidates, unitIdx)
	}
	rand.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})
	for _, unitIdx := range candidates {
		peer := pm.selectPeerForTransferUnit(unitIdx, maxRequestsPerPeer)
		if peer == "" {
			continue
		}

		if pm.sendTransferUnitRequest(unitIdx, peer) {
			log.Printf("Requested transferUnit %d from %s", unitIdx, peer)
		}

		if len(pm.activeRequests) >= maxActiveRequests {
			return
		}
	}

	//TODO: sequential mode
	// for unitIdx := uint64(0); unitIdx < pm.transferUnitCount; unitIdx++ {
	// 	unit := pm.transferUnits[unitIdx]

	// 	if unit.State != TransferUnitStateMissing {
	// 		continue
	// 	}

	// 	if _, active := pm.activeRequests[unitIdx]; active {
	// 		continue
	// 	}

	// 	peer := pm.selectPeerForTransferUnit(unitIdx, maxRequestsPerPeer)
	// 	if peer == "" {
	// 		continue
	// 	}

	// 	if pm.sendTransferUnitRequest(unitIdx, peer) {
	// 		log.Printf("Requested transferUnit %d from %s", unitIdx, peer)
	// 	}

	// 	if len(pm.activeRequests) >= maxActiveRequests {
	// 		return
	// 	}
	// }
}

func (pm *TransferUnitManager) selectPeerForTransferUnit(unit uint64, maxPerPeer int) protocol.NodeKey {
	pm.swarm.mu.RLock()
	defer pm.swarm.mu.RUnlock()

	var candidates []protocol.NodeKey
	minLoad := int(^uint(0) >> 1)
	var best protocol.NodeKey

	for peer, handler := range pm.swarm.Peers {
		if handler.GetState() != protocol.StateConnected {
			continue
		}

		if handler.Bitfield.bits == nil {
			continue
		}

		if !handler.Bitfield.Has(unit) {
			continue
		}

		load := len(pm.peerRequests[peer])
		if load >= maxPerPeer {
			continue
		}

		candidates = append(candidates, peer)

		if load < minLoad {
			minLoad = load
			best = peer
		}
	}

	if best != "" {
		return best
	}

	if len(candidates) > 0 {
		return candidates[rand.Intn(len(candidates))]
	}

	return ""
}

func (pm *TransferUnitManager) sendTransferUnitRequest(index uint64, peer protocol.NodeKey) bool {
	pm.swarm.mu.RLock()
	handler, exists := pm.swarm.Peers[peer]
	pm.swarm.mu.RUnlock()

	if !exists {
		return false
	}

	if err := handler.SendTransferUnitRequest(index); err != nil {
		log.Printf("Failed to send request for transferUnit %d to %s: %v", index, peer, err)
		return false
	}

	pm.activeRequests[index] = &transferUnitRequest{
		Index:    index,
		From:     peer,
		SentAt:   time.Now(),
		Attempts: 1,
		Timeout:  30 * time.Second,
	}

	pm.peerRequests[peer] = append(pm.peerRequests[peer], index)
	pm.transferUnits[index].State = TransferUnitStateDownloading

	return true
}
