package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"net/http"
	"piecewise/internal/bencode"
	"piecewise/internal/torrent"
	"strconv"
	"sync"
)

var (
	swarmMutex sync.RWMutex
	swarmPeers map[string]torrent.Peer = make(map[string]torrent.Peer)
)

func main() {
	port := 8080
	http.HandleFunc("/announce", handleAnnounce)

	fmt.Printf("lan tracker started on http://0.0.0.0:%d/announce\n", port)

	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		log.Fatalf("server went bye bye: %v", err)
	}
}

func handleAnnounce(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	peerId := query.Get("peer_id")
	portStr := query.Get("port")

	if peerId == "" || portStr == "" {
		http.Error(w, "missing peer_id or port", http.StatusBadRequest)
		return
	}

	peerPort, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		http.Error(w, "invalid port", http.StatusBadRequest)
		return
	}

	ipStr, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ipStr = r.RemoteAddr
	}
	parsedIp := net.ParseIP(ipStr).To4()
	if parsedIp == nil {
		http.Error(w, "couldn't parse IPv4 addr", http.StatusBadRequest)
		return
	}

	swarmMutex.Lock()
	swarmPeers[peerId] = torrent.Peer{IP: parsedIp, Port: uint16(peerPort)}
	swarmMutex.Unlock()

	fmt.Printf("registered peer: %s:%d\n", parsedIp.String(), peerPort)

	swarmMutex.RLock()
	var peerBytes bytes.Buffer
	for id, p := range swarmPeers {
		// skipping the requesting peer
		if id == peerId {
			continue
		}

		peerBytes.Write(p.IP)

		portBuf := make([]byte, 2)
		binary.BigEndian.PutUint16(portBuf, p.Port)
		peerBytes.Write(portBuf)
	}
	swarmMutex.RUnlock()

	// this is so stupid i literally have a struct for this but gotta make it a map
	// coz my bencode function take in a map
	// or i can change that up but i don't feel like
	trackerResponse := map[string]any{
		"interval": 1800,
		"peers":    peerBytes.String(),
	}

	bencodedBytes, err := bencode.Encode(trackerResponse)
	if err != nil {
		http.Error(w, fmt.Sprintf("bencoding error: %v", err), http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write(bencodedBytes)
}
