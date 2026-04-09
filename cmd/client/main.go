package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"piecewise/internal/p2p"
	"piecewise/internal/torrent"
)

func main() {
	torrentPath := flag.String("torrent", "", "path to .torrent file")
	seed := flag.Bool("seed", false, "start the tcp server to upload to peers")
	port := flag.Int("port", 6881, "port to listen on for incoming peers")

	flag.Parse()

	if *torrentPath == "" {
		log.Fatal("gotta provide that torrent path with --torrent flag")
	}

	// meta, err := torrent.ReadFrom("./debian-13.4.0-amd64-netinst.iso.torrent")
	meta, err := torrent.ReadFrom(*torrentPath)
	if err != nil {
		log.Fatalf("couldn't read torrent file: %v", err)
	}
	fmt.Println(meta)

	peerId := torrent.MakePeerId()

	trackerURL, err := torrent.BuildTrackerURL(&meta, peerId, *port)
	if err != nil {
		log.Fatalf("coudln't build tracker url: %v", err)
	}
	fmt.Println("requesting peers from: ", meta.Announce)

	trackerRes, err := torrent.RequestPeers(trackerURL)
	if err != nil {
		log.Fatalf("couldn't request peers: %v", err)
	}

	peers, err := torrent.UnmarshalPeers(trackerRes.Peers)
	if err != nil {
		log.Fatalf("couuldn't unmarshall peers: %v", err)
	}

	fmt.Printf("found %d peers\n", len(peers))

	for i := 0; i < min(100, len(peers)); i++ {
		fmt.Printf("\tPeer %d: %s:%d\n", i+1, peers[i].IP, peers[i].Port)
	}

	// workQueue := p2p.InitWorkQueue(&meta)
	tracker := p2p.NewPieceTracker(len(meta.PieceHashes))

	// outFile, err := os.Create(meta.Name)
	outFile, err := os.OpenFile(meta.Name, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Fatalf("couldn't open output file: %v", err)
	}
	defer outFile.Close()

	results := make(chan *p2p.PieceResult)

	if *seed {
		for i := range meta.PieceHashes {
			tracker.MarkDownloaded(i)
		}
		fmt.Println("initaliazed master seed, fully loaded into tracker")

		trackerUrl, err := torrent.BuildTrackerURL(&meta, peerId, *port)
		if err != nil {
			log.Fatalf("couldn't build tracker url: %v", err)
		}
		_, err = torrent.RequestPeers(trackerUrl)
		if err != nil {
			log.Fatalf("couldn't announce to tracker: %v", err)
		}
		fmt.Printf("announced to tracker at %s\n", meta.Announce)

		go func() {
			seeder := p2p.Seeder{
				Port:    *port,
				PeerId:  peerId,
				Meta:    &meta,
				Tracker: &tracker,
				Store:   outFile,
			}
			// err := p2p.StartServer(*port, meta.InfoHash, peerId, &tracker, outFile)
			err := seeder.Start()
			if err != nil {
				log.Fatalf("server crashed: %v", err)
			}
		}()

		select {}
	}

	for _, peer := range peers {
		go p2p.StartWorker(peer, &meta, peerId, &tracker, results)
	}

	done := 0
	for done < len(meta.PieceHashes) {
		res := <-results

		if tracker.MarkDownloaded(res.Index) {
			offset := int64(res.Index * meta.PieceLength)

			_, err = outFile.WriteAt(res.Buf, offset)
			if err != nil {
				log.Fatalf("couldn't write piece %d to disk : %v", res.Index, err)
			}

			done++
			fmt.Printf("\npiece %d got saved!!!!!!! (%d / %d)\n", res.Index, done, len(meta.PieceHashes))
		} else {
			fmt.Printf("\nignoring duplicate piece %d\n", res.Index)
		}
	}

	fmt.Println("\nwhoop whoop just downloaded an entire file!!!!")
}
