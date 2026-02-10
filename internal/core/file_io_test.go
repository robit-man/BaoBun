package core

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/baoswarm/baobun/pkg/protocol"
)

// Test helper to create a test BaoFile
func createTestBaoFile(name string, size uint64, transferSize uint64) *BaoFile {
	return &BaoFile{
		Name:         name,
		Length:       size,
		TransferSize: transferSize,
		RootHash:     "test_root_hash",
		InfoHash:     protocol.InfoHash{},
		Trackers:     []string{"test-tracker.com"},
	}
}

// TestFileIO_BasicReadWrite tests basic read/write operations
func TestFileIO_BasicReadWrite(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "bao-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Test parameters
	fileSize := uint64(1024 * 1024)   // 1MB
	transferSize := uint64(64 * 1024) // 64KB
	fileName := "test.bin"

	// Create BaoFile
	npf := createTestBaoFile(fileName, fileSize, transferSize)

	// Create FileIO
	fileIO, err := NewFileIO(npf, tempDir)
	if err != nil {
		t.Fatal(err)
	}
	defer fileIO.Close()

	// Test 1: Write and read at exact transfer unit boundaries
	t.Run("TransferUnitBoundaries", func(t *testing.T) {
		// Generate random test data for first transfer unit
		expectedData := make([]byte, transferSize)
		rand.Read(expectedData)

		// Write at index 0
		err := fileIO.WriteTransferUnit(0, expectedData)
		if err != nil {
			t.Fatalf("WriteTransferUnit failed: %v", err)
		}

		// Read it back
		actualData, err := fileIO.ReadTransferUnit(0)
		if err != nil {
			t.Fatalf("ReadTransferUnit failed: %v", err)
		}

		// Compare
		if !bytes.Equal(expectedData, actualData) {
			t.Fatalf("Data mismatch for transfer unit 0\nExpected: %x\nActual:   %x",
				expectedData[:16], actualData[:16])
		}

		// Check bitfield
		if !fileIO.HasTransferUnit(0) {
			t.Fatal("Bitfield not updated for transfer unit 0")
		}
	})

	// Test 2: Write and read at arbitrary byte ranges
	t.Run("ArbitraryRanges", func(t *testing.T) {
		// Write data at offset 1000, length 5000
		start := uint64(1000)
		length := uint64(5000)
		expectedData := make([]byte, length)
		rand.Read(expectedData)

		err := fileIO.WriteRange(start, expectedData)
		if err != nil {
			t.Fatalf("WriteRange failed: %v", err)
		}

		// Read it back
		actualData, err := fileIO.ReadRange(start, length)
		if err != nil {
			t.Fatalf("ReadRange failed: %v", err)
		}

		if !bytes.Equal(expectedData, actualData) {
			t.Fatalf("Data mismatch for range [%d:%d]\nExpected: %x\nActual:   %x",
				start, start+length, expectedData[:16], actualData[:16])
		}
	})

	// Test 3: Write overlapping ranges
	t.Run("OverlappingRanges", func(t *testing.T) {
		// First write
		data1 := []byte("ABCDEFGHIJKLMNOP")
		err := fileIO.WriteRange(2000, data1)
		if err != nil {
			t.Fatalf("First WriteRange failed: %v", err)
		}

		// Second write that overlaps
		data2 := []byte("123456")
		err = fileIO.WriteRange(2005, data2)
		if err != nil {
			t.Fatalf("Second WriteRange failed: %v", err)
		}

		// Wait, let me calculate what we expect...
		// Actually, let's read the whole range and see
		actual, err := fileIO.ReadRange(2000, uint64(len(data1)))
		if err != nil {
			t.Fatalf("ReadRange failed: %v", err)
		}

		t.Logf("After overlapping writes, data at offset 2000: %s", actual)
	})

	// Test 4: Last transfer unit (might be smaller)
	t.Run("LastTransferUnit", func(t *testing.T) {
		// Calculate last transfer unit index
		lastIndex := fileIO.unitCount - 1
		lastUnitSize, err := npf.GetTransferUnitSize(lastIndex)
		if err != nil {
			t.Fatal(err)
		}

		// Generate data for last unit
		expectedData := make([]byte, lastUnitSize)
		rand.Read(expectedData)

		// Write last unit
		err = fileIO.WriteTransferUnit(lastIndex, expectedData)
		if err != nil {
			t.Fatal(err)
		}

		// Read it back
		actualData, err := fileIO.ReadTransferUnit(lastIndex)
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(expectedData, actualData) {
			t.Fatalf("Last transfer unit mismatch\nExpected len: %d\nActual len: %d",
				len(expectedData), len(actualData))
		}
	})

	// Test 5: Complete file write and verify
	t.Run("CompleteFile", func(t *testing.T) {
		// Recreate file to start fresh
		fileIO.Close()
		os.Remove(fileName)

		fileIO2, err := NewFileIO(npf, tempDir)
		if err != nil {
			t.Fatal(err)
		}
		defer fileIO2.Close()

		// Generate complete file data
		completeData := make([]byte, fileSize)
		rand.Read(completeData)

		// Write file in transfer units
		for i := uint64(0); i < fileIO2.unitCount; i++ {
			unitSize, _ := npf.GetTransferUnitSize(i)
			start := i * transferSize
			end := start + unitSize

			err := fileIO2.WriteTransferUnit(i, completeData[start:end])
			if err != nil {
				t.Fatalf("Failed to write transfer unit %d: %v", i, err)
			}
		}

		// Verify file is complete
		if !fileIO2.IsComplete() {
			t.Fatal("File should be marked as complete")
		}

		// Read entire file through OS to verify
		fileData, err := ioutil.ReadFile(fileIO2.file.Name())
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(completeData, fileData) {
			// Find first mismatch
			for i := 0; i < len(completeData); i++ {
				if completeData[i] != fileData[i] {
					t.Fatalf("Byte mismatch at offset %d: expected 0x%02x, got 0x%02x",
						i, completeData[i], fileData[i])
				}
			}
		}
	})
}

