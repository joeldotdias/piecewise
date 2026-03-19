package p2p

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

const MaxBlockSize = 16384 // 16 KB
const MaxBacklog = 15

// this is a single tcp connection with a peer
// holding their current state
type Client struct {
	Conn     net.Conn
	Choked   bool
	Bitfield Bitfield
	peerId   [20]byte
	infoHash [20]byte
}

// block of data received from a peer
type Piece struct {
	Index int
	Begin int
	Block []byte
}

type PieceProgress struct {
	index      int // current piece being downloaded
	client     *Client
	buf        []byte // 256 KB buf to hold the final piece
	downloaded int
	requested  int
	backlog    int      // unfulfilled requests
	hash       [20]byte // expected hash of the current piece
}

func (c *Client) ReadLoop(work *PieceWork) ([]byte, error) {
	err := c.SendInterested()
	if err != nil {
		return nil, err
	}

	state := &PieceProgress{
		index:  work.index,
		client: c,
		buf:    make([]byte, work.length),
		hash:   work.hash,
	}

	for {
		c.Conn.SetReadDeadline(time.Now().Add(30 * time.Second))

		msg, err := ReadMessage(c.Conn)
		if err != nil {
			return nil, err
		}

		// Keep-Alive
		if msg == nil {
			continue
		}

		switch msg.Id {
		case MsgChoke:
			c.Choked = true
			// fmt.Println("\n\t-> got CHOKED")

		case MsgUnchoke:
			c.Choked = false
			// fmt.Println("\n\t-> got UNCHOKED")
			state.requestBlocks()

		case MsgBitfield:
			c.Bitfield = Bitfield(msg.Payload)
			// fmt.Printf("\n\t-> got BITFIELD | peer has %d bytes of inventory map\n", len(msg.Payload)) // we using big words

		case MsgHave:
			if len(msg.Payload) != 4 {
				return nil, fmt.Errorf("expected payload len 4 for MsgHave | got %d", len(msg.Payload))
			}
			index := int(binary.BigEndian.Uint32(msg.Payload))
			c.Bitfield.SetPiece(index)
			// fmt.Printf("\n\t-> peer just got a piece %d\n", index)

		case MsgPiece:
			piece, err := ParsePiecePayload(msg.Payload)
			if err != nil {
				return nil, fmt.Errorf("couldn't parse piece: %w", err)
			}

			copy(state.buf[piece.Begin:], piece.Block)

			state.downloaded += len(piece.Block)
			state.backlog--

			// fmt.Printf("\ndownloaded a block | progress: %d / %d bytes", state.downloaded, len(state.buf))

			if state.downloaded >= len(state.buf) {
				// fmt.Println("\ndownloaded an entire piece %d!!!!!!", state.index)

				err := state.verify()
				if err != nil {
					return nil, err
				}

				return state.buf, nil
			}

			// still not done, then we continue asking for more blocks
			state.requestBlocks()
		}
	}
}

func (state *PieceProgress) requestBlocks() {
	for state.requested < len(state.buf) && state.backlog < MaxBacklog {
		bytesLeft := len(state.buf) - state.requested
		blockSize := min(bytesLeft, MaxBlockSize)

		err := state.client.SendRequest(state.index, state.requested, blockSize)
		if err != nil {
			fmt.Printf("couldn't send request: %v\n", err)
			return
		}

		state.requested += blockSize
		state.backlog++
	}
}

func (state *PieceProgress) verify() error {
	check := sha1.Sum(state.buf)

	if !bytes.Equal(check[:], state.hash[:]) {
		return fmt.Errorf("piece %d failed integrity check", state.index)
	}

	return nil
}

func (c *Client) SendRequest(index, begin, length int) error {
	payload := make([]byte, 12)
	binary.BigEndian.PutUint32(payload[0:4], uint32(index))
	binary.BigEndian.PutUint32(payload[4:8], uint32(begin))
	binary.BigEndian.PutUint32(payload[8:12], uint32(length))

	msg := Message{
		Id:      MsgRequest,
		Payload: payload,
	}

	return c.sendMessage(&msg)
}

func ParsePiecePayload(payload []byte) (Piece, error) {
	if len(payload) < 8 {
		return Piece{}, fmt.Errorf("length too small. payload should at least contain 8 bytes for index and begin | got %d", len(payload))
	}

	piece := Piece{
		Index: int(binary.BigEndian.Uint32(payload[0:4])),
		Begin: int(binary.BigEndian.Uint32(payload[4:8])),
		Block: payload[8:],
	}

	return piece, nil
}

func (c *Client) SendInterested() error {
	msg := Message{
		Id: MsgInterested,
	}

	return c.sendMessage(&msg)
}

func (c *Client) SendNotInterested() error {
	msg := Message{
		Id: MsgNotInterested,
	}

	return c.sendMessage(&msg)
}

func (c *Client) sendMessage(msg *Message) error {
	_, err := c.Conn.Write(msg.Serialize())
	if err != nil {
		return fmt.Errorf("couldn't send message: \n\t%w", err)
	}

	return nil
}
