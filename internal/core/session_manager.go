package core

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/baoswarm/baobun/internal/config"
	"github.com/baoswarm/baobun/pkg/protocol"
	"github.com/nknorg/ncp-go"
	nkn "github.com/nknorg/nkn-sdk-go"
)

type SessionManager struct {
	client   *nkn.MultiClient
	mu       sync.Mutex
	sessions map[protocol.NodeKey]*Session
	swarms   map[protocol.InfoHash]*Swarm
}

type Session struct {
	conn     net.Conn
	peer     protocol.NodeKey
	refCount int
	writeMu  sync.RWMutex
	created  time.Time
}

func NewSessionManager(client *nkn.MultiClient) *SessionManager {
	sm := &SessionManager{
		client:   client,
		sessions: make(map[protocol.NodeKey]*Session),
		swarms:   make(map[protocol.InfoHash]*Swarm),
	}

	go sm.acceptLoop()
	return sm
}

func (sm *SessionManager) acceptLoop() {
	log.Println("Starting the accept loop")

	for {
		conn, err := sm.client.Accept()
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}

		peer := protocol.NodeKey(conn.RemoteAddr().String())
		log.Printf("Accepted session with %s", peer)

		// Create session and start read loop
		sess := &Session{
			conn:     conn,
			peer:     peer,
			refCount: 1,
			created:  time.Now(),
		}

		sm.mu.Lock()
		sm.sessions[peer] = sess
		sm.mu.Unlock()

		go sm.readLoop(sess)
	}
}

func (sm *SessionManager) RegisterSwarm(
	swarm *Swarm,
) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.swarms[swarm.InfoHash] = swarm
}

func (sm *SessionManager) GetSession(peer protocol.NodeKey) (*Session, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if s, ok := sm.sessions[peer]; ok {
		s.refCount++
		return s, nil
	}

	conn, err := sm.client.DialWithConfig(string(peer), &nkn.DialConfig{
		DialTimeout: config.DialTimeoutMs,
		SessionConfig: &ncp.Config{
			MTU: config.MTU,
		},
	})
	if err != nil {
		return nil, err
	}

	sess := &Session{
		conn:     conn,
		peer:     peer,
		refCount: 1,
	}

	sm.sessions[peer] = sess
	go sm.readLoop(sess)

	return sess, nil
}

func (sm *SessionManager) readLoop(sess *Session) {
	serializer := NewProtobufSerializer()
	reader := bufio.NewReader(sess.conn)

	for {
		// 1. Read length prefix
		var length uint32
		if err := binary.Read(reader, binary.BigEndian, &length); err != nil {
			log.Printf("Read length error from %s: %v", sess.peer, err)
			sm.Release(sess.peer)
			return
		}

		// 2. Read protobuf payload
		buf := make([]byte, length)
		if _, err := io.ReadFull(reader, buf); err != nil {
			log.Printf("Read body error from %s: %v", sess.peer, err)
			sm.Release(sess.peer)
			return
		}

		// 3. Unmarshal protobuf
		var msg protocol.PeerMessage
		if err := serializer.UnmarshalPeerMessage(buf, &msg); err != nil {
			log.Printf("Unmarshal error from %s: %v", sess.peer, err)
			sm.Release(sess.peer)
			return
		}

		// 4. Handle handshake specially
		if msg.Type == protocol.MsgHandshake {
			sm.handleHandshake(sess, msg, serializer)
			continue
		}

		// 5. Find swarm
		sm.mu.Lock()
		swarm, swarmExists := sm.swarms[msg.InfoHash]
		sm.mu.Unlock()

		if !swarmExists {
			log.Printf("No swarm for infohash: %s", msg.InfoHash)
			continue
		}

		// 6. Find handler
		swarm.mu.RLock()
		handler := swarm.Peers[sess.peer]
		swarm.mu.RUnlock()

		if handler == nil {
			log.Printf("No handler for peer %s in swarm %s", sess.peer, msg.InfoHash)
			continue
		}

		handler.HandleMessage(msg)
	}
}

