package bitmap

import (
	"math/rand"
	"testing"
)

func checkSet(t *testing.T, v *Bitmap256, pos uint8, value bool) {
	t.Helper()
	if value {
		v.Set(pos)
	} else {
		v.Clear(pos)
	}
	if v.Get(pos) != value {
		t.Errorf("Value at %d != expected %v", pos, value)
	}
}

func TestSeq(t *testing.T) {
	var vec Bitmap256

	for i := 0; i < 256; i++ {
		if vec.Count() != i {
			t.Errorf("Count %d != expected %d", vec.Count(), i)
		}
		checkSet(t, &vec, uint8(i), true)
		for j := 0; j < i; j++ {
			// All bits up to i are true
			if !vec.Get(uint8(j)) {
				t.Errorf("Bit at %d not set", j)
			}
			c := vec.CountLess(uint8(j))
			if c != j {
				t.Errorf("CountLess(%d) %d != expected %d", j, c, j)
			}
		}
		if vec.Count() != i+1 {
			t.Errorf("Count %d != expected %d", vec.Count(), i+1)
		}
	}

	for i := 0; i < 256; i++ {
		if vec.Count() != 256-i {
			t.Errorf("Count %d != expected %d", vec.Count(), 256-i)
		}
		checkSet(t, &vec, uint8(i), false)
		for j := 0; j < 256; j++ {
			// All bits up to i are false, and everything after is true
			if j <= i {
				if vec.Get(uint8(j)) {
					t.Errorf("Bit at %d set", j)
				}
				c := vec.CountLess(uint8(j))
				if c != 0 {
					t.Errorf("CountLess(%d) %d != expected 0", j, c)
				}
			} else {
				if !vec.Get(uint8(j)) {
					t.Errorf("Bit at %d not set", j)
				}
			}
		}
		if vec.Count() != 255-i {
			t.Errorf("Count %d != expected %d", vec.Count(), 255-i)
		}
	}
}

func TestRandom(t *testing.T) {
	rand.Seed(1)

	var vec Bitmap256
	bitsSet := make(map[uint8]bool)

	for i := 0; i < 10; i++ {
		r := uint8(rand.Uint32())
		bitsSet[r] = true
		checkSet(t, &vec, r, true)
	}
	if vec.Count() != len(bitsSet) {
		t.Errorf("Count %d != expected %d", vec.Count(), len(bitsSet))
	}
	lessCount := 0
	for i := 0; i < 256; i++ {
		val := bitsSet[uint8(i)]
		if vec.Get(uint8(i)) != val {
			t.Errorf("Bit %d value %v != expected %v", i, vec.Get(uint8(i)), val)
		}
		c := vec.CountLess(uint8(i))
		if c != lessCount {
			t.Errorf("CountLess(%d) %d != expected %d", i, c, lessCount)
		}
		if val {
			lessCount++
		}
	}
	for i := range bitsSet {
		bitsSet[i] = false
		checkSet(t, &vec, uint8(i), false)
		for i := 0; i < 256; i++ {
			val := bitsSet[uint8(i)]
			if vec.Get(uint8(i)) != val {
				t.Errorf("Bit %d value %v != expected %v", i, vec.Get(uint8(i)), val)
			}
		}
	}
}

func BenchmarkSet(b *testing.B) {
	var vec Bitmap256
	for i := 0; i < b.N; i++ {
		vec.Set(uint8(i))
	}
}

func BenchmarkClear(b *testing.B) {
	var vec Bitmap256
	for i := 0; i < b.N; i++ {
		vec.Clear(uint8(i))
	}
}

var dummyStore int

func BenchmarkGet(b *testing.B) {
	var vec Bitmap256
	for i := 0; i < 64; i++ {
		r := uint8(rand.Uint32())
		vec.Set(r)
	}
	b.ResetTimer()

	z := 0
	for i := 0; i < b.N; i++ {
		if vec.Get(uint8(i)) {
			z++
		}
	}
	dummyStore = z
}

func BenchmarkCount(b *testing.B) {
	var vec Bitmap256
	for i := 0; i < 64; i++ {
		r := uint8(rand.Uint32())
		vec.Set(r)
	}
	b.ResetTimer()

	z := 0
	for i := 0; i < b.N; i++ {
		z += vec.Count()
	}
	dummyStore = z
}

func BenchmarkCountLess(b *testing.B) {
	var vec Bitmap256
	for i := 0; i < 64; i++ {
		r := uint8(rand.Uint32())
		vec.Set(r)
	}
	b.ResetTimer()

	z := 0
	for i := 0; i < b.N; i++ {
		z += vec.CountLess(uint8(i))
	}
	dummyStore = z
}