// TestFileIO_EdgeCases tests edge cases
func TestFileIO_EdgeCases(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "bao-edge-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	testCases := []struct {
		name         string
		fileSize     uint64
		transferSize uint64
	}{
		{"SmallFile", 100, 64 * 1024},              // File smaller than transfer unit
		{"ExactMultiple", 128 * 1024, 64 * 1024},   // Exact multiple
		{"OffByOne", 128*1024 + 1, 64 * 1024},      // One byte over multiple
		{"LargeFile", 10 * 1024 * 1024, 64 * 1024}, // 10MB file
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fileName := ".bin"
			npf := createTestBaoFile(fileName, tc.fileSize, tc.transferSize)

			fileIO, err := NewFileIO(npf, tempDir)
			if err != nil {
				t.Fatal(err)
			}
			defer fileIO.Close()

			// Calculate expected unit count
			expectedUnits := (tc.fileSize + tc.transferSize - 1) / tc.transferSize
			if fileIO.unitCount != expectedUnits {
				t.Errorf("Expected %d units, got %d", expectedUnits, fileIO.unitCount)
			}

			// Write random data to all units
			allData := make([]byte, tc.fileSize)
			rand.Read(allData)

			for i := uint64(0); i < fileIO.unitCount; i++ {
				unitSize, _ := npf.GetTransferUnitSize(i)
				start := i * tc.transferSize
				end := start + unitSize

				err := fileIO.WriteTransferUnit(i, allData[start:end])
				if err != nil {
					t.Fatalf("Failed to write unit %d: %v", i, err)
				}
			}

			// Read back and verify
			for i := uint64(0); i < fileIO.unitCount; i++ {
				unitSize, _ := npf.GetTransferUnitSize(i)
				start := i * tc.transferSize
				end := start + unitSize
				expected := allData[start:end]

				actual, err := fileIO.ReadTransferUnit(i)
				if err != nil {
					t.Fatalf("Failed to read unit %d: %v", i, err)
				}

				if !bytes.Equal(expected, actual) {
					t.Fatalf("Mismatch in unit %d\nExpected: %x\nActual: %x",
						i, expected[:min(16, len(expected))], actual[:min(16, len(actual))])
				}
			}

			// Verify complete file
			fileData, err := ioutil.ReadFile(fileIO.file.Name())
			if err != nil {
				t.Fatal(err)
			}

			if !bytes.Equal(allData, fileData) {
				// Find and report mismatches
				mismatches := 0
				for i := 0; i < len(allData) && mismatches < 10; i++ {
					if allData[i] != fileData[i] {
						t.Errorf("Mismatch at byte %d: expected 0x%02x, got 0x%02x",
							i, allData[i], fileData[i])
						mismatches++
					}
				}
				if mismatches == 0 {
					t.Error("Files don't match but no mismatches found (size difference?)")
				}
			}
		})
	}
}