func (sm *SessionManager) handleHandshake(sess *Session, msg protocol.PeerMessage, serializer Serializer) {
	var hs protocol.HandshakePayload
	if err := serializer.UnmarshalHandshakePayload(msg.Payload, &hs); err != nil {
		log.Printf("Failed to unmarshal handshake: %v", err)
		return
	}

	log.Printf("Received handshake from %s for swarm %s", sess.peer, hs.InfoHash)

	sm.mu.Lock()
	swarm, swarmExists := sm.swarms[hs.InfoHash]
	sm.mu.Unlock()

	if !swarmExists {
		log.Printf("Swarm %s not found for handshake from %s", hs.InfoHash, sess.peer)
		return
	}

	// Check if we already have a handler for this peer
	swarm.mu.Lock()
	handler, exists := swarm.Peers[sess.peer]

	if !exists {
		// Create handler for incoming connection
		handler = &PeerHandler{
			Peer:               sess.peer,
			Swarm:              swarm,
			Session:            sess,
			state:              protocol.StateHandshaking,
			handshakeReceived:  make(chan struct{}),
			connected:          make(chan struct{}),
			theirHandshakeSeen: true,
			serializer:         NewProtobufSerializer(),
		}
		swarm.Peers[sess.peer] = handler
	} else {
		// Update existing handler
		handler.theirHandshakeSeen = true
		handler.Session = sess // Update session reference
	}
	swarm.mu.Unlock()

	// Send our handshake back
	if err := handler.SendHandshake(sess.peer); err != nil {
		log.Printf("Failed to send handshake response: %v", err)
		return
	}

	// Mark as connected if we've completed handshake
	if handler.ourHandshakeSent && handler.theirHandshakeSeen {
		handler.SetState(protocol.StateConnected)

		log.Println("Sending bitfield:")
		uploadBitfield := BitfieldFromBytes(swarm.UploadBitfieldBytes())
		log.Println(uploadBitfield.ToString(swarm.FileIO.unitCount))

		// Send bitfield after handshake
		if err := handler.SendBitfield(swarm.UploadBitfieldBytes()); err != nil {
			log.Printf("bitfield send failed: %v", err)
			// Continue anyway - this isn't fatal
		}
	}

	// Handle the handshake message
	handler.HandleMessage(msg)
}

// Enhanced ConnectPeer method
func (sm *SessionManager) ConnectPeer(
	swarm *Swarm,
	peerKey protocol.NodeKey,
	timeout time.Duration,
) (*PeerHandler, error) {

	// Check if we already have a handler
	swarm.mu.RLock()
	if handler, exists := swarm.Peers[peerKey]; exists {
		swarm.mu.RUnlock()

		currentState := handler.GetState()
		// If already connected, return it
		if currentState >= protocol.StateConnected {
			return handler, nil
		}

		// If handshaking, wait for completion
		if handler.WaitForConnection(timeout) {
			handler.state = protocol.StateClosed
			return handler, nil
		}
		return nil, fmt.Errorf("connection timeout")
	}
	swarm.mu.RUnlock()

	// Create new handler
	handler, err := BaoHandler(sm, swarm, peerKey, true, NewProtobufSerializer())
	if err != nil {
		return nil, err
	}

	// Send handshake
	if err := handler.SendHandshake(peerKey); err != nil {
		handler.Close(sm)
		return nil, err
	}

	// Wait for handshake response
	if !handler.WaitForHandshake(timeout) {
		handler.Close(sm)
		return nil, fmt.Errorf("handshake timeout")
	}

	/*log.Println("Sending bitfield as handshake response:")
	log.Println(swarm.FileIO.haveUnits.ToString(swarm.FileIO.unitCount))

	// Send bitfield after handshake
	if err := handler.SendBitfield(swarm.FileIO.haveUnits.Bytes()); err != nil {
		log.Printf("bitfield send failed: %v", err)
		// Continue anyway - this isn't fatal
	}*/

	handler.SetState(protocol.StateConnected)
	return handler, nil
}

func (sm *SessionManager) Release(peer protocol.NodeKey) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	s := sm.sessions[peer]
	if s == nil {
		return
	}

	s.refCount--
	if s.refCount > 0 {
		return
	}

	s.conn.Close()
	delete(sm.sessions, peer)
}
