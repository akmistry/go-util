package bitmap

import (
	"math/bits"
)

// A Bitmap256 is a bitmap of 256 elements.
//
// Using a fixed-size structure avoids the overhead of using a slice (both the
// 3-word slice structure, and the associated cache-miss due to the slice
// indirection), at the cost of flexibility.
type Bitmap256 struct {
	words [4]uint64
}

// Set the bit at position pos to true.
func (v *Bitmap256) Set(pos uint8) {
	v.words[pos>>6] |= 1 << (pos & 63)
}

// Set the bit at position pos to false.
func (v *Bitmap256) Clear(pos uint8) {
	v.words[pos>>6] &= ^(1 << (pos & 63))
}

// Return the bit value at position pos.
func (v *Bitmap256) Get(pos uint8) bool {
	return ((v.words[pos>>6] >> (pos & 63)) & 1) == 1
}

// Return the number of true bits ("population count").
func (v *Bitmap256) Count() int {
	return bits.OnesCount64(v.words[0]) +
		bits.OnesCount64(v.words[1]) +
		bits.OnesCount64(v.words[2]) +
		bits.OnesCount64(v.words[3])
}

// Return the number of true bits up to, but not including, position pos.
func (v *Bitmap256) CountLess(pos uint8) int {
	index := int(pos >> 6)
	count := 0

	switch index {
	case 3:
		count = bits.OnesCount64(v.words[2])
		fallthrough
	case 2:
		count += bits.OnesCount64(v.words[1])
		fallthrough
	case 1:
		count += bits.OnesCount64(v.words[0])
	case 0:
	}

	/*
		// Branch-free version. Slower than the switch above.
		tempIndex := index
		nonZero := (tempIndex | (tempIndex >> 1)) & 1
		count = bits.OnesCount64(v.words[0]) & -nonZero

		tempIndex -= nonZero
		nonZero = (tempIndex | (tempIndex >> 1)) & 1
		count += bits.OnesCount64(v.words[1]) & -nonZero

		tempIndex -= nonZero
		nonZero = tempIndex & 1
		count += bits.OnesCount64(v.words[2]) & -nonZero
	*/

	mask := (1 << (pos & 63)) - uint64(1)
	count += bits.OnesCount64(v.words[index] & mask)
	return count
}
