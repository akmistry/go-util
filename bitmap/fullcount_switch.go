//go:build !arm64

package bitmap

import "math/bits"

// Return the number of true bits up to, but not including, position pos.
func (v *Bitmap256) CountLess(pos uint8) int {
	index := int(pos >> 6)
	mask := (1 << (pos & 63)) - uint64(1)
	count := bits.OnesCount64(v.words[index] & mask)

	switch index {
	case 3:
		count += bits.OnesCount64(v.words[2])
		fallthrough
	case 2:
		count += bits.OnesCount64(v.words[1])
		fallthrough
	case 1:
		count += bits.OnesCount64(v.words[0])
	case 0:
	}
	return count
}
