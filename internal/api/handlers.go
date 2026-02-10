// internal/api/handlers.go
package api

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/baoswarm/baobun/internal/core"
	"github.com/baoswarm/baobun/pkg/protocol"
)

type Server struct {
	api        *Adapter
	coreClient *core.Client
}

func NewServer(api *Adapter, core *core.Client) *Server {
	return &Server{
		api:        api,
		coreClient: core,
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
