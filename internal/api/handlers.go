// internal/api/handlers.go
package api

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"

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

	ih, err := s.coreClient.ImportBao(
		"./test.bao",
		"./webclient/",
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Loaded swarm %x", ih)

	// ---------------- Announce ----------------
	s.coreClient.AnnounceSwarm(
		context.Background(),
		ih,
		protocol.EventStarted,
	)

	w.WriteHeader(http.StatusOK)
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
