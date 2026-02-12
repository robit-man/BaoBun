// internal/api/adapter.go
package api

import (
	"fmt"

	"github.com/baoswarm/baobun/internal/core"
)

type Adapter struct {
	client *core.Client
}

func NewAdapter(c *core.Client) *Adapter {
	return &Adapter{client: c}
}

func (a *Adapter) Torrents() []TorrentStatus {
	coreTorrents := a.client.Swarms

	out := make([]TorrentStatus, 0, len(coreTorrents))
	for _, t := range coreTorrents {
		remaining := t.CalcLeft()
		downloaded := t.File.Length - remaining
		uploaded := t.Uploaded

		ratio := 0.0
		if downloaded > 0 {
			ratio = float64(uploaded) / float64(downloaded)
		}

		downrate := uint32(0)
		uprate := uint32(0)

		record := TorrentStatus{
			ID:         fmt.Sprintf("%x", t.InfoHash),
			Name:       t.File.Name,
			DownRate:   uint32(downrate),
			UpRate:     uint32(uprate),
			Downloaded: downloaded,
			Uploaded:   uploaded,
			Ratio:      ratio,
			Peers:      make([]PeerStatus, 0),
			State:      mapState(t),
			FileSize:   t.File.Length,
			Remaining:  remaining,
			Files: []FileStatus{
				{
					Path:      t.File.Name,
					Length:    t.File.Length,
					Remaining: remaining,
				},
			},
		}

		for _, p := range t.Peers {
			peerstatus := PeerStatus{
				ID:       string(p.Peer),
				DownRate: p.UploadRate(), //flipped because if a peer is uploading to us, we are downloading.
				UpRate:   p.DownloadRate(),
			}

			downrate += peerstatus.DownRate
			uprate += peerstatus.UpRate

			switch p.GetState() {
			case 0:
				peerstatus.State = "connecting"
			case 1:
				peerstatus.State = "handshake"
			case 2:
				peerstatus.State = "active"
			case 3:
				peerstatus.State = "closed"
			}

			record.Peers = append(record.Peers, peerstatus)
		}

		record.DownRate = downrate
		record.UpRate = uprate

		out = append(out, record)
	}

	return out
}

func mapState(s *core.Swarm) TorrentState {
	left := s.CalcLeft()
	if left == 0 && len(s.Peers) > 0 {
		return "seeding"
	}
	if left == 0 {
		return StateStopped
	}
	if len(s.Peers) == 0 {
		return StateStopped
	}
	return "downloading"
}
