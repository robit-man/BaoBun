// internal/api/types.go
package api

type TorrentState string

const (
	StateDownloading TorrentState = "downloading"
	StateSeeding     TorrentState = "seeding"
	StateStopped     TorrentState = "stopped"
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
	ID         string       `json:"id"`
	Name       string       `json:"name"`
	DownRate   uint32       `json:"downRate"` // bytes/sec
	UpRate     uint32       `json:"upRate"`
	Downloaded uint64       `json:"downloaded"`
	Uploaded   uint64       `json:"uploaded"`
	Ratio      float64      `json:"ratio"`
	Peers      []PeerStatus `json:"peers"`
	Files      []FileStatus `json:"files"`
	State      TorrentState `json:"state"`
	FileSize   uint64       `json:"fileSize"`
	Remaining  uint64       `json:"remaining"`
}

type FileStatus struct {
	Path      string `json:"path"`
	Length    uint64 `json:"length"`
	Remaining uint64 `json:"remaining"`
}

type PeerStatus struct {
	ID       string    `json:"id"`
	State    PeerState `json:"state"`
	DownRate uint32    `json:"downRate"`
	UpRate   uint32    `json:"upRate"`
}

type UploadBaoResponse struct {
	InfoHash string `json:"infoHash"`
	Name     string `json:"name"`
}

type SeedConfigResponse struct {
	Seeds           []string `json:"seeds"`
	SeedLength      int      `json:"seedLength"`
	SeedCount       int      `json:"seedCount"`
	RestartRequired bool     `json:"restartRequired"`
}

type SeedConfigUpdateRequest struct {
	Seeds []string `json:"seeds"`
}

type TorrentActionRequest struct {
	IDs []string `json:"ids"`
}

type HideTorrentActionRequest struct {
	IDs     []string `json:"ids"`
	Passkey string   `json:"passkey"`
}

type HiddenPasskeyRequest struct {
	Passkey string `json:"passkey"`
}

type TorrentActionResponse struct {
	Processed  int    `json:"processed"`
	Hidden     int    `json:"hidden"`
	Remaining  int    `json:"remaining"`
	Successful bool   `json:"successful"`
	Message    string `json:"message"`
}

type HiddenCountResponse struct {
	Count int `json:"count"`
}
