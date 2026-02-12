package api

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/crypto/scrypt"
)

const hiddenStoreVersion = 1

var ErrInvalidPasskey = errors.New("invalid passkey")

type HiddenStore struct {
	mu   sync.Mutex
	path string
	data hiddenStoreDisk
}

type hiddenStoreDisk struct {
	Version int               `json:"version"`
	Items   []hiddenStoreItem `json:"items"`
}

type hiddenStoreItem struct {
	ID         string `json:"id"`
	Salt       string `json:"salt"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
	HiddenAt   int64  `json:"hidden_at_unix"`
}

type HiddenPayload struct {
	InfoHash     string `json:"info_hash"`
	FileLocation string `json:"file_location"`
	BaoJSON      []byte `json:"bao_json"`
}

func NewHiddenStore(path string) (*HiddenStore, error) {
	store := &HiddenStore{
		path: path,
		data: hiddenStoreDisk{
			Version: hiddenStoreVersion,
			Items:   make([]hiddenStoreItem, 0),
		},
	}

	if err := store.load(); err != nil {
		return nil, err
	}

	return store, nil
}

func (s *HiddenStore) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.data.Items)
}

func (s *HiddenStore) Hide(payload HiddenPayload, passkey string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if passkey == "" {
		return fmt.Errorf("passkey is required")
	}
	if payload.InfoHash == "" {
		return fmt.Errorf("infohash is required")
	}

	plain, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal hidden payload: %w", err)
	}

	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return fmt.Errorf("failed to generate salt: %w", err)
	}

	key, err := deriveHiddenKey(passkey, salt)
	if err != nil {
		return err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("failed to create gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plain, nil)
	record := hiddenStoreItem{
		ID:         payload.InfoHash,
		Salt:       base64.StdEncoding.EncodeToString(salt),
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
		HiddenAt:   time.Now().Unix(),
	}

	s.upsert(record)
	return s.save()
}

func (s *HiddenStore) Unhide(passkey string) ([]HiddenPayload, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if passkey == "" {
		return nil, fmt.Errorf("passkey is required")
	}

	out := make([]HiddenPayload, 0)
	remaining := make([]hiddenStoreItem, 0, len(s.data.Items))

	for _, record := range s.data.Items {
		payload, err := decryptHiddenRecord(record, passkey)
		if err != nil {
			remaining = append(remaining, record)
			continue
		}
		out = append(out, payload)
	}

	if len(out) == 0 && len(s.data.Items) > 0 {
		return nil, ErrInvalidPasskey
	}

	if len(out) > 0 {
		s.data.Items = remaining
		if err := s.save(); err != nil {
			return nil, err
		}
	}

	return out, nil
}

func (s *HiddenStore) upsert(item hiddenStoreItem) {
	for i := range s.data.Items {
		if s.data.Items[i].ID == item.ID {
			s.data.Items[i] = item
			return
		}
	}
	s.data.Items = append(s.data.Items, item)
}

func (s *HiddenStore) load() error {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read hidden store: %w", err)
	}

	var disk hiddenStoreDisk
	if err := json.Unmarshal(raw, &disk); err != nil {
		return fmt.Errorf("failed to decode hidden store: %w", err)
	}

	if disk.Version != hiddenStoreVersion {
		return fmt.Errorf("unsupported hidden store version %d", disk.Version)
	}

	s.data = disk
	if s.data.Items == nil {
		s.data.Items = make([]hiddenStoreItem, 0)
	}

	return nil
}

func (s *HiddenStore) save() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return fmt.Errorf("failed to create hidden store dir: %w", err)
	}

	body, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal hidden store: %w", err)
	}
	body = append(body, '\n')

	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, body, 0644); err != nil {
		return fmt.Errorf("failed to write hidden store temp file: %w", err)
	}

	if err := os.Rename(tmp, s.path); err != nil {
		_ = os.Remove(s.path)
		if errRetry := os.Rename(tmp, s.path); errRetry != nil {
			_ = os.Remove(tmp)
			return fmt.Errorf("failed to finalize hidden store file: %w", errRetry)
		}
	}

	return nil
}

func decryptHiddenRecord(record hiddenStoreItem, passkey string) (HiddenPayload, error) {
	salt, err := base64.StdEncoding.DecodeString(record.Salt)
	if err != nil {
		return HiddenPayload{}, err
	}
	nonce, err := base64.StdEncoding.DecodeString(record.Nonce)
	if err != nil {
		return HiddenPayload{}, err
	}
	ciphertext, err := base64.StdEncoding.DecodeString(record.Ciphertext)
	if err != nil {
		return HiddenPayload{}, err
	}

	key, err := deriveHiddenKey(passkey, salt)
	if err != nil {
		return HiddenPayload{}, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return HiddenPayload{}, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return HiddenPayload{}, err
	}
	if len(nonce) != gcm.NonceSize() {
		return HiddenPayload{}, ErrInvalidPasskey
	}

	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return HiddenPayload{}, ErrInvalidPasskey
	}

	var payload HiddenPayload
	if err := json.Unmarshal(plain, &payload); err != nil {
		return HiddenPayload{}, err
	}

	return payload, nil
}

func deriveHiddenKey(passkey string, salt []byte) ([]byte, error) {
	if passkey == "" {
		return nil, fmt.Errorf("passkey is required")
	}
	return scrypt.Key([]byte(passkey), salt, 1<<15, 8, 1, 32)
}
