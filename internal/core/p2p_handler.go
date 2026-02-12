package core

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/baoswarm/baobun/internal/debugs"
	"github.com/baoswarm/baobun/pkg/protocol"
)

type PeerHandler struct {
	Peer     protocol.NodeKey
	Swarm    *Swarm
	Session  *Session
	Bitfield Bitfield

	// Serializer for message encoding/decoding
	serializer Serializer

	// State management
	state             protocol.ConnectionState
	stateMu           sync.RWMutex
	handshakeReceived chan struct{}
	connected         chan struct{}

	// For handshake tracking
	ourHandshakeSent   bool
	theirHandshakeSeen bool

	// For synchronization
	mu sync.Mutex

	// Tracking bandwidth
	uploadedTotal   uint64
	uploadSamples   []bandwidthSample
	downloadedTotal uint64
	downloadSamples []bandwidthSample
}

type bandwidthSample struct {
	t time.Time
	n int
}

func BaoHandler(
	sm *SessionManager,
	swarm *Swarm,
	peer protocol.NodeKey,
	initiateHandshake bool,
	serializer Serializer,
) (*PeerHandler, error) {
	// Get or create session
	sess, err := sm.GetSession(peer)
	if err != nil {
		return nil, err
	}

	ph := &PeerHandler{
		Peer:              peer,
		Swarm:             swarm,
		Session:           sess,
		serializer:        serializer,
		state:             protocol.StateConnecting,
		handshakeReceived: make(chan struct{}),
		connected:         make(chan struct{}),
	}

	// Register handler before any message processing
	swarm.mu.Lock()
	swarm.Peers[peer] = ph
	swarm.mu.Unlock()

	return ph, nil
}

func (ph *PeerHandler) WaitForHandshake(timeout time.Duration) bool {
	select {
	case <-ph.handshakeReceived:
		return true
	case <-time.After(timeout):
		return false
	}
}

func (ph *PeerHandler) WaitForConnection(timeout time.Duration) bool {
	select {
	case <-ph.connected:
		return true
	case <-time.After(timeout):
		return false
	}
}

func (ph *PeerHandler) SetState(state protocol.ConnectionState) {
	ph.stateMu.Lock()
	oldState := ph.state
	ph.state = state
	ph.stateMu.Unlock()

	// Notify state changes
	if oldState < protocol.StateHandshaking && state >= protocol.StateHandshaking {
		select {
		case <-ph.handshakeReceived:
			// Already closed
		default:
			close(ph.handshakeReceived)
		}
	}
	if oldState < protocol.StateConnected && state >= protocol.StateConnected {
		select {
		case <-ph.connected:
			// Already closed
		default:
			close(ph.connected)
		}
	}
}

func (ph *PeerHandler) GetState() protocol.ConnectionState {
	ph.stateMu.RLock()
	defer ph.stateMu.RUnlock()
	return ph.state
}

