package core

import (
	"log"
	"sync"

	"github.com/baoswarm/baobun/pkg/protocol"
)

// Swarm tracks one torrent
type Swarm struct {
	File   *BaoFile
	FileIO *FileIO

	InfoHash protocol.InfoHash

	Downloaded uint64
	Uploaded   uint64

	Peers map[protocol.NodeKey]*PeerHandler // peerKey → handler

	mu sync.RWMutex // Protect Peers map

	// TransferUnit management
	TransferUnitManager *TransferUnitManager

	FileLocation string

	//Proof Cache, a place to keep validated proofs
	ProofCache map[uint64]*protocol.Proof // peerKey → handler
	//TODO: We should merge proofs upwards on the tree to mimimize memory footprint, and consider clearing this map when the full file is available since
	//at that point we can just generate proofs on demand, but needs to be researched if its worth keeping proof or not..
}

func NewSwarm(infoHash protocol.InfoHash, file *BaoFile, fileLocation string) *Swarm {
	swarm := &Swarm{
		File:         file,
		InfoHash:     infoHash,
		Peers:        make(map[protocol.NodeKey]*PeerHandler),
		FileLocation: fileLocation,
		ProofCache:   make(map[uint64]*protocol.Proof),
	}

	// Initialize FileIO with cache
	fileIO, err := NewFileIO(file, fileLocation)
	if err != nil {
		log.Printf("Warning: failed to initialize FileIO: %v", err)
	} else {
		swarm.FileIO = fileIO
	}

	//TODO: rework the existing files check, only check if file changed, persist etc, for now we just check the whole thing on startup.
	for i := uint64(0); i < fileIO.unitCount; i++ {
		data, err := fileIO.ReadTransferUnit(i)
		if err != nil {
			log.Printf("Warning: failed to read transfer unit: %v", err)
		}
		hasData := false
		for _, b := range data {
			if b != 0 {
				hasData = true
				break
			}
		}
		if hasData {
			fileIO.haveUnits.Set(i)
		}
	}

	// for i := uint64(0); i < fileIO.unitCount; i++ {
	// 	hasTransferUnit := swarm.FileIO.HasTransferUnit(i)
	// 	if hasTransferUnit {
	// 		log.Printf("We have transferUnit %d", i)
	// 		swarm.Have.Set(i)
	// 	}
	// }

	// Initialize transferUnit manager
	swarm.TransferUnitManager = NewTransferUnitManager(swarm, fileIO.unitCount)

	return swarm
}

func (s *Swarm) CalcLeft() uint64 {
	var left uint64

	for i := uint64(0); i < s.FileIO.unitCount; i++ {
		if s.FileIO.haveUnits.Has(i) {
			continue
		}

		transferUnitLen := uint64(s.File.TransferSize)
		offset := i * transferUnitLen

		if offset >= s.File.Length {
			break // no more file data
		}

		if offset+transferUnitLen > s.File.Length {
			transferUnitLen = s.File.Length - offset
		}

		left += transferUnitLen
	}

	return left
}

// Add methods to update peer bitfield information
func (s *Swarm) UpdatePeerBitfield(peer protocol.NodeKey, bitfield Bitfield) {
	s.mu.RLock()
	handler, exists := s.Peers[peer]
	s.mu.RUnlock()

	if !exists {
		return
	}

	handler.Bitfield = bitfield
}

// Mark a transferUnit as downloaded
func (s *Swarm) MarkTransferUnitComplete(transferUnitIndex uint64, data []byte) {
	s.FileIO.haveUnits.Set(transferUnitIndex)
	s.Downloaded += uint64(len(data))

	// Notify transferUnit manager
	s.TransferUnitManager.MarkTransferUnitComplete(transferUnitIndex)

	// Send HAVE messages to all connected peers
	s.BroadcastHave(transferUnitIndex)

	log.Println(s.FileIO.haveUnits.ToString(s.File.GetTransferUnitCount()))
}

// Broadcast HAVE message to all peers
func (s *Swarm) BroadcastHave(transferUnitIndex uint64) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, peer := range s.Peers {
		if peer.GetState() == protocol.StateConnected {
			go peer.SendHave(transferUnitIndex)
		}
	}
}

// GetFileIO returns a FileIO instance for a swarm's file
// func (s *Swarm) GetFileIO(cacheSize int) (*FileIO, error) {
// 	if s.File == nil {
// 		return nil, errors.New("swarm has no file associated")
// 	}

// 	return NewFileIO(s.File)
// }

// Don't forget to close FileIO when swarm is done
func (s *Swarm) Close() error {
	//TODO: make sure this is called when we are done with a swarm.
	if s.FileIO != nil {
		return s.FileIO.Close()
	}
	return nil
}