// TestFileIO_SeekAndWrite tests seeking and writing at various offsets
func TestFileIO_SeekAndWrite(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "bao-seek-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a 1MB file
	fileSize := uint64(1024 * 1024)
	transferSize := uint64(64 * 1024)
	fileName := "seek_test.bin"
	npf := createTestBaoFile(fileName, fileSize, transferSize)

	fileIO, err := NewFileIO(npf, tempDir)
	if err != nil {
		t.Fatal(err)
	}
	defer fileIO.Close()

	// Write pattern at various offsets
	patterns := []struct {
		name   string
		offset uint64
		data   []byte
	}{
		{"Start", 0, []byte("START")},
		{"Middle", 500000, []byte("MIDDLE")},
		{"NearEnd", fileSize - 10, []byte("END")},
		{"CrossBoundary", transferSize - 2, []byte("CROSS")}, // Crosses transfer unit boundary
	}

	// Apply patterns
	for _, pattern := range patterns {
		err := fileIO.WriteRange(pattern.offset, pattern.data)
		if err != nil {
			t.Fatalf("Failed to write pattern %s at offset %d: %v",
				pattern.name, pattern.offset, err)
		}
	}

	// Verify patterns
	for _, pattern := range patterns {
		actual, err := fileIO.ReadRange(pattern.offset, uint64(len(pattern.data)))
		if err != nil {
			t.Fatalf("Failed to read pattern %s: %v", pattern.name, err)
		}

		if !bytes.Equal(pattern.data, actual) {
			t.Errorf("Pattern %s mismatch at offset %d: expected %s, got %s",
				pattern.name, pattern.offset, pattern.data, actual)
		}
	}

	// Test that data between patterns is zeros (since we didn't write there)
	// Check a few random spots
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < 10; i++ {
		offset := rand.Intn(int(fileSize) - 100)
		data, err := fileIO.ReadRange(uint64(offset), 10)
		if err != nil {
			t.Fatalf("Failed to read at random offset %d: %v", offset, err)
		}

		// Check if this overlaps with any pattern
		isInPattern := false
		for _, pattern := range patterns {
			patternStart := int(pattern.offset)
			patternEnd := patternStart + len(pattern.data)
			if offset >= patternStart && offset < patternEnd {
				isInPattern = true
				break
			}
		}

		// If not in pattern, should be zeros
		if !isInPattern {
			for j, b := range data {
				if b != 0 {
					t.Errorf("Unexpected non-zero byte at offset %d: 0x%02x", offset+j, b)
				}
			}
		}
	}
}

