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

func UnmarshalPeers(peersBlob any) ([]Peer, error) {
	// tracker sent a compact string
	if compact, ok := peersBlob.(string); ok {
		return unmarshalCompactPeers([]byte(compact))
	}

	// tracker sent a dictionary list
	if list, ok := peersBlob.([]any); ok {
		return unmarshalDictionaryPeers(list)
	}

	return nil, fmt.Errorf("weird peers format: expected compact str or list, got %T", peersBlob)
}

func unmarshalCompactPeers(compactPeers []byte) ([]Peer, error) {
	const peerSize = 6
	lenPeers := len(compactPeers)

	if lenPeers%peerSize != 0 {
		return nil, fmt.Errorf("malformed peers. length should be multiple of 6. got: %d", len(compactPeers))
	}

	// this makes initial len zero but caps at lenPeers/peerSize. pretty cool
	peers := make([]Peer, 0, lenPeers/peerSize)

	for i := 0; i < lenPeers; i += 6 {
		peers = append(peers, Peer{
			IP:   net.IP(compactPeers[i : i+4]),
			Port: binary.BigEndian.Uint16(compactPeers[i+4 : i+6]),
		})
	}

	return peers, nil
}

func unmarshalDictionaryPeers(list []any) ([]Peer, error) {
	var peers []Peer
	for _, item := range list {
		dict, ok := item.(map[string]any)
		if !ok {
			continue
		}

		ipStr, okIp := dict["ip"].(string)
		portInt, okPort := dict["port"].(int64)
		if okIp && okPort {
			peers = append(peers, Peer{
				IP:   net.ParseIP(ipStr),
				Port: uint16(portInt),
			})
		}
	}

	return peers, nil
}

func MakePeerId() [20]byte {
	var buf [20]byte

	copy(buf[0:8], []byte("-PW0001-"))
	rand.Read(buf[8:])

	return buf
}
