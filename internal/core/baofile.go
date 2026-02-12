package core

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/baoswarm/baobun/internal/config"
	"github.com/baoswarm/baobun/pkg/protocol"
	"github.com/zeebo/blake3"
)

// BaoFile represents a .bao swarm file using BLAKE3's native tree
type BaoFile struct {
	Name     string            `json:"name"`
	Length   uint64            `json:"length"`
	RootHash string            `json:"root_hash"` // BLAKE3 root hash of entire file
	InfoHash protocol.InfoHash `json:"info_hash"` // BLAKE3 of canonical JSON representation
	Trackers []string          `json:"trackers"`  // Tracker addresses
}

// CanonicalBaoFile is used for consistent hashing
type CanonicalBaoFile struct {
	Name         string   `json:"name"`
	Length       uint64   `json:"length"`
	TransferSize uint64   `json:"transfer_size"`
	RootHash     string   `json:"root_hash"`
	Trackers     []string `json:"trackers"`
}

// CreateFromFile creates an BaoFile from a local file using BLAKE3's tree
func CreateFromFile(filePath string, trackers []string) (*BaoFile, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	filesize := fi.Size()

	rootHash, err := ComputeMerkleRootOnDisk(file)
	if err != nil {
		return nil, fmt.Errorf("failed to hash file: %w", err)
	}

	rootHashHex := hex.EncodeToString(rootHash[:])

	bao := &BaoFile{
		Name:     fi.Name(),
		Length:   uint64(filesize),
		Trackers: trackers,
		RootHash: rootHashHex,
	}

	// Calculate info hash
	if err := bao.calculateInfoHash(); err != nil {
		return nil, fmt.Errorf("failed to calculate info hash: %w", err)
	}

	return bao, nil
}

// calculateInfoHash computes the BLAKE3 hash of the canonical JSON representation
func (n *BaoFile) calculateInfoHash() error {
	// Create canonical representation
	canonical := CanonicalBaoFile{
		Name:     n.Name,
		Length:   n.Length,
		RootHash: n.RootHash,
		Trackers: n.Trackers,
	}

	// Sort for consistency
	sort.Strings(canonical.Trackers)

	// Marshal with sorted keys
	data, err := json.Marshal(canonical)
	if err != nil {
		return fmt.Errorf("failed to marshal for hashing: %w", err)
	}

	// Use BLAKE3
	hash := blake3.Sum256(data)
	n.InfoHash = hash
	return nil
}

// GetTransferUnitCount returns the number of transfer units
func (n *BaoFile) GetTransferUnitCount() uint64 {
	return (n.Length + uint64(config.TransferUnitSize-1)) / uint64(config.TransferUnitSize)
}

// GetTransferUnitSize returns the size of a specific transfer unit
func (n *BaoFile) GetTransferUnitSize(unitIndex uint64) (uint64, error) {
	if unitIndex >= n.GetTransferUnitCount() {
		return 0, fmt.Errorf("transfer unit index out of bounds")
	}

	start := unitIndex * uint64(config.TransferUnitSize)
	end := start + uint64(config.TransferUnitSize)
	if end > n.Length {
		return n.Length - start, nil
	}

	return uint64(config.TransferUnitSize), nil
}

// Save writes the BaoFile to disk
func (n *BaoFile) Save(outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(n); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}

// SaveDefault writes the BaoFile with default naming (filename.bao)
func (n *BaoFile) SaveDefault() error {
	outputPath := n.Name + ".bao"
	return n.Save(outputPath)
}

// Load loads an BaoFile from disk
func Load(filePath string) (*BaoFile, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open .bao file: %w", err)
	}
	defer file.Close()

	var bao BaoFile
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&bao); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	// Recalculate info hash to ensure consistency
	if err := bao.calculateInfoHash(); err != nil {
		return nil, fmt.Errorf("failed to recalculate info hash: %w", err)
	}

	return &bao, nil
}

// LoadFromBytes loads an BaoFile from JSON bytes
func LoadFromBytes(data []byte) (*BaoFile, error) {
	var bao BaoFile
	if err := json.Unmarshal(data, &bao); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	// Recalculate info hash to ensure consistency
	if err := bao.calculateInfoHash(); err != nil {
		return nil, fmt.Errorf("failed to recalculate info hash: %w", err)
	}

	return &bao, nil
}

// IsSameContent compares root hashes of two BaoFiles
func (n *BaoFile) IsSameContent(other *BaoFile) bool {
	return n.RootHash == other.RootHash && n.Length == other.Length
}
