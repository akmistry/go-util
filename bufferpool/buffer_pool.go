package bufferpool

import (
	"math/bits"
	"sync"
)

const (
	MinSizeBits = 4
	MaxSizeBits = 24

	MinBufferSize = 1 << MinSizeBits
)

var pool [MaxSizeBits + 1]*sync.Pool

func init() {
	for i := MinSizeBits; i <= MaxSizeBits; i++ {
		size := 1 << uint(i)
		pool[i] = &sync.Pool{New: func() interface{} {
			b := make([]byte, size)
			return &b
		}}
	}
}

func ceilLog2(size int) int {
	return bits.Len(uint(size) - 1)
}

func isPow2(size int) bool {
	return size > 0 && (size&(size-1)) == 0
}

func GetUninit(size int) *[]byte {
	bits := ceilLog2(size)
	if bits < MinSizeBits || bits > MaxSizeBits {
		b := make([]byte, size)
		return &b
	}

	b := pool[bits].Get().(*[]byte)
	*b = (*b)[:size]

	return b
}

func Get(size int) *[]byte {
	buf := GetUninit(size)

	if (*buf)[0] != 0 {
		// 'make' zero-initialises slices, so if we see a non-zero value in the
		// first byte, we know the slice was a re-use and needs to be zero'd.
		zb := (*buf)[:cap(*buf)]
		for i := range zb {
			zb[i] = 0
		}
	}

	return buf
}

func Put(buf *[]byte) {
	size := cap(*buf)
	*buf = (*buf)[:size]
	bits := ceilLog2(size)
	if !isPow2(size) || bits < MinSizeBits || bits > MaxSizeBits {
		return
	}

	// Poison the first byte to indicate to Get() this was a re-used buffer.
	(*buf)[0] = 1

	pool[bits].Put(buf)
}
