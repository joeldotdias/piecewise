package torrent

import (
	"crypto/sha1"
	"fmt"
	"os"
	"piecewise/internal/bencode"
)

/* How an unparsed .torrent file looks
 * announce: url of the tracker server
 * info: nested dict containing the actual file stuff
 *
 * in the info dict,
 * name: obviously, the name of the file
 * length: once again obviously, length of the file in bytes
 * piece length: size in bytes of each chunk
 * pieces: binary string with a bunch of 20 byte hashes of each piece
 */

type TorrentMeta struct {
	Announce    string
	InfoHash    [20]byte
	PieceHashes [][20]byte
	PieceLength int
	Length      int
	Name        string
}

func ReadFrom(path string) (TorrentMeta, error) {
	meta := TorrentMeta{}

	f, err := os.Open(path)
	if err != nil {
		return meta, err
	}
	defer f.Close()

	decoded, err := bencode.Decode(f)
	if err != nil {
		return meta, err
	}

	torrentMap, ok := decoded.(map[string]any)
	if !ok {
		return meta, fmt.Errorf("expected decoded torrent to be a map")
	}

	announce, ok := torrentMap["announce"].(string)
	if !ok {
		return meta, fmt.Errorf("weird 'announce' url")
	}
	meta.Announce = announce

	info, ok := torrentMap["info"].(map[string]any)
	if !ok {
		return meta, fmt.Errorf("weird 'info' map")
	}

	bencodedInfo, err := bencode.Encode(info)
	if err != nil {
		return meta, err
	}

	meta.InfoHash = sha1.Sum(bencodedInfo)

	if length, ok := info["length"].(int64); ok {
		meta.Length = int(length)
	} else if files, ok := info["files"].([]any); ok {
		var totalLength int
		for _, file := range files {
			fileMap, ok := file.(map[string]any)
			if !ok {
				continue
			}

			if fileLength, ok := fileMap["length"].(int64); ok {
				totalLength += int(fileLength)
			}
		}

		meta.Length = totalLength
	} else {
		return meta, fmt.Errorf("weird 'length': neither single nor multi file structure")
	}

	pieceLength, ok := info["piece length"].(int64)
	if !ok {
		return meta, fmt.Errorf("weird 'piece length'")
	}
	meta.PieceLength = int(pieceLength)

	name, ok := info["name"].(string)
	if !ok {
		return meta, fmt.Errorf("weird 'name'")
	}
	meta.Name = name

	piecesStr, ok := info["pieces"].(string)
	if !ok {
		return meta, fmt.Errorf("weird 'pieces'")
	}

	if len(piecesStr)%20 != 0 {
		return meta, fmt.Errorf("length of pieces should be a multiple of 20 | got %d", len(piecesStr))
	}

	pieceHashes := make([][20]byte, 0, len(piecesStr)/20)

	i := 0
	for i < len(piecesStr) {
		var hash [20]byte
		copy(hash[:], piecesStr[i:i+20])
		pieceHashes = append(pieceHashes, hash)

		i += 20
	}

	meta.PieceHashes = pieceHashes

	return meta, nil
}

func (t TorrentMeta) String() string {
	return fmt.Sprintf(
		"Torrent Metadata:\n"+
			"  Name:        %s\n"+
			"  Announce:    %s\n"+
			"  Size:        %d bytes\n"+
			"  PieceLength: %d bytes\n"+
			"  InfoHash:    %x\n"+ // %x formats the 20 bytes as a hex string
			"  Piece Count: %d\n",
		t.Name, t.Announce, t.Length, t.PieceLength, t.InfoHash, len(t.PieceHashes),
	)
}
