package bufferpool

import (
	"math/bits"
	"sync"
)

const (
	MinSizeBits = 4
	MaxSizeBits = 24
)

var pool [MaxSizeBits + 1]*sync.Pool

func init() {
	for i := MinSizeBits; i <= MaxSizeBits; i++ {
		size := 1 << uint(i)
		pool[i] = &sync.Pool{New: func() interface{} { return make([]byte, size) }}
	}
}

func ceilLog2(size int) int {
	return bits.Len(uint(size) - 1)
}

func isPow2(size int) bool {
	return size > 0 && (size&(size-1)) == 0
}

func Get(size int) []byte {
	bits := ceilLog2(size)
	if bits < MinSizeBits || bits > MaxSizeBits {
		return make([]byte, size)
	}

	return pool[bits].Get().([]byte)[:size]
}

func Put(buf []byte) {
	size := cap(buf)
	buf = buf[:size]
	bits := ceilLog2(size)
	if !isPow2(size) || bits < MinSizeBits || bits > MaxSizeBits {
		return
	}

	for i := range buf {
		buf[i] = 0
	}
	pool[bits].Put(buf)
}
