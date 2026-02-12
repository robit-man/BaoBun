package core

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/baoswarm/baobun/pkg/protocol"
)

func TestProofStoreSaveLoadRoundTrip(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "proof-store-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	var infoHash protocol.InfoHash
	copy(infoHash[:], mustDecodeHex32(t, "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"))

	store := NewProofStore(tempDir, infoHash)

	original := &protocol.Proof{
		LeafStart: 12,
		LeafCount: 4,
		Nodes: []protocol.ProofNode{
			{
				Hash:  mustHash32(t, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
				Level: 0,
			},
			{
				Hash:  mustHash32(t, "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
				Level: 1,
			},
		},
	}

	if err := store.Save(7, original); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded, err := store.LoadAll()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	got, ok := loaded[7]
	if !ok {
		t.Fatalf("expected proof for unit 7")
	}

	if got.LeafStart != original.LeafStart || got.LeafCount != original.LeafCount {
		t.Fatalf("leaf metadata mismatch: got (%d,%d), expected (%d,%d)",
			got.LeafStart, got.LeafCount, original.LeafStart, original.LeafCount)
	}

	if len(got.Nodes) != len(original.Nodes) {
		t.Fatalf("node count mismatch: got %d, expected %d", len(got.Nodes), len(original.Nodes))
	}

	for i := range got.Nodes {
		if got.Nodes[i].Level != original.Nodes[i].Level {
			t.Fatalf("node %d level mismatch: got %d, expected %d", i, got.Nodes[i].Level, original.Nodes[i].Level)
		}
		if got.Nodes[i].Hash != original.Nodes[i].Hash {
			t.Fatalf("node %d hash mismatch", i)
		}
	}
}

func TestProofStoreIgnoresCorruptFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "proof-store-corrupt-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	var infoHash protocol.InfoHash
	copy(infoHash[:], mustDecodeHex32(t, "11223344556677889900aabbccddeeff11223344556677889900aabbccddeeff"))

	store := NewProofStore(tempDir, infoHash)

	validProof := &protocol.Proof{
		LeafStart: 0,
		LeafCount: 1,
		Nodes: []protocol.ProofNode{
			{
				Hash:  mustHash32(t, "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"),
				Level: 0,
			},
		},
	}

	if err := store.Save(1, validProof); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	// Drop a corrupt file into the proof cache directory.
	dir := filepath.Join(tempDir, ".baobun", "proofs", hex.EncodeToString(infoHash[:]))
	if err := os.WriteFile(filepath.Join(dir, "2.json"), []byte("{not-json"), 0644); err != nil {
		t.Fatalf("failed to write corrupt proof file: %v", err)
	}

	loaded, err := store.LoadAll()
	if err == nil {
		t.Fatalf("expected load warning error for corrupt file")
	}

	if _, ok := loaded[1]; !ok {
		t.Fatalf("valid proof should still load when one file is corrupt")
	}
}

func mustDecodeHex32(t *testing.T, s string) []byte {
	t.Helper()

	decoded, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("hex decode failed: %v", err)
	}
	if len(decoded) != 32 {
		t.Fatalf("expected 32-byte input, got %d", len(decoded))
	}
	return decoded
}

func mustHash32(t *testing.T, s string) [32]byte {
	t.Helper()

	decoded := mustDecodeHex32(t, s)
	var out [32]byte
	copy(out[:], decoded)
	return out
}
