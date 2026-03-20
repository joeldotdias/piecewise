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
	numPieces  int
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

	for i, r := range pt.rarity {
		if !pt.downloaded[i] && !pt.pending[i] && peerBf.HasPiece(i) {
			if r < minRarity {
				minRarity = r
				bestIndex = i
			}
		}
	}

	if bestIndex != -1 {
		pt.pending[bestIndex] = true
		return bestIndex, true
	}

	return -1, false
}

func (pt *PieceTracker) MarkFailed(index int) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.pending[index] = false
}

func (pt *PieceTracker) MarkDownloaded(index int) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.downloaded[index] = true
	pt.pending[index] = false
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
		numPieces:  numPieces,
	}
}
