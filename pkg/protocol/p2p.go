package protocol

import (
	"encoding/json"
)

type PeerMessageType string

const (
	MsgHandshake PeerMessageType = "handshake"
	MsgBitfield  PeerMessageType = "bitfield"
	MsgHave      PeerMessageType = "have"
	MsgRequest   PeerMessageType = "request"
	MsgTransfer  PeerMessageType = "transfer"
	MsgReject    PeerMessageType = "reject"
)

type PeerMessage struct {
	InfoHash InfoHash        `json:"infohash"`
	Type     PeerMessageType `json:"type"`
	Payload  json.RawMessage `json:"payload"`
}

type HandshakePayload struct {
	InfoHash InfoHash `json:"info_hash"`
	PeerID   string   `json:"peer_id"`
}

type BitfieldPayload struct {
	Bits []byte `json:"bits"`
}

// Add to protocol package
type ConnectionState int

const (
	StateConnecting ConnectionState = iota
	StateHandshaking
	StateConnected
	StateClosed
)

type TransferRequestPayload struct {
	UnitIndex uint64 `json:"unit_index"`
}

type HavePayload struct {
	UnitIndex uint64 `json:"unit_index"`
}

type RejectPayload struct {
	UnitIndex uint64 `json:"unit_index"`
	Reason    string `json:"reason,omitempty"`
}

// TransferPayload includes the segment data and its Bao proof
type TransferPayload struct {
	UnitIndex uint64 `json:"unit_index"`
	Data      []byte `json:"data"`
	Proof     *Proof `json:"proof,omitempty"` // Optional proof for verification
}

// ----------------------------
// Proof types
// ----------------------------

type ProofNode struct {
	Hash  [32]byte
	Level uint8 // tree level (0 = leaf)
}

type Proof struct {
	LeafStart int64
	LeafCount int64
	Nodes     []ProofNode
}

const (
	HashSize = 32 // BLAKE3 output size
)

type Hash [HashSize]byte