func (ph *PeerHandler) Send(msg protocol.PeerMessage) error {
	ph.Session.writeMu.Lock()
	defer ph.Session.writeMu.Unlock()

	data, err := ph.serializer.MarshalPeerMessage(&msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// 1. Write length prefix (uint32, big-endian)
	if err := binary.Write(ph.Session.conn, binary.BigEndian, uint32(len(data))); err != nil {
		return fmt.Errorf("failed to write length: %w", err)
	}

	// 2. Write protobuf payload
	if _, err := ph.Session.conn.Write(data); err != nil {
		return fmt.Errorf("failed to write payload: %w", err)
	}

	return nil
}

func (ph *PeerHandler) SendHandshake(peerID protocol.NodeKey) error {
	ph.mu.Lock()
	if ph.ourHandshakeSent {
		ph.mu.Unlock()
		return nil // Already sent
	}
	ph.mu.Unlock()

	payload, err := ph.serializer.MarshalHandshakePayload(&protocol.HandshakePayload{
		InfoHash: ph.Swarm.InfoHash,
		PeerID:   string(peerID),
	})
	if err != nil {
		return fmt.Errorf("failed to marshal handshake: %w", err)
	}

	err = ph.Send(protocol.PeerMessage{
		InfoHash: ph.Swarm.InfoHash,
		Type:     protocol.MsgHandshake,
		Payload:  payload,
	})

	if err == nil {
		ph.mu.Lock()
		ph.ourHandshakeSent = true
		ph.mu.Unlock()

		if ph.GetState() == protocol.StateConnecting {
			ph.SetState(protocol.StateHandshaking)
		}
	}

	return err
}

func (ph *PeerHandler) SendBitfield(bits []byte) error {
	payload, err := ph.serializer.MarshalBitfieldPayload(&protocol.BitfieldPayload{Bits: bits})
	if err != nil {
		return fmt.Errorf("failed to marshal bitfield: %w", err)
	}

	return ph.Send(protocol.PeerMessage{
		InfoHash: ph.Swarm.InfoHash,
		Type:     protocol.MsgBitfield,
		Payload:  payload,
	})
}

// Enhance HandleMessage to handle transferUnit-related messages
func (ph *PeerHandler) HandleMessage(msg protocol.PeerMessage) {
	switch msg.Type {
	case protocol.MsgHandshake:
		ph.mu.Lock()
		ph.theirHandshakeSeen = true
		ph.mu.Unlock()

		if ph.ourHandshakeSent && ph.theirHandshakeSeen {
			ph.SetState(protocol.StateConnected)
		}

	case protocol.MsgBitfield:
		var bf protocol.BitfieldPayload
		if err := ph.serializer.UnmarshalBitfieldPayload(msg.Payload, &bf); err != nil {
			log.Printf("Failed to unmarshal bitfield from %s: %v", ph.Peer, err)
			return
		}
		ph.Bitfield = BitfieldFromBytes(bf.Bits)

		log.Printf("Received bitfield:")
		log.Println(ph.Bitfield.ToString(ph.Swarm.FileIO.unitCount))

		// Notify swarm about updated bitfield
		ph.Swarm.UpdatePeerBitfield(ph.Peer, ph.Bitfield)
		ph.Swarm.TransferUnitManager.scheduleDownloads()

	case protocol.MsgHave:
		var have protocol.HavePayload
		if err := ph.serializer.UnmarshalHavePayload(msg.Payload, &have); err != nil {
			log.Printf("Failed to unmarshal have from %s: %v", ph.Peer, err)
			return
		}

		if ph.Bitfield.bits == nil {
			//TODO: we should init the bitfield if its null, and when we eventually do receive the full initial bitfield state from the peer
			//we should then AND the initial bitfied state with the one initialized here so we ensure we have both the full init and all the additional HAVE msg bits
			log.Println("Read TODO: BITFIELD SHOULDNT BE NIL!")
		}

		// Update bitfield
		ph.Bitfield.Set(have.UnitIndex)

	case protocol.MsgRequest:
		var req protocol.TransferRequestPayload
		if err := ph.serializer.UnmarshalTransferRequestPayload(msg.Payload, &req); err != nil {
			log.Printf("Failed to unmarshal request from %s: %v", ph.Peer, err)
			return
		}

		debugs.NumTransferRequestReceived++
		debugs.LogNums()

		// Handle incoming transferUnit request (if we have the transferUnit)
		ph.handleIncomingRequest(req.UnitIndex)

	case protocol.MsgTransfer:
		var transferUnit protocol.TransferPayload
		if err := ph.serializer.UnmarshalTransferPayload(msg.Payload, &transferUnit); err != nil {
			log.Printf("Failed to unmarshal transferUnit from %s: %v", ph.Peer, err)
			return
		}

		// Verify the proof if included
		if transferUnit.Proof != nil {
			baoProof := transferUnit.Proof

			rootHash, _ := hex.DecodeString(ph.Swarm.File.RootHash)

			err := VerifyProof(transferUnit.Data, baoProof, protocol.Hash(rootHash), int64(ph.Swarm.File.Length))
			if err != nil {
				log.Printf("proof verification failed for unit %d: %s",
					transferUnit.UnitIndex, err)

				return
			}

			// Proof is valid - data is authentic
		} else {
			// Handle case where proof is missing (backward compatibility or error)
			log.Printf("missing proof for unit %d", transferUnit.UnitIndex)
			return

		}

		ph.recordDownload(len(transferUnit.Data))

		writeErr := ph.Swarm.FileIO.WriteTransferUnit(transferUnit.UnitIndex, transferUnit.Data)
		if writeErr == nil {
			if err := ph.Swarm.SaveProof(transferUnit.UnitIndex, transferUnit.Proof); err != nil {
				log.Printf("failed to persist proof for unit %d: %v", transferUnit.UnitIndex, err)
			}

			// Notify swarm about completed transferUnit
			ph.Swarm.MarkTransferUnitComplete(transferUnit.UnitIndex, transferUnit.Data)

			// Clean up our request tracking
			ph.Swarm.TransferUnitManager.transferUnitCompleteChan <- transferUnitCompleteEvent{
				transferUnit: transferUnit.UnitIndex,
				data:         transferUnit.Data,
			}

			//TODO: Set readonly as soon as we fully downloaded the file
			//ph.Swarm.FileIO.SwitchToReadOnly()

		} else {
			fmt.Println("Failed to write transfer unit to disk: " + writeErr.Error())
		}
	}
}

func (ph *PeerHandler) handleIncomingRequest(transferUnitIndex uint64) {
	// Only serve units that we can prove.
	if !ph.Swarm.CanServeTransferUnit(transferUnitIndex) {
		// Don't have it
		return
	}

	transferUnitData, err := ph.Swarm.FileIO.ReadTransferUnit(transferUnitIndex)
	if err != nil {
		//TODO: panic here, and find out why this happens in the first place.
		//log.Panicf("Failed to read transferUnitdata for transferUnitIndex %d.", transferUnitIndex)

		log.Printf("INVESTIGATE!!!!: Failed to read transferUnitdata for transferUnitIndex %d.", transferUnitIndex)
	}

	// Send the transferUnit
	if err := ph.SendTransferUnit(transferUnitIndex, transferUnitData); err != nil {
		log.Printf("Failed to send transferUnit %d to %s: %v", transferUnitIndex, ph.Peer, err)
	} else {
		ph.Swarm.Uploaded += uint64(len(transferUnitData))
	}
}

func (ph *PeerHandler) Close(sm *SessionManager) {
	ph.SetState(protocol.StateClosed)

	// Remove from swarm
	ph.Swarm.mu.Lock()
	delete(ph.Swarm.Peers, ph.Peer)
	ph.Swarm.mu.Unlock()

	// Release session
	if sm != nil {
		sm.Release(ph.Peer)
	}

	log.Printf("Closed connection to peer %s", ph.Peer)
}

func (ph *PeerHandler) SendTransferUnitRequest(transferUnitIndex uint64) error {
	payload, err := ph.serializer.MarshalTransferRequestPayload(&protocol.TransferRequestPayload{
		UnitIndex: transferUnitIndex,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal transferUnit request: %w", err)
	}

	debugs.NumTransferRequestSend++
	debugs.LogNums()

	return ph.Send(protocol.PeerMessage{
		InfoHash: ph.Swarm.InfoHash,
		Type:     protocol.MsgRequest,
		Payload:  payload,
	})
}

func (ph *PeerHandler) SendHave(transferUnitIndex uint64) error {
	payload, err := ph.serializer.MarshalHavePayload(&protocol.HavePayload{
		UnitIndex: transferUnitIndex,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal have message: %w", err)
	}

	return ph.Send(protocol.PeerMessage{
		InfoHash: ph.Swarm.InfoHash,
		Type:     protocol.MsgHave,
		Payload:  payload,
	})
}

func (ph *PeerHandler) SendTransferUnit(transferUnitIndex uint64, data []byte) error {
	// Calculate the offset for this transfer unit
	offset := int64(transferUnitIndex) * 64 * 1024 // 64KB per unit
	length := int64(len(data))

	var proof *protocol.Proof
	// Check if we have any cached proof
	proof = ph.Swarm.GetProof(transferUnitIndex)

	if proof == nil {
		// Generate proof for this segment
		generatedProof, calculatedRoot, err := GenerateProofOnDisk(ph.Swarm.FileIO.file, offset, length)
		if err != nil {
			return fmt.Errorf("failed to generate proof: %w", err)
		}

		calculatedRootHashBytes := hex.EncodeToString(calculatedRoot[:])

		// Optional: Verify the root matches what we expect
		// This ensures we're generating proofs correctly
		if calculatedRootHashBytes != ph.Swarm.File.RootHash {
			return fmt.Errorf("root hash mismatch: file may have been modified")
		}

		proof = generatedProof
		if err := ph.Swarm.SaveProof(transferUnitIndex, generatedProof); err != nil {
			log.Printf("failed to persist generated proof for unit %d: %v", transferUnitIndex, err)
		}
	} else {
		log.Println("Used cached proof")
	}

	// Convert to protocol proof format
	// Create payload with proof
	payload, err := ph.serializer.MarshalTransferPayload(&protocol.TransferPayload{
		UnitIndex: transferUnitIndex,
		Data:      data,
		Proof:     proof,
	})

	if err != nil {
		return fmt.Errorf("failed to marshal transferUnit: %w", err)
	}

	ph.recordUpload(len(data))

	return ph.Send(protocol.PeerMessage{
		InfoHash: ph.Swarm.InfoHash,
		Type:     protocol.MsgTransfer,
		Payload:  payload,
	})
}

func (ph *PeerHandler) recordUpload(n int) {

	debugs.NumTransferResponseSend++
	debugs.LogNums()

	now := time.Now()

	ph.mu.Lock()
	defer ph.mu.Unlock()

	ph.uploadedTotal += uint64(n)
	ph.uploadSamples = append(ph.uploadSamples, bandwidthSample{
		t: now,
		n: n,
	})
}

func (ph *PeerHandler) UploadRate() uint32 {

	cutoff := time.Now().Add(-5 * time.Second)

	ph.mu.Lock()
	defer ph.mu.Unlock()

	var sum int
	var i int

	for _, s := range ph.uploadSamples {
		if s.t.After(cutoff) {
			sum += s.n
			ph.uploadSamples[i] = s
			i++
		}
	}

	// drop old samples
	ph.uploadSamples = ph.uploadSamples[:i]

	return uint32(sum)
}

func (ph *PeerHandler) recordDownload(n int) {

	debugs.NumTransferResponseReceived++
	debugs.LogNums()

	now := time.Now()

	ph.mu.Lock()
	defer ph.mu.Unlock()

	ph.downloadedTotal += uint64(n)
	ph.downloadSamples = append(ph.downloadSamples, bandwidthSample{
		t: now,
		n: n,
	})
}

func (ph *PeerHandler) DownloadRate() uint32 {
	cutoff := time.Now().Add(-5 * time.Second)

	ph.mu.Lock()
	defer ph.mu.Unlock()

	var sum int
	var i int

	for _, s := range ph.downloadSamples {
		if s.t.After(cutoff) {
			sum += s.n
			ph.downloadSamples[i] = s
			i++
		}
	}

	// drop old samples
	ph.downloadSamples = ph.downloadSamples[:i]

	return uint32(float32(sum) / float32(5.0))
}
