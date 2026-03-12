package torrent

import (
	"crypto/sha1"
	"fmt"
	"os"
	"piecewise/internal/bencode"
)

type TorrentFile struct {
	Announce    string // tracker url
	InfoHash    [20]byte
	PieceHashes [][20]byte
	PieceLength int
	Length      int
	Name        string
}

func ReadFrom(path string) (TorrentFile, error) {
	torrentFile := TorrentFile{}

	f, err := os.Open(path)
	if err != nil {
		return torrentFile, err
	}
	defer f.Close()

	decoded, err := bencode.Decode(f)
	if err != nil {
		return torrentFile, err
	}

	torrentMap, ok := decoded.(map[string]any)
	if !ok {
		return torrentFile, fmt.Errorf("expected decoded torrent to be a map")
	}

	announce, ok := torrentMap["announce"].(string)
	if !ok {
		return torrentFile, fmt.Errorf("weird 'announce' url")
	}
	torrentFile.Announce = announce

	info, ok := torrentMap["info"].(map[string]any)
	if !ok {
		return torrentFile, fmt.Errorf("weird 'info' map")
	}

	bencodedInfo, err := bencode.Encode(info)
	if err != nil {
		return torrentFile, err
	}

	torrentFile.InfoHash = sha1.Sum(bencodedInfo)

	length, ok := info["length"].(int64)
	if !ok {
		return torrentFile, fmt.Errorf("weird 'length'")
	}
	torrentFile.Length = int(length)

	pieceLength, ok := info["piece length"].(int64)
	if !ok {
		return torrentFile, fmt.Errorf("weird 'piece length'")
	}
	torrentFile.PieceLength = int(pieceLength)

	name, ok := info["name"].(string)
	if !ok {
		return torrentFile, fmt.Errorf("weird 'name'")
	}
	torrentFile.Name = name

	piecesStr, ok := info["pieces"].(string)
	if !ok {
		return torrentFile, fmt.Errorf("weird 'pieces'")
	}

	if len(piecesStr)%20 != 0 {
		return torrentFile, fmt.Errorf("length of pieces should be a multiple of 20 | got %d", len(piecesStr))
	}

	pieceHashes := make([][20]byte, 0, len(piecesStr)/20)

	i := 0
	for i < len(piecesStr) {
		var hash [20]byte
		copy(hash[:], piecesStr[i:i+20])
		pieceHashes = append(pieceHashes, hash)

		i += 20
	}

	torrentFile.PieceHashes = pieceHashes

	return torrentFile, nil
}
