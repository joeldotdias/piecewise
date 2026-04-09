package main

import (
	"fmt"
	"log"
	"os"
	"piecewise/internal/bencode"
)

func main() {
	localTrackerURL := "http://192.168.0.107:8080/announce"

	inputFile := "debian-13.4.0-amd64-netinst.iso.torrent"
	outputFile := "debian-lan.torrent"

	file, err := os.Open(inputFile)
	if err != nil {
		log.Fatalf("Failed to open original torrent: %v", err)
	}
	defer file.Close()

	// 2. Decode the original torrent using your parser
	rawDecoded, err := bencode.Decode(file)
	if err != nil {
		log.Fatalf("Failed to decode: %v", err)
	}

	torrentDict, ok := rawDecoded.(map[string]any)
	if !ok {
		log.Fatalf("Invalid torrent file format")
	}

	// 3. Hijack the announce URL!
	torrentDict["announce"] = localTrackerURL

	// 4. Encode it back using your encoder
	encodedBytes, err := bencode.Encode(torrentDict)
	if err != nil {
		log.Fatalf("Failed to encode: %v", err)
	}

	// 5. Save the new LAN party file
	err = os.WriteFile(outputFile, encodedBytes, 0644)
	if err != nil {
		log.Fatalf("Failed to save: %v", err)
	}

	fmt.Printf("Successfully generated %s pointing to %s\n", outputFile, localTrackerURL)

}
