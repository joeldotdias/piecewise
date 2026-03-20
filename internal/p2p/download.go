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

func StartWorker(peer torrent.Peer, meta *torrent.TorrentMeta, peerId [20]byte, tracker *PieceTracker, results chan *PieceResult) {
	// keeping the client nil to start so it'll directly go into the reconnect branch
	// don't know if its better to just initialize it here. might change it later
	var client *Client
	var fails int
	const maxFails = 5

	for {
		if tracker.IsDone() {
			if client != nil {
				client.Conn.Close()
			}
			return
		}

		if client == nil {
			if fails >= maxFails {
				fmt.Printf("peers %s gone bye bye. killing worker\n", peer.IP)
				return
			}

			conn, err := DialPeer(peer.IP, peer.Port, meta.InfoHash, peerId)
			if err != nil {
				fails++
				backoff := time.Duration(math.Pow(2, float64(fails))) * time.Second

				fmt.Printf("couldn't to connect to %s | retrying in %v\n", peer.IP, backoff)
				time.Sleep(backoff)
				continue
			}

			fails = 0
			client = &Client{
				Conn:   conn,
				Choked: true,
			}

			err = client.ReadInitialState(tracker)
			if err != nil {
				client.Conn.Close()
				client = nil
				continue
			}
		}

		idx, ok := tracker.PickNextPiece(client.Bitfield)
		if !ok {
			// tiny sleep so we don't keep spamming cpu
			time.Sleep(1 * time.Second)
			continue
		}

		length := meta.PieceLength
		if idx == len(meta.PieceHashes)-1 {
			leftover := meta.Length % meta.PieceLength
			if leftover != 0 {
				length = leftover
			}
		}

		work := &PieceWork{
			index:  idx,
			hash:   meta.PieceHashes[idx],
			length: length,
		}

		buf, err := client.ReadLoop(work, tracker)
		if err != nil {
			fails++
			fmt.Printf("peer %s dropped piece %d | requeueing\n", peer.IP, work.index)
			tracker.MarkFailed(work.index)

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
}

// keeping this here to remember the bad performance
// func DumbWorkQueue(peer torrent.Peer, infoHash [20]byte, peerId [20]byte, workQueue chan *PieceWork, results chan *PieceResult) {
// 	var client *Client
// 	var fails int
// 	const maxFails = 5
//
// 	for work := range workQueue {
// 		if client == nil {
// 			if fails >= maxFails {
// 				fmt.Printf("peer %s gone bye bye. killing worker\n", peer.IP)
// 				workQueue <- work
// 				return
// 			}
//
// 			conn, err := DialPeer(peer.IP, peer.Port, infoHash, peerId)
// 			if err != nil {
// 				fails++
// 				backoff := time.Duration(math.Pow(2, float64(fails))) * time.Second
//
// 				fmt.Printf("couldn't to connect to %s | retrying in %v\n", peer.IP, backoff)
// 				workQueue <- work
// 				time.Sleep(backoff)
// 				continue
// 			}
//
// 			fails = 0
// 			client = &Client{
// 				Conn:   conn,
// 				Choked: true,
// 			}
// 		}
//
// 		buf, err := client.ReadLoop(work)
// 		if err != nil {
// 			fails++
// 			fmt.Printf("peer %s dropped piece %d | requeueing\n", peer.IP, work.index)
// 			workQueue <- work
//
// 			client.Conn.Close()
// 			client = nil
//
// 			time.Sleep(3 * time.Second)
// 			continue
// 		}
//
// 		results <- &PieceResult{
// 			Index: work.index,
// 			Buf:   buf,
// 		}
//
// 	}
//
// 	if client != nil {
// 		client.Conn.Close()
// 	}
// }
