package torrent

import (
	"fmt"
	"net/http"
	"net/url"
	"piecewise/internal/bencode"
	"strconv"
)

type TrackerResponse struct {
	Interval int
	Peers    any //string
}

func RequestPeers(trackerUrl string) (TrackerResponse, error) {
	tr := TrackerResponse{}

	res, err := http.Get(trackerUrl)
	if err != nil {
		return tr, fmt.Errorf("couldn't make request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return tr, fmt.Errorf("got status: %d", res.StatusCode)
	}

	rawDecoded, err := bencode.Decode(res.Body)
	if err != nil {
		return tr, fmt.Errorf("couldn't decoded body: %w", err)
	}

	decoded, ok := rawDecoded.(map[string]any)
	if !ok {
		return tr, fmt.Errorf("weird decoded body: %v", rawDecoded)
	}

	interval, ok := decoded["interval"].(int64)
	if !ok {
		return tr, fmt.Errorf("weird 'interval'")
	}
	tr.Interval = int(interval)

	peers, ok := decoded["peers"] //.(string)
	if !ok {
		return tr, fmt.Errorf("weird 'peers': missing from response")
	}
	tr.Peers = peers

	return tr, nil
}

func BuildTrackerURL(meta *TorrentMeta, peerId [20]byte) (string, error) {
	tu, err := url.Parse(meta.Announce)
	if err != nil {
		return "", err
	}

	params := url.Values{
		"info_hash":  []string{string(meta.InfoHash[:])},
		"peer_id":    []string{string(peerId[:])},
		"port":       []string{strconv.Itoa(6881)},
		"uploaded":   []string{"0"},
		"downloaded": []string{"0"},
		"left":       []string{strconv.Itoa(meta.Length)},
		"compact":    []string{"1"},
	}

	tu.RawQuery = params.Encode()

	return tu.String(), nil
}
