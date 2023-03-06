package tests

import (
	"math/rand"
	"runtime"
	"testing"
)

type simpleInterface interface {
	Foo() int
}

type simpleValue struct {
	a int
}

func (v *simpleValue) Foo() int {
	return v.a
}

func TestSlicing(t *testing.T) {
	const NumValues = 100
	// Create a slice and insert random values
	var s []simpleInterface
	for i := 0; i < NumValues; i++ {
		v := &simpleValue{rand.Int()}
		s = append(s, v)
	}
	if len(s) != NumValues {
		t.Fatalf("len(s) %d != %d", len(s), NumValues)
	}

	// Make the slice much smaller
	s = s[:NumValues/10]
	t.Logf("len(s) = %d, cap(s) = %d", len(s), cap(s))
	runtime.GC()
	t.Logf("After GC: len(s) = %d, cap(s) = %d", len(s), cap(s))

	// Reslice back
	s = s[:NumValues]
	t.Logf("After reslice: len(s) = %d, cap(s) = %d", len(s), cap(s))

	// Check to see if slice is still populated
	for i, sv := range s {
		if sv == nil {
			t.Errorf("s[%d] == nil", i)
		}
		v := sv.(*simpleValue)
		t.Logf("s[%d] = %d", i, v.a)
	}
}
