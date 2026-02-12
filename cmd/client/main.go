package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/baoswarm/baobun/internal/api"
	"github.com/baoswarm/baobun/internal/core"
	nkntransport "github.com/baoswarm/baobun/internal/transport/nkn"
	"github.com/baoswarm/baobun/internal/webui"
	"github.com/baoswarm/baobun/pkg/protocol"
	"github.com/nknorg/nkn-sdk-go"
)

func main() {
	// Set the flags for the default logger
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// fmt.Println("Spinning up 3 test clients that all want file X")
	// fmt.Println("Client with file X available will come online after you press enter.")

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting working directory:", err)
		return
	}

	core0 := LaunchCore(filepath.Join(cwd, "downloads_0"), "0mmutsimutsimutsimutsimutsimutsi", true)
	go LaunchWebApp(core0, ":8880")
	core1 := LaunchCore(filepath.Join(cwd, "downloads_1"), "1mmutsimutsimutsimutsimutsimutsi", true)
	go LaunchWebApp(core1, ":8881")
	core2 := LaunchCore(filepath.Join(cwd, "downloads_2"), "2mmutsimutsimutsimutsimutsimutsi", true)
	go LaunchWebApp(core2, ":8882")
	core := LaunchCore(filepath.Join(cwd, "downloads"), "immutsimutsimutsimutsimutsimutsi", true)
	go LaunchWebApp(core, ":8888")

	// Setup signal handling
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		fmt.Println("Shutting down gracefully...")

		core0.Transport.Close()
		core1.Transport.Close()
		core2.Transport.Close()
		core.Transport.Close()

		os.Exit(0)
	}()

	defer core0.Transport.Close()
	defer core1.Transport.Close()
	defer core2.Transport.Close()
	defer core.Transport.Close()

	select {} // block forever
}

func LaunchCore(downloadsLocation string, seed string, loadTest bool) *core.Client {
	// ---------------- Identity ----------------
	account, err := nkn.NewAccount([]byte(seed))
	if err != nil {
		log.Fatal(err)
	}

	client, err := nkn.NewMultiClientV2(
		account,
		"bao",
		&nkn.ClientConfig{
			MultiClientNumClients:     4,
			MultiClientOriginalClient: true,
			WebRTC:                    false,
			SeedRPCServerAddr:         nkn.NewStringArray("http://85.215.219.214:30003"),
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	// ---------------- Transport ----------------
	transport := nkntransport.NewTransport(client)

	// ---------------- Core client ----------------
	coreClient := core.NewClient(client.Address(), transport, transport.Sessions)

	// ---------------- Load .bao ----------------
	if loadTest {
		ih, err := coreClient.ImportBao(
			"./test.bao",
			downloadsLocation,
		)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Loaded swarm %x", ih)

		// ---------------- Announce ----------------
		// Run initial announce in background so startup doesn't block
		// while waiting for tracker responses.
		go coreClient.AnnounceSwarm(
			context.Background(),
			ih,
			protocol.EventStarted,
		)
	}

	//TODO: stagger or find some way to avoid this being a massive burst of announcements
	reannounceTicker := time.NewTicker(time.Second * 10)
	go func() {
		for {
			select {
			case <-reannounceTicker.C:
				// Execute the task when a tick is received
				coreClient.ReannounceAllSwarms(context.Background())
			}
		}
	}()
	return coreClient
}

func LaunchWebApp(core *core.Client, addr string) {
	apiAdapter := api.NewAdapter(core)
	apiServer := api.NewServer(apiAdapter, core)

	mux := http.NewServeMux()

	// API
	mux.HandleFunc("/api/v1/torrents", apiServer.HandleTorrents)
	mux.HandleFunc("/api/v1/bao", apiServer.UploadBao)

	// UI
	mux.Handle("/", webui.Handler())

	log.Println("listening on", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
