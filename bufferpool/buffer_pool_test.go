package bufferpool

import (
	"math/rand"
	"testing"
)

func TestBufferPool(t *testing.T) {
	const maxPow = 26

	for i := 0; i < maxPow; i++ {
		size := 1 << uint(i)
		buf := Get(size)
		if len(*buf) != size {
			t.Errorf("len(buf) %d != size %d", len(*buf), size)
		}
		if cap(*buf) != size {
			t.Errorf("cap(buf) %d != size %d", cap(*buf), size)
		}
		Put(buf)

		buf = Get(size)
		if len(*buf) != size {
			t.Errorf("len(buf) %d != size %d", len(*buf), size)
		}
		if cap(*buf) != size {
			t.Errorf("cap(buf) %d != size %d", cap(*buf), size)
		}
		Put(buf)
	}
}

func TestBufferPoolNonPow2(t *testing.T) {
	const maxSize = 1024 * 1024
	const iterations = 1000

	for i := 0; i < iterations; i++ {
		size := rand.Intn(maxSize)
		if size == 0 || (size&(size-1)) == 0 {
			// Zero or power of two. Skip.
			continue
		}

		buf := Get(size)
		if len(*buf) != size {
			t.Errorf("len(buf) %d != size %d", len(*buf), size)
		}
		if cap(*buf) > size*2 {
			t.Errorf("cap(buf) %d > size*2 %d", cap(*buf), size*2)
		}
		Put(buf)
	}
}

func BenchmarkBufferPool(b *testing.B) {
	const size = 1024

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf := Get(size)
		Put(buf)
	}
}
