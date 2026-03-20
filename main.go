package main

import (
	"fmt"
	"log"
	"os"
	"piecewise/internal/p2p"
	"piecewise/internal/torrent"
)

func main() {
	// meta, err := torrent.ReadFrom("internal/torrent/testdata/debian-13.3.0-amd64-DVD-1.iso.torrent") // compact str torrent
	meta, err := torrent.ReadFrom("./debian-13.4.0-amd64-netinst.iso.torrent")
	if err != nil {
		log.Fatalf("couldn't read torrent file: %v", err)
	}
	fmt.Println(meta)

	peerId := torrent.MakePeerId()

	trackerURL, err := torrent.BuildTrackerURL(&meta, peerId)
	if err != nil {
		log.Fatalf("coudln't build tracker url: %v", err)
	}
	fmt.Println("requesting peers from: ", meta.Announce)

	trackerRes, err := torrent.RequestPeers(trackerURL)
	if err != nil {
		log.Fatalf("couldn't request peers: %v", err)
	}

	// peers, err := torrent.UnmarshalPeers([]byte(trackerRes.Peers))
	peers, err := torrent.UnmarshalPeers(trackerRes.Peers)
	if err != nil {
		log.Fatalf("couuldn't unmarshall peers: %v", err)
	}

	fmt.Printf("found %d seeing Debian\n", len(peers))

	for i := 0; i < min(100, len(peers)); i++ {
		fmt.Printf("\tPeer %d: %s:%d\n", i+1, peers[i].IP, peers[i].Port)
	}

	// workQueue := p2p.InitWorkQueue(&meta)
	tracker := p2p.NewPieceTracker(len(meta.PieceHashes))
	results := make(chan *p2p.PieceResult)

	outFile, err := os.Create(meta.Name)
	if err != nil {
		log.Fatalf("couldn't create output file: %v", err)
	}
	defer outFile.Close()

	for _, peer := range peers {
		// go p2p.StartWorker(peer, meta.InfoHash, peerId, workQueue, results)
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
			fmt.Printf("\npiece %d got saved!!!!!!! (%d / %d)\n", res.Index, done+1, len(meta.PieceHashes))
		} else {
			fmt.Printf("\nignoring duplicate piece %d\n", res.Index)
		}
	}

	// for done := 0; done < len(meta.PieceHashes); done++ {
	// 	res := <-results
	// 	offset := int64(res.Index * meta.PieceLength)
	//
	// 	_, err = outFile.WriteAt(res.Buf, offset)
	// 	if err != nil {
	// 		log.Fatalf("couldn't write piece %d to disk : %v", res.Index, err)
	// 	}
	//
	// 	tracker.MarkDownloaded(res.Index)
	// 	fmt.Printf("\npiece %d got saved!!!!!!! (%d / %d)\n", res.Index, done+1, len(meta.PieceHashes))
	// }

	fmt.Println("\nwhoop whoop just downloaded an entire file!!!!")
}
