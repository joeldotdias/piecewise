package p2p

import (
	"encoding/binary"
	"fmt"
	"io"
)

type MessageId byte

// i could do iota here but since these are actual values
// might be better to just hardcode it
const (
	MsgChoke         MessageId = 0
	MsgUnchoke       MessageId = 1
	MsgInterested    MessageId = 2
	MsgNotInterested MessageId = 3
	MsgHave          MessageId = 4
	MsgBitfield      MessageId = 5
	MsgRequest       MessageId = 6
	MsgPiece         MessageId = 7
	MsgCancel        MessageId = 8
)

/* how a message looks
 * length (4 bytes)
 * id (1 byte) -> one of those ids above
 * payload
 */

type Message struct {
	Id      MessageId
	Payload []byte
}

func (m *Message) Serialize() []byte {
	// keep-alive case. we just don't do anything here
	// the buffer will just have the first 4 length bytes set to zero
	if m == nil {
		return make([]byte, 4)
	}

	messageLen := 1 + len(m.Payload)
	buf := make([]byte, 4+messageLen)

	binary.BigEndian.PutUint32(buf[0:4], uint32(messageLen))
	buf[4] = byte(m.Id)
	copy(buf[5:], m.Payload)

	return buf
}

func ReadMessage(r io.Reader) (*Message, error) {
	lengthBuf := make([]byte, 4)

	_, err := io.ReadFull(r, lengthBuf)
	if err != nil {
		return nil, fmt.Errorf("couldn't read length: \n\t%w", err)
	}

	length := binary.BigEndian.Uint32(lengthBuf)

	// keep-alive
	if length == 0 {
		return nil, nil
	}

	messageBuf := make([]byte, length)
	_, err = io.ReadFull(r, messageBuf)
	if err != nil {
		return nil, fmt.Errorf("couldn't read message: \n\t%w", err)
	}

	// so slicing an array of len 1 at [1:] will just return an empty slice instead of crashing
	// weird but nice
	m := Message{
		Id:      MessageId(messageBuf[0]),
		Payload: messageBuf[1:],
	}

	return &m, nil
}

func FormatPiece(index int, begin int, block []byte) Message {
	payload := make([]byte, 8+len(block))

	binary.BigEndian.PutUint32(payload[0:4], uint32(index))
	binary.BigEndian.PutUint32(payload[4:8], uint32(begin))
	copy(payload[8:], block)

	return Message{
		Id:      MsgPiece,
		Payload: payload,
	}
}
