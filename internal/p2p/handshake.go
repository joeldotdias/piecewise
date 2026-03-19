package p2p

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"time"
)

const (
	handshakeLen = 68
	pstrLen      = 19
)

type Handshake struct {
	Pstr     string
	Reserved [8]byte
	InfoHash [20]byte
	PeerId   [20]byte
}

func DialPeer(ip net.IP, port uint16, infoHash [20]byte, myId [20]byte) (net.Conn, error) {
	addr := net.JoinHostPort(ip.String(), strconv.Itoa(int(port)))

	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return nil, fmt.Errorf("couldn't make connection: %w", err)
	}

	// a peer may accept a connection but never send a handshake back
	// which is bad manners so we don't want that happening
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	hs := NewHandshake(infoHash, myId)
	hsBuf := hs.Serialize()

	_, err = conn.Write(hsBuf)
	if err != nil {
		return nil, fmt.Errorf("couldn't send handshake: %w", err)
	}

	peerHs, err := ReadHandshake(conn)
	if err != nil {
		return nil, fmt.Errorf("didn't get handshake back from peer: \n\t%w", err)
	}

	if peerHs.InfoHash != infoHash {
		conn.Close()
		return nil, fmt.Errorf("torrent mismatch, expected: %x | got: %x", infoHash, peerHs.InfoHash)
	}

	// we gotta reset the deadline which was set for getting back a handshake
	// coz downloading may take well beyond 5 seconds
	conn.SetDeadline(time.Time{})

	return conn, nil
}

func ReadHandshake(r io.Reader) (Handshake, error) {
	h := Handshake{}

	lengthBuf := make([]byte, 1)
	_, err := io.ReadFull(r, lengthBuf)
	if err != nil {
		return h, fmt.Errorf("couldn't read protocol length: \n\t%w", err)
	}

	handshakeBuf := make([]byte, int(lengthBuf[0])+48)
	_, err = io.ReadFull(r, handshakeBuf)
	if err != nil {
		return h, fmt.Errorf("couldn't read handshake body: \n\t%w", err)
	}

	h.Pstr = string(handshakeBuf[0:pstrLen])
	copy(h.InfoHash[:], handshakeBuf[27:47])
	copy(h.PeerId[:], handshakeBuf[47:67])

	return h, nil
}

func (h *Handshake) Serialize() []byte {
	buf := make([]byte, handshakeLen)

	buf[0] = byte(len(h.Pstr))
	copy(buf[1:1+pstrLen], h.Pstr)
	copy(buf[28:48], h.InfoHash[:])
	copy(buf[48:68], h.PeerId[:])

	return buf
}

func NewHandshake(infoHash, peerId [20]byte) Handshake {
	var h Handshake

	h.Pstr = "BitTorrent protocol"
	h.InfoHash = infoHash
	h.PeerId = peerId

	return h
}
