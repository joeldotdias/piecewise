package p2p

import (
	"math"
	"sync"
)

type PieceTracker struct {
	mu         sync.RWMutex
	rarity     []int
	pending    []bool
	downloaded []bool

	// dramatic name but this is what its actually called
	// https://en.wikipedia.org/wiki/Glossary_of_BitTorrent_terms#Endgame/Endgame_mode
	isEndgame bool
}

func (pt *PieceTracker) AddPeerBitfield(bf Bitfield) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	for i := range pt.rarity {
		if bf.HasPiece(i) {
			pt.rarity[i]++
		}
	}
}

func (pt *PieceTracker) PickNextPiece(peerBf Bitfield) (int, bool) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	bestIndex := -1
	minRarity := math.MaxInt
	unassignedLeft := false

	for i, r := range pt.rarity {
		if !pt.downloaded[i] && !pt.pending[i] {
			unassignedLeft = true

			if peerBf.HasPiece(i) {
				if r < minRarity {
					minRarity = r
					bestIndex = i
				}
			}
		}
	}

	if !unassignedLeft {
		pt.isEndgame = true
	}

	// normal where we found a fresh piece
	if bestIndex != -1 {
		pt.pending[bestIndex] = true
		return bestIndex, true
	}

	// endgame if we don't find a fresh piece
	// then we straight up dogpile on the pieces being downloaded
	if pt.isEndgame {
		for i := range pt.rarity {
			if pt.pending[i] && !pt.downloaded[i] && peerBf.HasPiece(i) {
				return i, true
			}
		}
	}

	return -1, false
}

func (pt *PieceTracker) MarkFailed(index int) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.pending[index] = false
}

func (pt *PieceTracker) MarkDownloaded(index int) bool {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	// duplicate
	if pt.downloaded[index] {
		return false
	}

	pt.downloaded[index] = true
	pt.pending[index] = false

	return true
}

func (pt *PieceTracker) IsDownloaded(index int) bool {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	return pt.downloaded[index]
}

func (pt *PieceTracker) IsDone() bool {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	for _, done := range pt.downloaded {
		if !done {
			return false
		}
	}

	return true
}

func NewPieceTracker(numPieces int) PieceTracker {
	return PieceTracker{
		rarity:     make([]int, numPieces),
		pending:    make([]bool, numPieces),
		downloaded: make([]bool, numPieces),
	}
}
