// internal/api/adapter.go
package api

import "github.com/baoswarm/baobun/internal/core"

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

		downrate := uint32(0)
		uprate := uint32(0)

		record := TorrentStatus{
			ID:        "id",
			Name:      t.File.Name,
			DownRate:  uint32(downrate),
			UpRate:    uint32(uprate),
			State:     mapState(t),
			FileSize:  t.File.Length,
			Remaining: t.CalcLeft(),
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
	if s.CalcLeft() == 0 {
		return "seeding"
	}
	if len(s.Peers) == 0 {
		return "stalled"
	}
	return "downloading"
}
