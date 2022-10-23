//go:build ignore

package bitmap

import "math/bits"

// Return the number of true bits up to, but not including, position pos.
func (v *Bitmap256) CountLess(pos uint8) int {
	index := int(pos >> 6)
	mask := (1 << (pos & 63)) - uint64(1)
	count := bits.OnesCount64(v[index] & mask)

	var fullCounts [4]int
	fullCounts[1] = bits.OnesCount64(v[0])
	fullCounts[2] = fullCounts[1] + bits.OnesCount64(v[1])
	fullCounts[3] = fullCounts[2] + bits.OnesCount64(v[2])
	return count + fullCounts[index]
}
