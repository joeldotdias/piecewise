package p2p

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"piecewise/internal/torrent"
)

type Seeder struct {
	Port    int
	PeerId  [20]byte
	Meta    *torrent.TorrentMeta
	Tracker *PieceTracker
	Store   io.ReaderAt
}

func (s *Seeder) Start() error {
	addr := fmt.Sprintf(":%d", s.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("couldn't start seeder on port %d: %w", s.Port, err)
	}

	fmt.Printf("listening for peers on %s\n", addr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("couldn't accept connection: %v\n", err)
			continue
		}

		go s.handshakePeer(conn)
	}
}

func (s *Seeder) handshakePeer(conn net.Conn) {
	defer conn.Close()

	fmt.Printf("incoming from %s\n", conn.RemoteAddr().String())

	reqHandshake, err := ReadHandshake(conn)
	if err != nil {
		fmt.Printf("dropping %s: weird handshake\n", conn.RemoteAddr().String())
		return
	}

	if reqHandshake.InfoHash != s.Meta.InfoHash {
		fmt.Printf("dropping %s: infohash mismatch\n", conn.RemoteAddr().String())
		return
	}

	resHandshake := NewHandshake(s.Meta.InfoHash, s.PeerId)
	_, err = conn.Write(resHandshake.Serialize())
	if err != nil {
		fmt.Printf("dropping %s: didn't send handshake reply\n", conn.RemoteAddr().String())
		return
	}

	bfMsg := Message{
		Id:      MsgBitfield,
		Payload: s.Tracker.GetBitfield(),
	}
	_, err = conn.Write(bfMsg.Serialize())
	if err != nil {
		fmt.Printf("dropping %s: couldn't send bitfield\n", conn.RemoteAddr().String())
		return
	}

	unChokeMsg := Message{Id: MsgUnchoke}
	_, err = conn.Write(unChokeMsg.Serialize())
	if err != nil {
		fmt.Printf("dropping %s: couldn't send unchoke\n", conn.RemoteAddr().String())
		return
	}

	s.serveDataLoop(conn)
}

func (s *Seeder) serveDataLoop(conn net.Conn) {
	defer conn.Close()

	for {
		msg, err := ReadMessage(conn)
		if err != nil {
			fmt.Printf("peer disconnected: %v\n", err)
			return
		}

		// Keep-Alive
		if msg == nil {
			continue
		}

		if msg.Id == MsgRequest {
			if len(msg.Payload) != 12 {
				fmt.Println("malformed request; dropping peer")
				return
			}

			index := int(binary.BigEndian.Uint32(msg.Payload[0:4]))
			if !s.Tracker.IsDownloaded(index) {
				fmt.Printf("peer requested piece %d but it ain't verified yet, skipping\n", index)
				continue
			}

			begin := int(binary.BigEndian.Uint32(msg.Payload[4:8]))
			length := int(binary.BigEndian.Uint32(msg.Payload[8:12]))

			pieceSize := s.Meta.PieceSize(index)

			if length > MaxBlockSize || begin < 0 || length <= 0 || begin+length > pieceSize {
				fmt.Printf("peer sent bad request (index=%d, begin=%d, length=%d), dropping\n", index, begin, length)
				return
			}

			fileOffset := int64((index * s.Meta.PieceLength) + begin)

			blockBuf := make([]byte, length)
			_, err = s.Store.ReadAt(blockBuf, fileOffset)
			if err != nil {
				fmt.Printf("disk read failed: %v\n", err)
				return
			}

			pieceMsg := FormatPiece(index, begin, blockBuf)
			_, err = conn.Write(pieceMsg.Serialize())
			if err != nil {
				fmt.Printf("couldn't send piece: %v", err)
				return
			}
		}
	}
}

// func StartServer(port int, myInfoHash [20]byte, myPeerId [20]byte, tracker *PieceTracker, seedFile io.ReaderAt) error {
// 	addr := fmt.Sprintf(":%d", port)
// 	listener, err := net.Listen("tcp", addr)
// 	if err != nil {
// 		return fmt.Errorf("couldn't start server on port %d: %w", port, err)
// 	}
//
// 	fmt.Printf("listening for peers on %s\n", addr)
//
// 	for {
// 		conn, err := listener.Accept()
// 		if err != nil {
// 			fmt.Printf("couldn't accept connection: %v\n", err)
// 			continue
// 		}
//
// 		go handshakePeer(conn, myInfoHash, myPeerId, tracker, seedFile)
// 	}
// }

// func handshakePeer(conn net.Conn, myInfoHash [20]byte, myPeerId [20]byte, tracker *PieceTracker, seedFile io.ReaderAt) {
// 	defer conn.Close()
//
// 	fmt.Printf("incoming from %s\n", conn.RemoteAddr().String())
//
// 	reqHandshake, err := ReadHandshake(conn)
// 	if err != nil {
// 		fmt.Printf("dropping %s: weird handshake\n", conn.RemoteAddr().String())
// 		return
// 	}
//
// 	if reqHandshake.InfoHash != myInfoHash {
// 		fmt.Printf("dropping %s: infohash mismatch\n", conn.RemoteAddr().String())
// 		return
// 	}
//
// 	resHandshake := NewHandshake(myInfoHash, myPeerId)
// 	_, err = conn.Write(resHandshake.Serialize())
// 	if err != nil {
// 		fmt.Printf("dropping %s: didn't send handshake reply\n", conn.RemoteAddr().String())
// 		return
// 	}
//
// 	bfPayload := tracker.GetBitfield()
// 	bfMsg := Message{
// 		Id:      MsgBitfield,
// 		Payload: bfPayload,
// 	}
// 	_, err = conn.Write(bfMsg.Serialize())
// 	if err != nil {
// 		fmt.Printf("dropping %s: couldn't send bitfield\n", conn.RemoteAddr().String())
// 		return
// 	}
//
// 	unChokeMsg := Message{Id: MsgUnchoke}
// 	_, err = conn.Write(unChokeMsg.Serialize())
// 	if err != nil {
// 		fmt.Printf("dropping %s: couldn't send unchoke\n", conn.RemoteAddr().String())
// 		return
// 	}
//
// 	serveDataLoop(conn, seedFile)
// }

// func serveDataLoop(conn net.Conn, seedFile io.ReaderAt, pieceLength int) {
// 	defer conn.Close()
//
// 	for {
// 		msg, err := ReadMessage(conn)
// 		if err != nil {
// 			fmt.Printf("peer disconnected: %v\n", err)
// 			return
// 		}
//
// 		// Keep-Alive
// 		if msg == nil {
// 			continue
// 		}
//
// 		if msg.Id == MsgRequest {
// 			if len(msg.Payload) != 12 {
// 				fmt.Println("malformed request; dropping peer")
// 				return
// 			}
//
// 			index := int(binary.BigEndian.Uint32(msg.Payload[0:4]))
// 			begin := int(binary.BigEndian.Uint32(msg.Payload[4:8]))
// 			length := int(binary.BigEndian.Uint32(msg.Payload[8:12]))
//
// 			fileOffset := int64((index * pieceLength) + begin)
//
// 			blockBuf := make([]byte, length)
// 			_, err = seedFile.ReadAt(blockBuf, fileOffset)
//
// 		}
// 	}
// }
