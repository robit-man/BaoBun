package core

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/baoswarm/baobun/pkg/protocol"
)

const proofFileVersion = 1

type ProofStore struct {
	dir string
}

type proofDiskFile struct {
	Version   int             `json:"version"`
	UnitIndex uint64          `json:"unit_index"`
	Proof     proofDiskRecord `json:"proof"`
}

type proofDiskRecord struct {
	LeafStart int64              `json:"leaf_start"`
	LeafCount int64              `json:"leaf_count"`
	Nodes     []proofDiskNodeRef `json:"nodes"`
}

type proofDiskNodeRef struct {
	Hash  string `json:"hash"`
	Level uint8  `json:"level"`
}

func NewProofStore(fileLocation string, infoHash protocol.InfoHash) *ProofStore {
	return &ProofStore{
		dir: filepath.Join(
			fileLocation,
			".baobun",
			"proofs",
			hex.EncodeToString(infoHash[:]),
		),
	}
}

func (ps *ProofStore) LoadAll() (map[uint64]*protocol.Proof, error) {
	loaded := make(map[uint64]*protocol.Proof)

	entries, err := os.ReadDir(ps.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return loaded, nil
		}
		return loaded, fmt.Errorf("failed to read proof cache directory %q: %w", ps.dir, err)
	}

	loadFailures := 0

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		path := filepath.Join(ps.dir, entry.Name())

		index, err := parseUnitIndex(entry.Name())
		if err != nil {
			loadFailures++
			continue
		}

		data, err := os.ReadFile(path)
		if err != nil {
			loadFailures++
			continue
		}

		var onDisk proofDiskFile
		if err := json.Unmarshal(data, &onDisk); err != nil {
			loadFailures++
			continue
		}

		if onDisk.Version != proofFileVersion {
			loadFailures++
			continue
		}

		if onDisk.UnitIndex != index {
			loadFailures++
			continue
		}

		proof, err := diskToProof(onDisk.Proof)
		if err != nil {
			loadFailures++
			continue
		}

		loaded[index] = proof
	}

	if loadFailures > 0 {
		return loaded, fmt.Errorf(
			"loaded %d proofs with %d invalid proof files",
			len(loaded),
			loadFailures,
		)
	}

	return loaded, nil
}

func (ps *ProofStore) Save(unitIndex uint64, proof *protocol.Proof) error {
	if proof == nil {
		return fmt.Errorf("cannot save nil proof")
	}

	if err := os.MkdirAll(ps.dir, 0755); err != nil {
		return fmt.Errorf("failed to create proof cache directory: %w", err)
	}

	record, err := proofToDisk(proof)
	if err != nil {
		return err
	}

	onDisk := proofDiskFile{
		Version:   proofFileVersion,
		UnitIndex: unitIndex,
		Proof:     record,
	}

	data, err := json.MarshalIndent(onDisk, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal proof cache entry: %w", err)
	}
	data = append(data, '\n')

	target := ps.pathForUnit(unitIndex)
	tmp := target + ".tmp"

	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("failed to write proof cache temp file: %w", err)
	}

	if err := os.Rename(tmp, target); err != nil {
		// Windows does not overwrite existing files on rename.
		_ = os.Remove(target)
		if errRetry := os.Rename(tmp, target); errRetry != nil {
			_ = os.Remove(tmp)
			return fmt.Errorf("failed to finalize proof cache file: %w", errRetry)
		}
	}

	return nil
}

func (ps *ProofStore) pathForUnit(unitIndex uint64) string {
	return filepath.Join(ps.dir, fmt.Sprintf("%d.json", unitIndex))
}

func parseUnitIndex(name string) (uint64, error) {
	base := strings.TrimSuffix(name, filepath.Ext(name))
	if base == "" {
		return 0, fmt.Errorf("empty proof cache filename")
	}
	return strconv.ParseUint(base, 10, 64)
}

func proofToDisk(proof *protocol.Proof) (proofDiskRecord, error) {
	record := proofDiskRecord{
		LeafStart: proof.LeafStart,
		LeafCount: proof.LeafCount,
		Nodes:     make([]proofDiskNodeRef, 0, len(proof.Nodes)),
	}

	for _, node := range proof.Nodes {
		record.Nodes = append(record.Nodes, proofDiskNodeRef{
			Hash:  hex.EncodeToString(node.Hash[:]),
			Level: node.Level,
		})
	}

	return record, nil
}

func diskToProof(record proofDiskRecord) (*protocol.Proof, error) {
	out := &protocol.Proof{
		LeafStart: record.LeafStart,
		LeafCount: record.LeafCount,
		Nodes:     make([]protocol.ProofNode, 0, len(record.Nodes)),
	}

	for _, node := range record.Nodes {
		decoded, err := hex.DecodeString(node.Hash)
		if err != nil {
			return nil, fmt.Errorf("failed to decode proof node hash: %w", err)
		}
		if len(decoded) != 32 {
			return nil, fmt.Errorf("invalid proof node hash length: %d", len(decoded))
		}

		var hash [32]byte
		copy(hash[:], decoded)

		out.Nodes = append(out.Nodes, protocol.ProofNode{
			Hash:  hash,
			Level: node.Level,
		})
	}

	return out, nil
}

func cloneProof(in *protocol.Proof) *protocol.Proof {
	if in == nil {
		return nil
	}

	out := &protocol.Proof{
		LeafStart: in.LeafStart,
		LeafCount: in.LeafCount,
		Nodes:     make([]protocol.ProofNode, len(in.Nodes)),
	}
	copy(out.Nodes, in.Nodes)

	return out
}
