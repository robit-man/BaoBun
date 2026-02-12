package config

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sync"
)

const (
	SeedLength = 32
	SeedCount  = 4
)

var defaultSeeds = []string{
	"0mmutsimutsimutsimutsimutsimutsi",
	"1mmutsimutsimutsimutsimutsimutsi",
	"2mmutsimutsimutsimutsimutsimutsi",
	"immutsimutsimutsimutsimutsimutsi",
}

type SeedFile struct {
	Seeds []string `json:"seeds"`
}

type SeedStore struct {
	path string

	mu    sync.RWMutex
	seeds []string
}

func NewSeedStore(path string) (*SeedStore, error) {
	store := &SeedStore{
		path: path,
	}

	if err := store.loadOrInit(); err != nil {
		return nil, err
	}

	return store, nil
}

func (s *SeedStore) Seeds() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return cloneSeeds(s.seeds)
}

func (s *SeedStore) SetSeeds(seeds []string) error {
	if err := ValidateSeeds(seeds); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.seeds = cloneSeeds(seeds)
	return s.persistLocked()
}

func (s *SeedStore) GenerateAndSet() ([]string, error) {
	generated, err := GenerateSeeds()
	if err != nil {
		return nil, err
	}

	if err := s.SetSeeds(generated); err != nil {
		return nil, err
	}

	return generated, nil
}

func (s *SeedStore) loadOrInit() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			s.mu.Lock()
			s.seeds = cloneSeeds(defaultSeeds)
			defer s.mu.Unlock()
			return s.persistLocked()
		}
		return fmt.Errorf("failed to read seed config %q: %w", s.path, err)
	}

	var file SeedFile
	if err := json.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("failed to parse seed config %q: %w", s.path, err)
	}

	if err := ValidateSeeds(file.Seeds); err != nil {
		return fmt.Errorf("invalid seed config %q: %w", s.path, err)
	}

	s.mu.Lock()
	s.seeds = cloneSeeds(file.Seeds)
	s.mu.Unlock()

	return nil
}

func (s *SeedStore) persistLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return fmt.Errorf("failed to create seed config directory: %w", err)
	}

	file := SeedFile{
		Seeds: cloneSeeds(s.seeds),
	}

	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize seed config: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(s.path, data, 0644); err != nil {
		return fmt.Errorf("failed to write seed config %q: %w", s.path, err)
	}

	return nil
}

func ValidateSeeds(seeds []string) error {
	if len(seeds) != SeedCount {
		return fmt.Errorf("expected %d seeds, got %d", SeedCount, len(seeds))
	}

	for i, seed := range seeds {
		if len(seed) != SeedLength {
			return fmt.Errorf("seed %d must be %d characters, got %d", i, SeedLength, len(seed))
		}
	}

	return nil
}

func GenerateSeeds() ([]string, error) {
	out := make([]string, SeedCount)
	for i := 0; i < SeedCount; i++ {
		seed, err := generateSeed(SeedLength)
		if err != nil {
			return nil, err
		}
		out[i] = seed
	}
	return out, nil
}

func generateSeed(length int) (string, error) {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	max := big.NewInt(int64(len(chars)))

	buf := make([]byte, length)
	for i := range buf {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", fmt.Errorf("failed to generate random seed: %w", err)
		}
		buf[i] = chars[n.Int64()]
	}

	return string(buf), nil
}

func cloneSeeds(in []string) []string {
	out := make([]string, len(in))
	copy(out, in)
	return out
}
