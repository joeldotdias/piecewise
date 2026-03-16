package p2p

import (
	"fmt"
	"piecewise/internal/torrent"
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

	for i := 0; i < numPieces; i++ {
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
	conn, err := DialPeer(peer.IP, peer.Port, infoHash, peerId)
	if err != nil {
		return
	}
	defer conn.Close()

	client := Client{
		Conn:   conn,
		Choked: true,
	}

	for work := range workQueue {
		buf, err := client.ReadLoop(work)

		if err != nil {
			fmt.Printf("Whoops peer %s messed up piece %d | Requeueing\n", peer.IP, work.index)
			workQueue <- work
			// gotta kill this worker coz connection is dead
			return
		}

		results <- &PieceResult{
			Index: work.index,
			Buf:   buf,
		}
	}
}
