package main

import (
	"crypto/sha1"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"piecewise/internal/bencode"
)

func main() {
	filePath := flag.String("file", "", "file to create torrent for")
	announce := flag.String("tracker", "http://192.168.0.107:8080/announce", "tracker announce URL")
	pieceLen := flag.Int("piece-length", 256*1024, "piece length in bytes")
	outputPath := flag.String("out", "", "output .torrent file")
	flag.Parse()

	if *filePath == "" {
		log.Fatal("--file is required")
	}
	if *outputPath == "" {
		*outputPath = *filePath + ".torrent"
	}

	f, err := os.Open(*filePath)
	if err != nil {
		log.Fatalf("couldn't open file: %v", err)
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		log.Fatalf("couldn't stat file: %v", err)
	}
	fileSize := int(stat.Size())

	// hash pieces
	fmt.Printf("hashing %s (%d bytes) with piece length %d...\n", *filePath, fileSize, *pieceLen)
	buf := make([]byte, *pieceLen)
	var pieceHashes []byte
	pieceCount := 0
	for {
		n, err := io.ReadFull(f, buf)
		if n > 0 {
			hash := sha1.Sum(buf[:n])
			pieceHashes = append(pieceHashes, hash[:]...)
			pieceCount++
		}
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			log.Fatalf("read error: %v", err)
		}
	}
	fmt.Printf("hashed %d pieces\n", pieceCount)

	// build info dict — key order matters for consistent info hash!
	// bencode dicts must be sorted by key, make sure your encoder does this
	infoDict := map[string]any{
		"length":       fileSize,
		"name":         filepath.Base(*filePath),
		"piece length": *pieceLen,
		"pieces":       string(pieceHashes),
	}

	// compute info hash from bencoded info dict
	infoBencoded, err := bencode.Encode(infoDict)
	if err != nil {
		log.Fatalf("couldn't encode info dict: %v", err)
	}
	infoHash := sha1.Sum(infoBencoded)
	fmt.Printf("info hash: %x\n", infoHash)

	// write .torrent file
	torrentDict := map[string]any{
		"announce": *announce,
		"info":     infoDict,
	}
	torrentBytes, err := bencode.Encode(torrentDict)
	if err != nil {
		log.Fatalf("couldn't encode torrent: %v", err)
	}
	err = os.WriteFile(*outputPath, torrentBytes, 0644)
	if err != nil {
		log.Fatalf("couldn't write torrent file: %v", err)
	}
	fmt.Printf("wrote %s\n", *outputPath)
}
