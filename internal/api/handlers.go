// internal/api/handlers.go
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	appconfig "github.com/baoswarm/baobun/internal/config"
	"github.com/baoswarm/baobun/internal/core"
	"github.com/baoswarm/baobun/pkg/protocol"
)

type Server struct {
	api        *Adapter
	coreClient *core.Client
	seedStore  *appconfig.SeedStore
}

func NewServer(api *Adapter, core *core.Client, seedStore *appconfig.SeedStore) *Server {
	return &Server{
		api:        api,
		coreClient: core,
		seedStore:  seedStore,
	}
}

func (s *Server) HandleTorrents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	torrents := s.api.Torrents()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(torrents)
}

func (s *Server) UploadBao(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

	// Read all binary data from POST body
	dataFromPost, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	if len(dataFromPost) == 0 {
		http.Error(w, "empty upload", http.StatusBadRequest)
		return
	}

	downloadDir := s.resolveDownloadDir()

	// Try treating upload as a .bao descriptor first.
	ih, err := s.coreClient.ImportBaoData(dataFromPost, downloadDir)
	if err != nil {
		// If not a .bao descriptor, treat upload as a raw file and generate a .bao.
		// Older frontends may not send X-Filename, so keep a safe fallback name.
		fileName := decodeUploadFilename(r.Header.Get("X-Filename"))
		if fileName == "" {
			fileName = decodeUploadFilename(r.URL.Query().Get("filename"))
		}
		if fileName == "" {
			fileName = "upload.bin"
		}

		if err := os.MkdirAll(downloadDir, 0755); err != nil {
			http.Error(w, fmt.Sprintf("failed to create download dir: %v", err), http.StatusInternalServerError)
			return
		}

		targetPath := uniqueUploadPath(downloadDir, fileName)
		if writeErr := os.WriteFile(targetPath, dataFromPost, 0644); writeErr != nil {
			http.Error(w, fmt.Sprintf("failed to store uploaded file: %v", writeErr), http.StatusInternalServerError)
			return
		}

		baoFile, createErr := core.CreateFromFile(targetPath, s.resolveTrackers())
		if createErr != nil {
			http.Error(w, fmt.Sprintf("failed to create bao metadata: %v", createErr), http.StatusInternalServerError)
			return
		}

		ih, err = s.coreClient.ImportBaoFile(baoFile, downloadDir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Raw file upload is complete data; mark all units as present.
		if swarm, ok := s.coreClient.Swarms[ih]; ok {
			swarm.MarkAllUnitsAvailable()
		}
	}

	log.Printf("Loaded swarm %x", ih)

	// ---------------- Announce ----------------
	s.coreClient.AnnounceSwarm(
		context.Background(),
		ih,
		protocol.EventStarted,
	)

	swarm, ok := s.coreClient.Swarms[ih]
	if !ok {
		http.Error(w, "swarm not found after import", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(UploadBaoResponse{
		InfoHash: fmt.Sprintf("%x", ih),
		Name:     swarm.File.Name,
	})
}

func (s *Server) resolveDownloadDir() string {
	for _, swarm := range s.coreClient.Swarms {
		if swarm.FileLocation != "" {
			return swarm.FileLocation
		}
	}
	return filepath.Clean("./webclient/")
}

func (s *Server) resolveTrackers() []string {
	trackers := make([]string, 0)
	seen := make(map[string]struct{})

	for _, swarm := range s.coreClient.Swarms {
		for _, tracker := range swarm.File.Trackers {
			if _, ok := seen[tracker]; ok {
				continue
			}
			seen[tracker] = struct{}{}
			trackers = append(trackers, tracker)
		}
	}

	if len(trackers) > 0 {
		return trackers
	}

	return append([]string(nil), appconfig.DefaultTrackers...)
}

func decodeUploadFilename(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	if decoded, err := url.QueryUnescape(raw); err == nil {
		raw = decoded
	}

	base := filepath.Base(raw)
	base = strings.ReplaceAll(base, "\x00", "")
	base = strings.TrimSpace(base)
	if base == "" || base == "." || base == string(filepath.Separator) {
		return ""
	}

	return base
}

func uniqueUploadPath(dir, filename string) string {
	ext := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, ext)
	if name == "" {
		name = "upload"
	}

	candidate := filepath.Join(dir, filename)
	if _, err := os.Stat(candidate); os.IsNotExist(err) {
		return candidate
	}

	for i := 1; i < 10000; i++ {
		next := filepath.Join(dir, fmt.Sprintf("%s_%d%s", name, i, ext))
		if _, err := os.Stat(next); os.IsNotExist(err) {
			return next
		}
	}

	return filepath.Join(dir, fmt.Sprintf("%s_%d%s", name, 10000, ext))
}

func (s *Server) HandleSeedConfig(w http.ResponseWriter, r *http.Request) {
	if s.seedStore == nil {
		http.Error(w, "seed store unavailable", http.StatusInternalServerError)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.writeSeedConfig(w)
	case http.MethodPut:
		defer r.Body.Close()

		var req SeedConfigUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}

		if err := s.seedStore.SetSeeds(req.Seeds); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		s.writeSeedConfig(w)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) GenerateSeedConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.seedStore == nil {
		http.Error(w, "seed store unavailable", http.StatusInternalServerError)
		return
	}

	if _, err := s.seedStore.GenerateAndSet(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.writeSeedConfig(w)
}

func (s *Server) writeSeedConfig(w http.ResponseWriter) {
	payload := SeedConfigResponse{
		Seeds:           s.seedStore.Seeds(),
		SeedLength:      appconfig.SeedLength,
		SeedCount:       appconfig.SeedCount,
		RestartRequired: true,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}