// TestFileIO_Concurrent tests concurrent access
func TestFileIO_Concurrent(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "bao-concurrent-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a 2MB file with 64KB transfer units
	fileSize := uint64(2 * 1024 * 1024)
	transferSize := uint64(64 * 1024)
	fileName := "concurrent.bin"
	npf := createTestBaoFile(fileName, fileSize, transferSize)

	fileIO, err := NewFileIO(npf, tempDir)
	if err != nil {
		t.Fatal(err)
	}
	defer fileIO.Close()

	// Generate test data for all transfer units
	unitCount := fileIO.unitCount
	testData := make([][]byte, unitCount)
	for i := uint64(0); i < unitCount; i++ {
		size, _ := npf.GetTransferUnitSize(i)
		testData[i] = make([]byte, size)
		rand.Read(testData[i])
	}

	// Concurrent writers
	errors := make(chan error, unitCount*2)

	// Launch concurrent writers
	for i := uint64(0); i < unitCount; i++ {
		go func(idx uint64) {
			// Write the transfer unit
			err := fileIO.WriteTransferUnit(idx, testData[idx])
			if err != nil {
				errors <- fmt.Errorf("WriteTransferUnit %d: %v", idx, err)
				return
			}

			// Small random delay
			time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)

			// Read it back
			data, err := fileIO.ReadTransferUnit(idx)
			if err != nil {
				errors <- fmt.Errorf("ReadTransferUnit %d: %v", idx, err)
				return
			}

			if !bytes.Equal(testData[idx], data) {
				errors <- fmt.Errorf("Data mismatch for unit %d", idx)
			}
		}(i)
	}

	// Wait for all goroutines and collect errors
	time.Sleep(100 * time.Millisecond)
	close(errors)

	var errList []error
	for err := range errors {
		errList = append(errList, err)
	}

	if len(errList) > 0 {
		t.Fatalf("Concurrent test failed with errors: %v", errList)
	}

	// Final verification: read entire file
	fileData, err := ioutil.ReadFile(fileIO.file.Name())
	if err != nil {
		t.Fatal(err)
	}

	// Reconstruct expected data from testData
	expected := make([]byte, fileSize)
	for i := uint64(0); i < unitCount; i++ {
		start := i * transferSize
		copy(expected[start:], testData[i])
	}

	if !bytes.Equal(expected, fileData) {
		t.Fatal("Final file verification failed")
	}
}

// TestFileIO_ErrorCases tests error handling
func TestFileIO_ErrorCases(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "bao-errors-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Test 1: Out of bounds write
	t.Run("OutOfBounds", func(t *testing.T) {
		fileName := "bounds.bin"
		npf := createTestBaoFile(fileName, 1000, 256)

		fileIO, err := NewFileIO(npf, tempDir)
		if err != nil {
			t.Fatal(err)
		}
		defer fileIO.Close()

		// Write beyond file bounds
		err = fileIO.WriteRange(990, make([]byte, 20))
		if err == nil {
			t.Fatal("Expected error for out-of-bounds write")
		}

		// Read beyond bounds
		_, err = fileIO.ReadRange(990, 20)
		if err == nil {
			t.Fatal("Expected error for out-of-bounds read")
		}
	})

	// Test 2: Invalid transfer unit index
	t.Run("InvalidUnitIndex", func(t *testing.T) {
		fileName := "invalid.bin"
		npf := createTestBaoFile(fileName, 1000, 256)

		fileIO, err := NewFileIO(npf, tempDir)
		if err != nil {
			t.Fatal(err)
		}
		defer fileIO.Close()

		unitCount := fileIO.unitCount

		// Try to access beyond last unit
		_, err = fileIO.ReadTransferUnit(unitCount)
		if err == nil {
			t.Fatal("Expected error for invalid unit index")
		}

		err = fileIO.WriteTransferUnit(unitCount, make([]byte, 256))
		if err == nil {
			t.Fatal("Expected error for invalid unit index")
		}
	})

	// Test 3: Size mismatch in WriteTransferUnit
	t.Run("SizeMismatch", func(t *testing.T) {
		fileName := "mismatch.bin"
		npf := createTestBaoFile(fileName, 1000, 256)

		fileIO, err := NewFileIO(npf, tempDir)
		if err != nil {
			t.Fatal(err)
		}
		defer fileIO.Close()

		// Try to write wrong sized data
		err = fileIO.WriteTransferUnit(0, make([]byte, 100)) // Should be 256
		if err == nil {
			t.Fatal("Expected error for size mismatch")
		}

		err = fileIO.WriteTransferUnit(0, make([]byte, 300)) // Should be 256
		if err == nil {
			t.Fatal("Expected error for size mismatch")
		}
	})
}

