package p2p

type Bitfield []byte

func (bf Bitfield) HasPiece(idx int) bool {
	byteIdx := idx / 8
	bitOffset := idx % 8

	if byteIdx < 0 || byteIdx >= len(bf) {
		return false
	}

	mask := byte(1 << (7 - bitOffset))

	return (bf[byteIdx] & mask) != 0
}

func (bf Bitfield) SetPiece(idx int) {
	byteIdx := idx / 8
	bitOffset := idx % 8

	if byteIdx < 0 || byteIdx >= len(bf) {
		return
	}

	mask := byte(1 << (7 - bitOffset))

	bf[byteIdx] |= mask
}
