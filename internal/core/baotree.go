package core

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/baoswarm/baobun/pkg/protocol"
	"github.com/zeebo/blake3"
)

const LeafSize = 1024 // 1KB BAO leaf size

// ----------------------------
// Utilities
// ----------------------------

func nextPow2(v int64) int64 {
	v--
	v |= v >> 1
	v |= v >> 2
	v |= v >> 4
	v |= v >> 8
	v |= v >> 16
	v |= v >> 32
	v++
	return v
}

func hashLeaf(data []byte) [32]byte {
	if len(data) == 0 {
		// Empty leaf hash
		return blake3.Sum256(nil)
	}
	return blake3.Sum256(data)
}

func hashParent(left, right [32]byte) [32]byte {
	h := blake3.New()
	h.Write(left[:])
	h.Write(right[:])
	return *(*[32]byte)(h.Sum(nil))
}

// ----------------------------
// Disk hashing helpers
// ----------------------------

func readLeaf(f *os.File, leaf int64) ([32]byte, error) {
	buf := make([]byte, LeafSize)
	n, err := f.ReadAt(buf, leaf*LeafSize)
	if err != nil && err != io.EOF {
		return [32]byte{}, err
	}
	// If we read less than a full leaf, the rest is implicitly zero
	if n < LeafSize {
		// Pad with zeros
		for i := n; i < LeafSize; i++ {
			buf[i] = 0
		}
	}
	return hashLeaf(buf), nil
}

// Hash a full subtree rooted at (start, size)
func hashSubtree(f *os.File, start, size, totalLeaves int64) ([32]byte, error) {
	if size == 1 {
		if start >= totalLeaves {
			// Empty leaf
			return hashLeaf(nil), nil
		}
		return readLeaf(f, start)
	}

	half := size / 2
	l, err := hashSubtree(f, start, half, totalLeaves)
	if err != nil {
		return [32]byte{}, err
	}
	r, err := hashSubtree(f, start+half, half, totalLeaves)
	if err != nil {
		return [32]byte{}, err
	}
	return hashParent(l, r), nil
}

// ----------------------------
// Root computation
// ----------------------------

func ComputeMerkleRootOnDisk(f *os.File) ([32]byte, error) {
	info, err := f.Stat()
	if err != nil {
		return [32]byte{}, err
	}

	totalLeaves := (info.Size() + LeafSize - 1) / LeafSize
	treeLeaves := nextPow2(totalLeaves)

	return hashSubtree(f, 0, treeLeaves, totalLeaves)
}

// ----------------------------
// Proof generation (FIXED for 1KB leaves)
// ----------------------------

func GenerateProofOnDisk(f *os.File, offset, length int64) (*protocol.Proof, [32]byte, error) {
	info, err := f.Stat()
	if err != nil {
		return nil, [32]byte{}, err
	}

	startLeaf := offset / LeafSize
	endLeaf := (offset + length + LeafSize - 1) / LeafSize

	totalLeaves := (info.Size() + LeafSize - 1) / LeafSize
	treeLeaves := nextPow2(totalLeaves)

	proof := &protocol.Proof{
		LeafStart: startLeaf,
		LeafCount: endLeaf - startLeaf,
	}

	// Calculate tree height
	height := uint8(0)
	for n := treeLeaves; n > 1; n >>= 1 {
		height++
	}

	var walk func(start, size int64, level uint8) ([32]byte, error)
	walk = func(start, size int64, level uint8) ([32]byte, error) {
		// Calculate coverage of current subtree
		subtreeStartLeaf := start
		subtreeEndLeaf := start + size

		// Check if this subtree is completely outside the range
		if subtreeEndLeaf <= startLeaf || subtreeStartLeaf >= endLeaf {
			// Completely outside - compress to single hash
			h, err := hashSubtree(f, start, size, totalLeaves)
			if err != nil {
				return [32]byte{}, err
			}
			proof.Nodes = append(proof.Nodes, protocol.ProofNode{
				Hash:  h,
				Level: level,
			})
			return h, nil
		}

		// Check if this is a leaf
		if size == 1 {
			if start >= totalLeaves {
				// Empty leaf
				return hashLeaf(nil), nil
			}
			return readLeaf(f, start)
		}

		// Internal node - recurse
		half := size / 2
		l, err := walk(start, half, level-1)
		if err != nil {
			return [32]byte{}, err
		}
		r, err := walk(start+half, half, level-1)
		if err != nil {
			return [32]byte{}, err
		}
		return hashParent(l, r), nil
	}

	root, err := walk(0, treeLeaves, height)
	if err != nil {
		return nil, [32]byte{}, err
	}

	return proof, root, nil
}

func VerifyProof(segment []byte, proof *protocol.Proof, expectedRoot [32]byte, fileSize int64) error {
	if proof.LeafCount == 0 {
		return errors.New("empty proof")
	}

	// Calculate tree size from file size
	totalLeaves := (fileSize + LeafSize - 1) / LeafSize
	treeLeaves := nextPow2(totalLeaves)

	// Compute leaf hashes for the segment
	leafHashes := make([][32]byte, proof.LeafCount)
	for i := int64(0); i < proof.LeafCount; i++ {
		start := i * LeafSize
		end := start + LeafSize
		if end > int64(len(segment)) {
			// Pad with zeros
			padded := make([]byte, LeafSize)
			copy(padded, segment[start:])
			leafHashes[i] = hashLeaf(padded)
		} else {
			leafHashes[i] = hashLeaf(segment[start:end])
		}
	}

	// Track position in proof nodes
	proofIdx := 0

	// Recursively verify from leaves to root
	var verify func(start, size, level int64) ([32]byte, error)
	verify = func(start, size, level int64) ([32]byte, error) {
		// Check if this range overlaps with our segment
		segStart := proof.LeafStart
		segEnd := proof.LeafStart + proof.LeafCount

		rangeStart := start
		rangeEnd := start + size

		// No overlap - this should be in proof
		if rangeEnd <= segStart || rangeStart >= segEnd {
			if proofIdx >= len(proof.Nodes) {
				return [32]byte{}, errors.New("missing proof node")
			}
			if proof.Nodes[proofIdx].Level != uint8(level) {
				return [32]byte{}, fmt.Errorf("proof node level mismatch at index %d: got %d, expected %d",
					proofIdx, proof.Nodes[proofIdx].Level, level)
			}
			hash := proof.Nodes[proofIdx].Hash
			proofIdx++
			return hash, nil
		}

		// Fully contained in segment
		if size == 1 {
			// This is a leaf in our segment
			idx := start - segStart
			if idx < 0 || idx >= int64(len(leafHashes)) {
				return [32]byte{}, fmt.Errorf("leaf index out of range: %d", idx)
			}
			return leafHashes[idx], nil
		}

		// Partial overlap - recurse
		half := size / 2
		left, err := verify(start, half, level-1)
		if err != nil {
			return [32]byte{}, err
		}
		right, err := verify(start+half, half, level-1)
		if err != nil {
			return [32]byte{}, err
		}
		return hashParent(left, right), nil
	}

	// Calculate tree height
	height := int64(0)
	for n := treeLeaves; n > 1; n >>= 1 {
		height++
	}

	root, err := verify(0, treeLeaves, height)
	if err != nil {
		return err
	}

	if root != expectedRoot {
		return fmt.Errorf("root mismatch: got %x, expected %x",
			root[:4], expectedRoot[:4])
	}

	return nil
}
