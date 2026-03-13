package torrent

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net"
)

type Peer struct {
	IP   net.IP // first 4 bytes
	Port uint16 // last 2 bytes
}

func UnmarshalPeers(compactPeers []byte) ([]Peer, error) {
	lenPeers := len(compactPeers)
	if lenPeers%6 != 0 {
		return nil, fmt.Errorf("malformed peers. length should be multiple of 6. got: %d", len(compactPeers))
	}

	// this makes initial len zero but caps at lenPeers/6. pretty cool
	peers := make([]Peer, 0, lenPeers/6)

	for i := 0; i < lenPeers; i += 6 {
		peers = append(peers, Peer{
			IP:   net.IP(compactPeers[i : i+4]),
			Port: binary.BigEndian.Uint16(compactPeers[i+4 : i+6]),
		})
	}

	return peers, nil
}

func MakePeerId() [20]byte {
	var buf [20]byte

	copy(buf[0:8], []byte("-PW0001-"))
	rand.Read(buf[8:])

	return buf
}
