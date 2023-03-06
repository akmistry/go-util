package tests

import (
	"fmt"
	"math/rand"
	"testing"
)

// Dummy, to ensure the compiler doesn't optimise away the memory access.
var x uint64

func BenchmarkCacheMiss_Read(b *testing.B) {
	const maxShift = 26
	mem := make([]uint64, (1 << maxShift))
	for i := range mem {
		mem[i] = rand.Uint64()
	}
	b.ResetTimer()

	for shift := 16; shift < maxShift; shift++ {
		memSize := uint64(1 << shift)
		memSizeMask := memSize - 1
		testName := fmt.Sprintf("MemSize%d", memSize*8)
		b.Run(testName, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				index := (uint64(i) * 0x2545F4914F6CDD1D) & memSizeMask
				x = mem[index]
			}
		})
	}
}

func BenchmarkCacheMiss_Write(b *testing.B) {
	const maxShift = 26
	mem := make([]uint64, (1 << maxShift))
	for i := range mem {
		mem[i] = rand.Uint64()
	}
	b.ResetTimer()

	for shift := 16; shift < maxShift; shift++ {
		memSize := uint64(1 << shift)
		memSizeMask := memSize - 1
		testName := fmt.Sprintf("MemSize%d", memSize*8)
		b.Run(testName, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				z := uint64(i) * 0x2545F4914F6CDD1D
				index := z & memSizeMask
				mem[index] = z
			}
		})
	}
}
