package torrent_test

import (
	"fmt"
	"piecewise/internal/torrent"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadFrom(t *testing.T) {
	got, err := torrent.ReadFrom("testdata/debian-13.3.0-amd64-DVD-1.iso.torrent")
	require.NoError(t, err, "couldn't read debian torrent file")

	require.NotEmpty(t, got.Announce, "announce url missing")
	require.NotEmpty(t, got.Name, "file name missing")
	require.Greater(t, got.Length, 0, "file length should be greater than 0")
	require.Greater(t, got.PieceLength, 0, "piece length should be greater than 0")
	require.NotEmpty(t, got.PieceHashes, "piece hashes missing")

	emptyHash := [20]byte{}
	require.NotEqual(t, emptyHash, got.InfoHash, "InfoHash couldn't be calculated")

	fmt.Println(got)
}
