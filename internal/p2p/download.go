package p2p

import (
	"fmt"
	"math"
	"piecewise/internal/torrent"
	"time"
)

// a job in the work queue
type PieceWork struct {
	index  int
	hash   [20]byte
	length int
}

// verified downloaded peice
type PieceResult struct {
	Index int
	Buf   []byte
}

func InitWorkQueue(meta *torrent.TorrentMeta) chan *PieceWork {
	numPieces := len(meta.PieceHashes)
	workQueue := make(chan *PieceWork, numPieces)

	for i := range numPieces {
		length := meta.PieceLength

		// if its the last peice, we set length to the leftover bytes
		if i == numPieces-1 {
			remaining := meta.Length % meta.PieceLength
			if remaining != 0 {
				length = remaining
			}
		}

		workQueue <- &PieceWork{
			index:  i,
			hash:   meta.PieceHashes[i],
			length: length,
		}
	}

	return workQueue
}

func StartWorker(peer torrent.Peer, infoHash [20]byte, peerId [20]byte, workQueue chan *PieceWork, results chan *PieceResult) {
	// keeping the client nil to start so it'll directly go into the reconnect branch
	// don't know if its better to just initialize it here. might change it later
	var client *Client
	var fails int
	const maxFails = 5

	for work := range workQueue {
		if client == nil {
			if fails >= maxFails {
				fmt.Printf("peer %s gone bye bye. killing worker\n", peer.IP)
				workQueue <- work
				return
			}

			conn, err := DialPeer(peer.IP, peer.Port, infoHash, peerId)
			if err != nil {
				fails++
				backoff := time.Duration(math.Pow(2, float64(fails))) * time.Second

				fmt.Printf("couldn't to connect to %s | retrying in %v\n", peer.IP, backoff)
				workQueue <- work
				time.Sleep(backoff)
				continue
			}

			fails = 0
			client = &Client{
				Conn:   conn,
				Choked: true,
			}
		}

		buf, err := client.ReadLoop(work)
		if err != nil {
			fails++
			fmt.Printf("peer %s dropped piece %d | requeueing\n", peer.IP, work.index)
			workQueue <- work

			client.Conn.Close()
			client = nil

			// probably fine to hardcode this small sleep before retrying
			// TODO: read what other clients do in these situations
			time.Sleep(3 * time.Second)
			continue
		}

		results <- &PieceResult{
			Index: work.index,
			Buf:   buf,
		}

	}

	if client != nil {
		client.Conn.Close()
	}
}
