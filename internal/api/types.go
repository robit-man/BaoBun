// internal/api/types.go
package api

type TorrentState string

const (
	StateDownloading TorrentState = "downloading"
	StateSeeding     TorrentState = "seeding"
	StatePaused      TorrentState = "paused"
	StateQueued      TorrentState = "queued"
	StateError       TorrentState = "error"
)

type PeerState string

const (
	StateOffline PeerState = "offline"
	StateOnline  PeerState = "online"
)

type TorrentStatus struct {
	ID        string       `json:"id"`
	Name      string       `json:"name"`
	DownRate  uint32       `json:"downRate"` // bytes/sec
	UpRate    uint32       `json:"upRate"`
	Peers     []PeerStatus `json:"peers"`
	State     TorrentState `json:"state"`
	FileSize  uint64       `json:"fileSize"`
	Remaining uint64       `json:"remaining"`
}

type PeerStatus struct {
	ID       string    `json:"id"`
	State    PeerState `json:"state"`
	DownRate uint32    `json:"downRate"`
	UpRate   uint32    `json:"upRate"`
}
