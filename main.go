package main

import (
	"fmt"
	"log"
	"net"
	"piecewise/internal/p2p"
	"piecewise/internal/torrent"
)

func main() {
	meta, err := torrent.ReadFrom("internal/torrent/testdata/debian-13.3.0-amd64-DVD-1.iso.torrent")
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

	peers, err := torrent.UnmarshalPeers([]byte(trackerRes.Peers))
	if err != nil {
		log.Fatalf("couuldn't unmarshall peers: %v", err)
	}

	fmt.Printf("found %d seeing Debian\n", len(peers))

	for i := 0; i < min(100, len(peers)); i++ {
		fmt.Printf("\tPeer %d: %s:%d\n", i+1, peers[i].IP, peers[i].Port)
	}

	var activeConn net.Conn

	for _, peer := range peers {
		fmt.Printf("dialing: %s:%d...\n", peer.IP, peer.Port)

		conn, err := p2p.DialPeer(peer.IP, peer.Port, meta.InfoHash, peerId)
		if err != nil {
			fmt.Printf("\t-> failed: %v\n", err)
			continue
		}

		fmt.Printf("\t-> woohoo! handhshake done with %s\n", peer.IP)
		activeConn = conn

		client := p2p.Client{
			Conn:   conn,
			Choked: true,
		}

		err = client.ReadLoop()
		if err != nil {
			fmt.Printf("\t-> peer disconnected: %v\n", err)
		}

		break
	}

	if activeConn == nil {
		log.Fatalf("\ncouldn't connect to any peers")
	} else {
		activeConn.Close()
	}
}
