package core

import (
	"math/bits"
	"strings"
)

type Bitfield struct {
	bits []byte
}

func NewBitfield(numPieces uint64) Bitfield {
	return Bitfield{
		bits: make([]byte, (numPieces+7)/8),
	}
}

func (b Bitfield) Has(piece uint64) bool {
	byteIdx := piece / 8
	bitIdx := 7 - (piece % 8)
	return b.bits[byteIdx]&(1<<bitIdx) != 0
}

func (b Bitfield) Set(piece uint64) {
	byteIdx := piece / 8
	bitIdx := 7 - (piece % 8)
	b.bits[byteIdx] |= 1 << bitIdx
}

func (b Bitfield) Clear(piece uint64) {
	byteIdx := piece / 8
	bitIdx := 7 - (piece % 8)
	b.bits[byteIdx] &^= 1 << bitIdx
}

func (b Bitfield) Count() uint64 {
	n := uint64(0)
	for _, by := range b.bits {
		n += uint64(bits.OnesCount8(by))
	}
	return n
}

func (b Bitfield) Bytes() []byte {
	return b.bits
}

func BitfieldFromBytes(data []byte) Bitfield {
	return Bitfield{bits: data}
}

// AllSet returns true if all bits up to numPieces are set
func (b Bitfield) AllSet(numPieces uint64) bool {
	if numPieces == 0 {
		return true
	}

	fullBytes := numPieces / 8
	remainingBits := numPieces % 8

	// Check full bytes (must be 0xFF)
	for i := uint64(0); i < fullBytes; i++ {
		if b.bits[i] != 0xFF {
			return false
		}
	}

	// Check remaining bits in the last byte (if any)
	if remainingBits > 0 {
		// Bits are MSB-first, so we want the top `remainingBits` set
		mask := byte(0xFF << (8 - remainingBits))
		if b.bits[fullBytes]&mask != mask {
			return false
		}
	}

	return true
}

// ToString returns a simple string of 1s and 0s
func (b Bitfield) ToString(numPieces uint64) string {
	var sb strings.Builder
	for i := uint64(0); i < numPieces; i++ {
		if b.Has(i) {
			sb.WriteString("█")
		} else {
			sb.WriteString("░")
		}
		if (i+1)%8 == 0 && i+1 < numPieces {
			sb.WriteString(" ")
		}
	}
	return sb.String()
}
