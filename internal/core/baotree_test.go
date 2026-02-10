package core

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/baoswarm/baobun/pkg/protocol"
)

func TestComputeMerkleRoot(t *testing.T) {
	// Create a temporary file with 10 KB of test data
	data := bytes.Repeat([]byte{0xAB}, 10*1024)
	tmpFile, err := ioutil.TempFile("", "bao_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.Write(data); err != nil {
		t.Fatal(err)
	}

	// Compute root using ComputeMerkleRoot
	f, err := os.Open(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	root, err := ComputeMerkleRootOnDisk(f)
	if err != nil {
		t.Fatal("ComputeMerkleRoot failed:", err)
	}

	if root == (protocol.Hash{}) {
		t.Fatal("Root hash is empty")
	}
}

func TestGenerateProofOnDisk(t *testing.T) {
	// Create a temporary file with 64 KB of test data
	data := bytes.Repeat([]byte{0xCD}, 64*1024)
	tmpFile, err := os.CreateTemp("", "bao_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.Write(data); err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	rootHash, err := ComputeMerkleRootOnDisk(f)
	if err != nil {
		t.Fatalf("failed to hash file: %s", err)
	}

	log.Println(rootHash)

	// Request a proof for a 16 KB segment starting at 8 KB
	offset := int64(8 * 1024)
	length := int64(16 * 1024)
	proof, root, err := GenerateProofOnDisk(f, offset, length)
	if err != nil {
		t.Fatal("GenerateProofOnDisk failed:", err)
	}

	// Extract the segment from file
	segment := data[offset : offset+length]

	// Verify the proof (using VerifyProof)
	if err := VerifyProof(segment, proof, rootHash, int64(len(data))); err != nil {
		t.Fatal("Proof verification failed:", err)
	}

	// Sanity checks
	if root == (protocol.Hash{}) {
		t.Fatal("Root hash is empty")
	}
}

// TestTransferScenario simulates the actual file transfer scenario
func TestTransferScenario(t *testing.T) {
	// Create a test file (similar to your real scenario)
	fileSize := int64(500 * 1024) // 500 KB file
	testData := make([]byte, fileSize)
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	tmpFile, err := os.CreateTemp("", "transfer_test_")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(testData); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	// Open file for reading
	file, err := os.Open(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	// SENDER SIDE: Compute root hash (shared via metadata)
	sharedRootHash, err := ComputeMerkleRootOnDisk(file)

	if err != nil {
		t.Fatal("Failed to compute root hash:", err)
	}

	t.Logf("Shared root hash: %x", sharedRootHash)

	// Simulate sending 64KB chunks
	chunkSize := int64(64 * 1024)
	numChunks := (fileSize + chunkSize - 1) / chunkSize

	for chunkIndex := int64(0); chunkIndex < numChunks; chunkIndex++ {
		offset := chunkIndex * chunkSize
		length := chunkSize
		if offset+length > fileSize {
			length = fileSize - offset
		}

		t.Logf("\n=== Testing chunk %d ===", chunkIndex)
		t.Logf("Offset: %d, Length: %d", offset, length)

		// SENDER: Generate proof
		proof, generatedRoot, err := GenerateProofOnDisk(file, offset, length)
		if err != nil {
			t.Fatalf("Chunk %d: Failed to generate proof: %v", chunkIndex, err)
		}

		t.Logf("Generated root: %x", generatedRoot)
		t.Logf("Siblings: %d", len(proof.Nodes))

		// Verify sender's root matches shared root
		if generatedRoot != sharedRootHash {
			t.Fatalf("Chunk %d: Sender's generated root doesn't match shared root!", chunkIndex)
		}

		// Read the actual segment data (this is what gets sent)
		segment := make([]byte, length)
		_, err = file.ReadAt(segment, offset)
		if err != nil {
			t.Fatalf("Chunk %d: Failed to read segment: %v", chunkIndex, err)
		}

		// RECEIVER SIDE: Verify the proof
		err = VerifyProof(segment, proof, sharedRootHash, fileSize)
		if err != nil {
			t.Fatalf("Chunk %d: Proof verification failed: %v", chunkIndex, err)
		}

		t.Logf("✓ Chunk %d verified successfully", chunkIndex)
	}

	t.Log("\n✓ All chunks verified successfully!")
}

// TestSpecificFailingChunk tests a specific chunk that's failing
func TestSpecificFailingChunk(t *testing.T) {
	// Use the same file size from your real scenario
	// Adjust this based on your actual file
	fileSize := int64(500 * 1024) // 500 KB
	testData := make([]byte, fileSize)
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	tmpFile, err := os.CreateTemp("", "specific_chunk_test_")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(testData); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	file, err := os.Open(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	// Compute shared root
	sharedRootHash, err := ComputeMerkleRootOnDisk(file)

	if err != nil {
		t.Fatal(err)
	}

	// Test chunk 4 (the one failing in your logs)
	chunkIndex := int64(4)
	offset := chunkIndex * 64 * 1024
	length := int64(64 * 1024)

	if offset+length > fileSize {
		length = fileSize - offset
	}

	t.Logf("Testing failing chunk %d", chunkIndex)
	t.Logf("Offset: %d, Length: %d", offset, length)
	t.Logf("Shared root: %x", sharedRootHash)

	// Generate proof
	proof, generatedRoot, err := GenerateProofOnDisk(file, offset, length)
	if err != nil {
		t.Fatal("Failed to generate proof:", err)
	}

	t.Logf("Generated root: %x", generatedRoot)
	t.Logf("Siblings: %d", len(proof.Nodes))

	// Read segment
	segment := make([]byte, length)
	_, err = file.ReadAt(segment, offset)
	if err != nil {
		t.Fatal("Failed to read segment:", err)
	}

	// Verify
	err = VerifyProof(segment, proof, sharedRootHash, fileSize)
	if err != nil {
		t.Fatalf("Verification failed: %v", err)
	}

	t.Log("✓ Verification successful")
}

// TestProofRoundTrip tests proof generation and verification for all chunks
func TestProofRoundTrip(t *testing.T) {
	sizes := []int64{
		1024,        // 1 KB (1 chunk)
		64 * 1024,   // 64 KB (1 chunk)
		128 * 1024,  // 128 KB (2 chunks)
		500 * 1024,  // 500 KB (8 chunks)
		1024 * 1024, // 1 MB (16 chunks)
	}

	for _, fileSize := range sizes {
		t.Run(fmt.Sprintf("FileSize_%dKB", fileSize/1024), func(t *testing.T) {
			testData := bytes.Repeat([]byte{0xAB}, int(fileSize))

			tmpFile, err := os.CreateTemp("", "roundtrip_test_")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpFile.Name())

			if _, err := tmpFile.Write(testData); err != nil {
				t.Fatal(err)
			}
			tmpFile.Close()

			file, err := os.Open(tmpFile.Name())
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()

			// Compute root
			rootHash, err := ComputeMerkleRootOnDisk(file)

			if err != nil {
				t.Fatal(err)
			}

			// Test all chunks
			chunkSize := int64(64 * 1024)
			numChunks := (fileSize + chunkSize - 1) / chunkSize

			for i := int64(0); i < numChunks; i++ {
				offset := i * chunkSize
				length := chunkSize
				if offset+length > fileSize {
					length = fileSize - offset
				}

				proof, genRoot, err := GenerateProofOnDisk(file, offset, length)
				if err != nil {
					t.Fatalf("Chunk %d: Generate failed: %v", i, err)
				}

				if genRoot != rootHash {
					t.Fatalf("Chunk %d: Root mismatch", i)
				}

				segment := testData[offset : offset+length]
				err = VerifyProof(segment, proof, rootHash, fileSize)
				if err != nil {
					t.Fatalf("Chunk %d: Verify failed: %v", i, err)
				}
			}

			t.Logf("✓ All %d chunks verified for %d KB file", numChunks, fileSize/1024)
		})
	}
}

func TestDebug500KB(t *testing.T) {
	fileSize := int64(500 * 1024)
	testData := make([]byte, fileSize)
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	tmpFile, err := os.CreateTemp("", "debug_500kb_")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(testData); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	file, err := os.Open(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	// Compute root
	fileInfo, _ := file.Stat()
	t.Logf("File size: %d", fileInfo.Size())

	rootHash, err := ComputeMerkleRootOnDisk(file)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Root hash: %x", rootHash)

	// Test first chunk (should work)
	t.Log("\n=== Testing chunk 0 ===")
	testChunk(t, file, 0, rootHash, testData)

	// Test middle chunk (likely to fail)
	t.Log("\n=== Testing chunk 4 ===")
	testChunk(t, file, 4, rootHash, testData)

	// Test last chunk
	lastChunk := (fileSize+64*1024-1)/(64*1024) - 1
	t.Logf("\n=== Testing last chunk %d ===", lastChunk)
	testChunk(t, file, lastChunk, rootHash, testData)
}

func testChunk(t *testing.T, file *os.File, chunkIndex int64, rootHash protocol.Hash, fullData []byte) {
	offset := chunkIndex * 64 * 1024
	length := int64(64 * 1024)

	fileInfo, _ := file.Stat()
	if offset+length > fileInfo.Size() {
		length = fileInfo.Size() - offset
	}

	t.Logf("Chunk %d: offset=%d, length=%d", chunkIndex, offset, length)

	// Generate proof
	proof, genRoot, err := GenerateProofOnDisk(file, offset, length)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	t.Logf("Generated root: %x", genRoot)
	t.Logf("Expected root:  %x", rootHash)
	t.Logf("Roots match: %v", genRoot == rootHash)
	t.Logf("Proof.LeafStart: %d", proof.LeafStart)
	t.Logf("Proof.LeafCount: %d", proof.LeafCount)
	t.Logf("Siblings: %d", len(proof.Nodes))

	// Calculate leaf positions
	leafStart := offset / LeafSize
	leafEnd := (offset + length + LeafSize - 1) / LeafSize
	t.Logf("Leaf range: [%d, %d)", leafStart, leafEnd)

	// Show sibling hashes
	for i, sib := range proof.Nodes {
		t.Logf("  Sibling[%d]: %x", i, sib.Hash[:8])
	}

	// Get segment
	segment := fullData[offset : offset+length]

	// Try verification
	err = VerifyProof(segment, proof, rootHash, fileInfo.Size())
	if err != nil {
		t.Logf("❌ Verification FAILED: %v", err)
		t.Fatal("Verification failed")
	} else {
		t.Log("✓ Verification succeeded")
	}
}

func debugPrintProof(proof *protocol.Proof) {
	fmt.Printf("Proof: start=%d, count=%d, nodes=%d\n",
		proof.LeafStart, proof.LeafCount, len(proof.Nodes))
	for i, node := range proof.Nodes {
		fmt.Printf("  Node %d: level=%d hash=%x\n", i, node.Level, node.Hash[:4])
	}
}

// Add this test
func TestSingleLeaf(t *testing.T) {
	data := make([]byte, 1024) // Exactly 1 leaf
	for i := range data {
		data[i] = byte(i % 256)
	}

	// Write to temp file
	tmpFile, _ := os.CreateTemp("", "test")
	defer os.Remove(tmpFile.Name())
	tmpFile.Write(data)
	tmpFile.Close()

	// Open and generate proof
	file, _ := os.Open(tmpFile.Name())
	defer file.Close()

	proof, root, err := GenerateProofOnDisk(file, 0, 1024)
	if err != nil {
		t.Fatal(err)
	}

	debugPrintProof(proof)

	// Verify
	err = VerifyProof(data, proof, root, 1024)
	if err != nil {
		t.Fatal(err)
	}
}
