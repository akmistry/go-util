package bitmap

import (
	"math"
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

// Return whether all bits are false
func (v *Bitmap256) Empty() bool {
	return (v[0] | v[1] | v[2] | v[3]) == 0
}

// Return whether all bits are true
func (v *Bitmap256) Full() bool {
	return (v[0] & v[1] & v[2] & v[3]) == math.MaxUint64
}

// Return the number of true bits ("population count").
func (v *Bitmap256) Count() int {
	return bits.OnesCount64(v[0]) +
		bits.OnesCount64(v[1]) +
		bits.OnesCount64(v[2]) +
		bits.OnesCount64(v[3])
}

// Return the position of the first true bit in the bitmap, and 256 if there
// are no true bits.
func (v *Bitmap256) FindFirstSet() int {
	return v.FindNextSet(0)
}

// Return the position of the first true bit starting at position pos, and
// 256 if there are no further true bits.
func (v *Bitmap256) FindNextSet(pos uint8) int {
	i := int(pos >> 6)
	off := pos & 63
	masked := v[i] & (^(uint64(1<<off) - 1))
	if masked != 0 {
		return (i << 6) + bits.TrailingZeros64(masked)
	}
	i++
	for ; i < 4; i++ {
		if v[i] != 0 {
			return (i << 6) + bits.TrailingZeros64(v[i])
		}
	}
	return 256
}

// Return the position of the first false bit in the bitmap, and 256 if there
// are no false bits.
func (v *Bitmap256) FindFirstClear() int {
	return v.FindNextClear(0)
}

// Return the position of the first false bit starting at position pos, and
// 256 if there are no further false bits.
func (v *Bitmap256) FindNextClear(pos uint8) int {
	i := int(pos >> 6)
	off := pos & 63
	masked := v[i] | (uint64(1<<off) - 1)
	if masked != math.MaxUint64 {
		return (i << 6) + bits.TrailingZeros64(^masked)
	}
	i++
	for ; i < 4; i++ {
		if v[i] != math.MaxUint64 {
			return (i << 6) + bits.TrailingZeros64(^v[i])
		}
	}
	return 256
}

// Return the position of the n-th (zero-indexed) true bit in the bitmap, and
// 256 if there are not true bits. The first true bit is n == 0.
func (v *Bitmap256) FindNthSet(n uint8) int {
	pos := 0
	for i := 0; i < 4; i++ {
		setCount := uint8(bits.OnesCount64(v[i]))
		if setCount <= n {
			pos += 64
			n -= setCount
			continue
		}

		temp := v[i]
		set32 := uint8(bits.OnesCount32(uint32(temp)))
		if set32 <= n {
			pos += 32
			n -= set32
			temp >>= 32
		}

		set16 := uint8(bits.OnesCount16(uint16(temp)))
		if set16 <= n {
			pos += 16
			n -= set16
			temp >>= 16
		}

		set8 := uint8(bits.OnesCount8(uint8(temp)))
		if set8 <= n {
			pos += 8
			n -= set8
			temp >>= 8
		}

		set4 := uint8(bits.OnesCount8(uint8(temp & 0x0F)))
		if set4 <= n {
			pos += 4
			n -= set4
			temp >>= 4
		}

		set2 := uint8(bits.OnesCount8(uint8(temp & 0x03)))
		if set2 <= n {
			pos += 2
			n -= set2
			temp >>= 2
		}

		set1 := uint8(temp & 1)
		if set1 <= n {
			pos++
			n -= set1
			temp >>= 1
		}
		break
	}
	return pos
}
