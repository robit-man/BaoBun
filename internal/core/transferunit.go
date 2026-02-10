// Add to protocol package or create a new transferunit.go file
package core

import (
	"sync"
	"time"

	"github.com/baoswarm/baobun/pkg/protocol"
)

type TransferUnitState int

const (
	TransferUnitStateMissing TransferUnitState = iota
	TransferUnitStateRequested
	TransferUnitStateDownloading
	TransferUnitStateComplete
	TransferUnitStateFailed
)

type TransferUnit struct {
	Index  uint64
	Hash   []byte
	Length uint32
	State  TransferUnitState

	// Which peers have this TransferUnit
	peersWithTransferUnit map[protocol.NodeKey]bool
	mu                    sync.RWMutex
}

type transferUnitRequest struct {
	Index    uint64
	From     protocol.NodeKey
	SentAt   time.Time
	Attempts int
	Timeout  time.Duration
}

type transferUnitAvailableEvent struct {
	peer         protocol.NodeKey
	transferUnit uint64
}

type transferUnitCompleteEvent struct {
	transferUnit uint64
	data         []byte
}

type peerUpdateEvent struct {
	peer     protocol.NodeKey
	online   bool
	bitfield Bitfield
}

type transferUnitRequestEvent struct {
	transferUnit uint64
	peer         protocol.NodeKey
}