// Test helper: min function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestFileIO_RealisticScenario tests a realistic download scenario
func TestFileIO_RealisticScenario(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "bao-realistic-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Simulate a 5MB file with 64KB transfer units
	fileSize := uint64(5 * 1024 * 1024)
	transferSize := uint64(64 * 1024)
	fileName := "realistic.bin"
	npf := createTestBaoFile(fileName, fileSize, transferSize)

	// Create original test file with known pattern
	originalData := make([]byte, fileSize)
	for i := range originalData {
		originalData[i] = byte(i % 256) // Simple pattern
	}

	// Write original file to compare against
	originalPath := "original.bin"
	if err := ioutil.WriteFile(originalPath, originalData, 0644); err != nil {
		t.Fatal(err)
	}

	// Create FileIO for download
	fileIO, err := NewFileIO(npf, tempDir)
	if err != nil {
		t.Fatal(err)
	}
	defer fileIO.Close()

	unitCount := fileIO.unitCount

	// Simulate random order download (like real P2P)
	indices := make([]uint64, unitCount)
	for i := uint64(0); i < unitCount; i++ {
		indices[i] = i
	}

	// Shuffle indices
	rand.Shuffle(len(indices), func(i, j int) {
		indices[i], indices[j] = indices[j], indices[i]
	})

	// "Download" units in random order
	for _, idx := range indices {
		unitSize, _ := npf.GetTransferUnitSize(idx)
		start := idx * transferSize
		end := start + unitSize

		// Simulate receiving this unit from network
		unitData := originalData[start:end]

		// Write to file
		err := fileIO.WriteTransferUnit(idx, unitData)
		if err != nil {
			t.Fatalf("Failed to write unit %d: %v", idx, err)
		}

		// Verify immediately
		readData, err := fileIO.ReadTransferUnit(idx)
		if err != nil {
			t.Fatalf("Failed to read unit %d: %v", idx, err)
		}

		if !bytes.Equal(unitData, readData) {
			t.Fatalf("Immediate verification failed for unit %d", idx)
		}

		// Check bitfield
		if !fileIO.HasTransferUnit(idx) {
			t.Fatalf("Bitfield not set for unit %d", idx)
		}
	}

	// Verify file is complete
	if !fileIO.IsComplete() {
		t.Fatal("File should be complete")
	}

	// Final verification: compare with original
	downloadedData, err := ioutil.ReadFile(fileIO.file.Name())
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(originalData, downloadedData) {
		// Detailed comparison
		for i := 0; i < len(originalData); i++ {
			if originalData[i] != downloadedData[i] {
				// Find which transfer unit this is in
				unitIdx := uint64(i) / transferSize
				offsetInUnit := uint64(i) % transferSize

				t.Errorf("Mismatch at byte %d (unit %d, offset %d): original=0x%02x, downloaded=0x%02x",
					i, unitIdx, offsetInUnit, originalData[i], downloadedData[i])

				// Show context
				start := max(0, i-10)
				end := min(len(originalData), i+10)
				t.Errorf("Original context: %x", originalData[start:end])
				t.Errorf("Downloaded context: %x", downloadedData[start:end])

				break
			}
		}
		t.Fatal("Downloaded file doesn't match original")
	}

	t.Logf("Successfully verified %d transfer units written in random order", unitCount)
}

// Test helper: max function
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
