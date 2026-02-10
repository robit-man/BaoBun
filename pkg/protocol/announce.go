package protocol

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
)

type InfoHash [32]byte

func (ih InfoHash) Bytes() []byte {
	return []byte(ih[:])
}

func InfoHashFromBytes(bytes []byte) InfoHash {
	return InfoHash(bytes)
}

func (h InfoHash) MarshalJSON() ([]byte, error) {
	return json.Marshal(hex.EncodeToString(h[:]))
}

func (h *InfoHash) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	decoded, err := hex.DecodeString(s)
	if err != nil {
		return err
	}
	if len(decoded) != 32 {
		return fmt.Errorf("invalid InfoHash length")
	}

	copy(h[:], decoded)
	return nil
}

type NodeKey string

type Peer struct {
	NodeKey  NodeKey
	IsSeeder bool
}

type AnnounceEvent string

const (
	EventStarted   AnnounceEvent = "started"
	EventStopped   AnnounceEvent = "stopped"
	EventCompleted AnnounceEvent = "completed"
)

type AnnounceRequest struct {
	InfoHash   InfoHash
	Event      AnnounceEvent
	Uploaded   uint64
	Downloaded uint64
	Left       uint64
	Timestamp  uint64
	// Timestamp, Signature (later)
}

type AnnounceResponse struct {
	Interval int
	Peers    []Peer
}
