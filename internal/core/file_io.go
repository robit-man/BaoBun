// file_io.go
package core

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/baoswarm/baobun/internal/config"
)

// FileIO handles range-based file storage for BaoFile
type FileIO struct {
	npf  *BaoFile
	file *os.File

	// Transfer-unit tracking
	unitCount uint64
	haveUnits Bitfield

	// Synchronization for shared metadata (not file writes)
	mu sync.RWMutex
}

// NewFileIO creates and prepares a file for ranged IO
func NewFileIO(npf *BaoFile, fileLocation string) (*FileIO, error) {
	if npf == nil {
		return nil, errors.New("BaoFile cannot be nil")
	}

	tuCount := npf.GetTransferUnitCount()

	f := &FileIO{
		npf:       npf,
		unitCount: tuCount,
		haveUnits: NewBitfield(tuCount),
	}

	if err := f.initializeFile(fileLocation); err != nil {
		return nil, err
	}

	return f, nil
}

func (f *FileIO) initializeFile(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	fullPath := filepath.Join(dir, f.npf.Name)

	file, err := os.OpenFile(
		fullPath,
		os.O_RDWR|os.O_CREATE,
		0644,
	)
	if err != nil {
		return err
	}

	f.file = file

	info, err := file.Stat()
	if err != nil {
		return err
	}

	if info.Size() != int64(f.npf.Length) {
		if err := file.Truncate(int64(f.npf.Length)); err != nil {
			return err
		}
	}

	return nil
}

// ReadRange reads an arbitrary byte range (concurrency-safe)
func (f *FileIO) ReadRange(start, length uint64) ([]byte, error) {
	if start+length > f.npf.Length {
		return nil, fmt.Errorf("range out of bounds")
	}

	buf := make([]byte, length)
	n, err := f.file.ReadAt(buf, int64(start))
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, err
	}

	return buf[:n], nil
}

// WriteRange writes data at an arbitrary byte offset (concurrency-safe)
func (f *FileIO) WriteRange(start uint64, data []byte) error {
	if start+uint64(len(data)) > f.npf.Length {
		return fmt.Errorf("range out of bounds")
	}

	// Safe to write concurrently using WriteAt
	n, err := f.file.WriteAt(data, int64(start))
	if err != nil {
		return err
	}
	if n != len(data) {
		return io.ErrShortWrite
	}

	return nil
}

// ReadTransferUnit reads a full transfer unit by index
func (f *FileIO) ReadTransferUnit(index uint64) ([]byte, error) {
	if index >= f.unitCount {
		return nil, fmt.Errorf("transfer unit out of range")
	}

	start := index * uint64(config.TransferUnitSize)
	size, _ := f.npf.GetTransferUnitSize(index)
	return f.ReadRange(start, size)
}

// WriteTransferUnit writes a full transfer unit by index
func (f *FileIO) WriteTransferUnit(index uint64, data []byte) error {
	if index >= f.unitCount {
		return fmt.Errorf("transfer unit out of range")
	}

	expectedSize, _ := f.npf.GetTransferUnitSize(index)
	if uint64(len(data)) != expectedSize {
		return fmt.Errorf("transfer unit size mismatch")
	}

	start := index * uint64(config.TransferUnitSize)
	if err := f.WriteRange(start, data); err != nil {
		return err
	}

	// Only lock for updating shared metadata
	f.mu.Lock()
	f.haveUnits.Set(index)
	f.mu.Unlock()

	return nil
}

// HasTransferUnit returns whether a transfer unit is present
func (f *FileIO) HasTransferUnit(index uint64) bool {
	if index >= f.unitCount {
		return false
	}

	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.haveUnits.Has(index)
}

// GetBitfield returns a copy of the have-units bitfield
func (f *FileIO) GetBitfield() Bitfield {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return BitfieldFromBytes(f.haveUnits.Bytes())
}

// IsComplete returns true if all units are present
func (f *FileIO) IsComplete() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.haveUnits.AllSet(f.unitCount)
}

// Sync flushes all file changes to disk
func (f *FileIO) Sync() error {
	return f.file.Sync()
}

// Switch to read only
func (f *FileIO) SwitchToReadOnly() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	originalPath := f.file.Name()

	err := f.Close()
	if err != nil {
		return err
	}

	file, err := os.Open(originalPath)
	if err != nil {
		return err
	}
	f.file = file

	return nil
}

// Close closes the underlying file
func (f *FileIO) Close() error {
	if f.file == nil {
		return nil
	}
	err := f.file.Close()
	f.file = nil
	return err
}

// GetFileInfo returns information about the underlying BaoFile
func (f *FileIO) GetFileInfo() *BaoFile {
	return f.npf
}
