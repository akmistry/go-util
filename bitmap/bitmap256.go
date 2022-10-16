package bitmap

import (
	"math/bits"
)

// A Bitmap256 is a bitmap of 256 elements.
//
// Using a fixed-size structure avoids the overhead of using a slice (both the
// 3-word slice structure, and the associated cache-miss due to the slice
// indirection), at the cost of flexibility.
type Bitmap256 [4]uint64

// Set the bit at position pos to true.
func (v *Bitmap256) Set(pos uint8) {
	v[pos>>6] |= 1 << (pos & 63)
}

// Set the bit at position pos to false.
func (v *Bitmap256) Clear(pos uint8) {
	v[pos>>6] &= ^(1 << (pos & 63))
}

// Return the bit value at position pos.
func (v *Bitmap256) Get(pos uint8) bool {
	return ((v[pos>>6] >> (pos & 63)) & 1) == 1
}

// Return the number of true bits ("population count").
func (v *Bitmap256) Count() int {
	return bits.OnesCount64(v[0]) +
		bits.OnesCount64(v[1]) +
		bits.OnesCount64(v[2]) +
		bits.OnesCount64(v[3])
}
