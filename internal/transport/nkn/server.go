package nkntransport

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/baoswarm/baobun/internal/core"
	"github.com/baoswarm/baobun/internal/debugs"
	"github.com/baoswarm/baobun/pkg/protocol"
	nkn "github.com/nknorg/nkn-sdk-go"
)

type Transport struct {
	client   *nkn.MultiClient
	Sessions *core.SessionManager
}

func NewTransport(client *nkn.MultiClient) *Transport {
	<-client.OnConnect.C

	// Accept sessions from any peer
	if err := client.Listen(nil); err != nil {
		panic(err)
	}

	log.Println("Listening at", client.Addr())

	sm := core.NewSessionManager(client)

	return &Transport{
		client:   client,
		Sessions: sm,
	}
}

func (t *Transport) Announce(
	ctx context.Context,
	address string,
	req protocol.AnnounceRequest,
) (protocol.AnnounceResponse, error) {

	data, err := json.Marshal(req)
	if err != nil {
		return protocol.AnnounceResponse{}, err
	}

	log.Println("Sending announcement")
	reply, err := t.client.Send(
		nkn.NewStringArray(address),
		data,
		nil,
	)
	if err != nil {
		debugs.ConnectedToTracker = false
		return protocol.AnnounceResponse{}, err
	}
	resp := <-reply.C
	if len(resp.Data) == 0 {
		debugs.ConnectedToTracker = false
		return protocol.AnnounceResponse{}, fmt.Errorf("no reply from tracker")
	}
	debugs.ConnectedToTracker = true
	log.Println("Announcement reply received...")

	var out protocol.AnnounceResponse
	if err := json.Unmarshal(resp.Data, &out); err != nil {
		return protocol.AnnounceResponse{}, err
	}

	return out, nil
}

func (t *Transport) Close() {
	t.client.Close()
	log.Println("Client closed...")
}
