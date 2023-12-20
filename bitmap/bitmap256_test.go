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

func checkFullEmpty(t *testing.T, v *Bitmap256) {
	t.Helper()

	if v.Full() != (v.Count() == 256) {
		t.Errorf("Unexpected Full %v : count %d", v.Full(), v.Count())
	}
	if v.Empty() != (v.Count() == 0) {
		t.Errorf("Unexpected Empty %v : count %d", v.Empty(), v.Count())
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
			ffs := vec.FindFirstSet()
			if ffs != 0 {
				t.Errorf("FindFirstSet() %d != expected 0", ffs)
			}
		}
		if vec.Count() != i+1 {
			t.Errorf("Count %d != expected %d", vec.Count(), i+1)
		}
		checkFullEmpty(t, &vec)
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
			ffs := vec.FindFirstSet()
			if ffs != i+1 {
				t.Errorf("FindFirstSet() %d != expected %d", ffs, i+1)
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

func TestFindFirstSet_Stress(t *testing.T) {
	emptyFfs := (&Bitmap256{}).FindFirstSet()
	if emptyFfs != 256 {
		t.Errorf("FindFirstSet() %d != expected 256", emptyFfs)
	}

	for i := 0; i < 1000; i++ {
		var vec Bitmap256
		for j := 0; j < 4; j++ {
			checkSet(t, &vec, uint8(rand.Uint32()), true)
		}

		ffs := vec.FindFirstSet()
		expectedFfs := 0
		for ; expectedFfs < 256 && !vec.Get(uint8(expectedFfs)); expectedFfs++ {
		}
		if ffs != expectedFfs {
			t.Errorf("FindFirstSet() %d != expected %d", ffs, expectedFfs)
		}
	}
}

func TestFindNextSet(t *testing.T) {
	var vec Bitmap256
	// Empty bitmap
	for i := 0; i < 256; i++ {
		next := vec.FindNextSet(uint8(i))
		if next != 256 {
			t.Errorf("empty set next %d != 256", next)
		}
	}

	// Full bitmap
	for i := 0; i < 256; i++ {
		vec.Set(uint8(i))
	}
	for i := 0; i < 256; i++ {
		next := vec.FindNextSet(uint8(i))
		if next != i {
			t.Errorf("full set next %d != %d", next, i)
		}
	}

	// Single bit set, exhaustive test
	vec = Bitmap256{}
	for i := 0; i < 256; i++ {
		vec.Set(uint8(i))
		for j := 0; j <= i; j++ {
			next := vec.FindNextSet(uint8(j))
			if next != i {
				t.Errorf("single vec(%d) next(%d) %d != %d",
					i, j, next, i)
			}
		}
		for j := i + 1; j < 256; j++ {
			next := vec.FindNextSet(uint8(j))
			if next != 256 {
				t.Errorf("single vec(%d) next(%d) %d != 256",
					i, j, next)
			}
		}
		vec.Clear(uint8(i))
	}
}

func TestFindNextSet_Stress(t *testing.T) {
	const Iterations = 10000
	const MaxSet = 16

	for i := 0; i < Iterations; i++ {
		var vec Bitmap256
		numSet := rand.Intn(MaxSet)
		for j := 0; j < numSet; j++ {
			vec.Set(uint8(rand.Uint32()))
		}

		// Exhaustively check every case for this bit pattern
		for j := 0; j < 256; j++ {
			start := uint8(j)
			next := vec.FindNextSet(start)
			//t.Logf("start %d, next %d", start, next)
			if next < int(start) {
				t.Errorf("next %d < start %d", next, start)
			} else if next == int(start) {
				if !vec.Get(start) {
					t.Errorf("start == next %d != true", next)
				}
			} else {
				for k := int(start); k < next; k++ {
					if vec.Get(uint8(k)) {
						t.Errorf("Unexpected set element at %d for start = %d, next = %d",
							k, start, next)
					}
				}
				if next < 256 && !vec.Get(uint8(next)) {
					t.Errorf("next %d not set", next)
				}
			}
		}
	}
}

func TestFindNextClear(t *testing.T) {
	var vec Bitmap256
	// Empty bitmap
	for i := 0; i < 256; i++ {
		next := vec.FindNextClear(uint8(i))
		if next != i {
			t.Errorf("empty set next %d != %d", next, i)
		}
	}

	// Full bitmap
	for i := 0; i < 256; i++ {
		vec.Set(uint8(i))
	}
	for i := 0; i < 256; i++ {
		next := vec.FindNextClear(uint8(i))
		if next != 256 {
			t.Errorf("full set next %d != 256", next)
		}
	}

	// Single bit unset, exhaustive test
	for i := 0; i < 256; i++ {
		vec.Clear(uint8(i))
		for j := 0; j <= i; j++ {
			next := vec.FindNextClear(uint8(j))
			if next != i {
				t.Errorf("single vec(%d) next(%d) %d != %d",
					i, j, next, i)
			}
		}
		for j := i + 1; j < 256; j++ {
			next := vec.FindNextClear(uint8(j))
			if next != 256 {
				t.Errorf("single vec(%d) next(%d) %d != 256",
					i, j, next)
			}
		}
		vec.Set(uint8(i))
	}
}

func TestFindNextClear_Stress(t *testing.T) {
	const Iterations = 10000
	const MaxClear = 16

	for i := 0; i < Iterations; i++ {
		var vec Bitmap256
		for j := 0; j < 256; j++ {
			vec.Set(uint8(j))
		}
		numSet := rand.Intn(MaxClear)
		for j := 0; j < numSet; j++ {
			vec.Clear(uint8(rand.Uint32()))
		}

		// Exhaustively check every case for this bit pattern
		for j := 0; j < 256; j++ {
			start := uint8(j)
			next := vec.FindNextClear(start)
			//t.Logf("start %d, next %d", start, next)
			if next < int(start) {
				t.Errorf("next %d < start %d", next, start)
			} else if next == int(start) {
				if vec.Get(start) {
					t.Errorf("start == next %d != false", next)
				}
			} else {
				for k := int(start); k < next; k++ {
					if !vec.Get(uint8(k)) {
						t.Errorf("Unexpected clear element at %d for start = %d, next = %d",
							k, start, next)
					}
				}
				if next < 256 && vec.Get(uint8(next)) {
					t.Errorf("next %d not clear", next)
				}
			}
		}
	}
}
func checkFindNth(t *testing.T, v *Bitmap256, i uint8, expected int) {
	t.Helper()
	p := v.FindNthSet(i)
	if p != expected {
		t.Errorf("FindNthSet(%d) %d != expected %d", i, p, expected)
	}
}

func TestFindNthSet(t *testing.T) {
	for i := 0; i < 256; i++ {
		emptyFns := (&Bitmap256{}).FindNthSet(uint8(i))
		if emptyFns != 256 {
			t.Errorf("FindNthSet(%d) %d != expected 256", i, emptyFns)
		}
	}

	var bm Bitmap256
	bm.Set(0)
	checkFindNth(t, &bm, 0, 0)
	checkFindNth(t, &bm, 1, 256)
	checkFindNth(t, &bm, 66, 256)

	bm.Set(1)
	checkFindNth(t, &bm, 0, 0)
	checkFindNth(t, &bm, 1, 1)
	checkFindNth(t, &bm, 2, 256)
	checkFindNth(t, &bm, 66, 256)

	bm.Clear(0)
	checkFindNth(t, &bm, 0, 1)
	checkFindNth(t, &bm, 1, 256)
	checkFindNth(t, &bm, 66, 256)

	bm.Set(63)
	bm.Set(64)
	bm.Set(65)
	checkFindNth(t, &bm, 0, 1)
	checkFindNth(t, &bm, 1, 63)
	checkFindNth(t, &bm, 2, 64)
	checkFindNth(t, &bm, 3, 65)
	checkFindNth(t, &bm, 4, 256)
	checkFindNth(t, &bm, 66, 256)

	bm = Bitmap256{}
	for i := 0; i < 256; i++ {
		bm.Set(uint8(i))

		for j := 0; j < 256; j++ {
			if j > i {
				checkFindNth(t, &bm, uint8(j), 256)
			} else {
				checkFindNth(t, &bm, uint8(j), j)
			}
		}
	}
	for i := 0; i < 256; i++ {
		checkFindNth(t, &bm, uint8(i), i)
	}
}

func TestFindNthSet_Stress(t *testing.T) {
	for i := 0; i < 1000; i++ {
		var vec Bitmap256

		numSet := rand.Intn(256)
		for i := 0; i < numSet; i++ {
			checkSet(t, &vec, uint8(rand.Uint32()), true)
		}

		count := 0
		for pos := 0; pos < 256; pos++ {
			if vec.Get(uint8(pos)) {
				resultPos := vec.FindNthSet(uint8(count))
				if resultPos != pos {
					t.Errorf("FindNthSet(%d) %d != expected %d", count, resultPos, pos)
				}
				count++
			}
		}
		if count < 256 {
			resultPos := vec.FindNthSet(uint8(count))
			if resultPos != 256 {
				t.Errorf("FindNthSet(%d) %d != expected 256", count, resultPos)
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

	z := false
	for i := 0; i < b.N; i++ {
		z = vec.Get(uint8(i)) || z
	}
	if !z {
		dummyStore = 7
	}
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

func BenchmarkFindFirstSet(b *testing.B) {
	var vec Bitmap256
	// Worst case
	vec.Set(255)

	z := 0
	for i := 0; i < b.N; i++ {
		z += vec.FindFirstSet()
	}
	dummyStore = z
}

func BenchmarkFindNextSet_Best(b *testing.B) {
	var vec Bitmap256
	// Best case. Using a random number prevents the compiler from
	// optimising away most code.
	k := uint8(rand.Intn(8))
	vec.Set(k + 1)

	z := 0
	for i := 0; i < b.N; i++ {
		z += vec.FindNextSet(k)
	}
	dummyStore = z
}

func BenchmarkFindNextSet_Worst(b *testing.B) {
	var vec Bitmap256
	// Worst case
	vec.Set(255)

	// Using a random number prevents the compiler from optimising away most
	// code.
	k := uint8(rand.Intn(8))
	z := 0
	for i := 0; i < b.N; i++ {
		z += vec.FindNextSet(k)
	}
	dummyStore = z
}

func BenchmarkFindNthSet(b *testing.B) {
	var vec Bitmap256
	// Worst case
	for i := 0; i < 256; i++ {
		vec.Set(uint8(i))
	}

	// Using a random number prevents the compiler from optimising away most
	// code.
	k := uint8(250 + rand.Intn(6))
	z := 0
	for i := 0; i < b.N; i++ {
		z += vec.FindNthSet(k)
	}
	dummyStore = z
}

func BenchmarkEmpty(b *testing.B) {
	var vec Bitmap256
	for i := 0; i < 64; i++ {
		r := uint8(rand.Uint32())
		vec.Set(r)
	}
	b.ResetTimer()

	z := false
	for i := 0; i < b.N; i++ {
		z = vec.Empty() || z
	}
	if z {
		b.Logf("Unreachable: %v", z)
	}
}

func BenchmarkFull(b *testing.B) {
	var vec Bitmap256
	for i := 0; i < 64; i++ {
		r := uint8(rand.Uint32())
		vec.Set(r)
	}
	b.ResetTimer()

	z := false
	for i := 0; i < b.N; i++ {
		z = vec.Full() || z
	}
	if z {
		b.Logf("Unreachable: %v", z)
	}
}
