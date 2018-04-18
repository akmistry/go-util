package parallelwriter

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/fnv"
	"math/rand"
	"sync"
	"testing"
	"time"
)

const headerLen = 8

func generatePacket(maxPayload int) []byte {
	l := rand.Intn(maxPayload) + headerLen
	buf := make([]byte, l)
	rand.Read(buf[headerLen:])
	binary.LittleEndian.PutUint32(buf, uint32(l))

	h := fnv.New32a()
	h.Write(buf[headerLen:])
	binary.LittleEndian.PutUint32(buf[4:], h.Sum32())

	return buf
}

func verifyPacket(buf []byte) (int, error) {
	length := binary.LittleEndian.Uint32(buf)
	hash := binary.LittleEndian.Uint32(buf[4:])

	h := fnv.New32a()
	h.Write(buf[headerLen:length])
	if h.Sum32() != hash {
		return 0, fmt.Errorf("hash %x != header hash %x", h.Sum32(), hash)
	}

	return int(length), nil
}

func checkBuffer(t *testing.T, buf []byte) {
	t.Helper()

	for len(buf) > 0 {
		n, err := verifyPacket(buf)
		if err != nil {
			t.Fatalf("Error verifying packet: %v", err)
		}
		buf = buf[n:]
	}
}

type countingWriter struct {
	bytes.Buffer
	count int
}

func (w *countingWriter) Write(buf []byte) (int, error) {
	// Blocking the write for a little bit causes more batching.
	time.Sleep(time.Microsecond)
	w.count++
	return w.Buffer.Write(buf)
}

func TestWriter(t *testing.T) {
	const threads = 8
	const loops = 1000
	const maxPayload = 1024

	outBuf := new(countingWriter)
	w := NewWriter(outBuf)

	var wg sync.WaitGroup
	wg.Add(threads)
	for i := 0; i < threads; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < loops; j++ {
				buf := generatePacket(maxPayload)
				_, err := w.Write(buf)
				if err != nil {
					t.Errorf("Error writing: %v", err)
				}
			}
		}()
	}
	wg.Wait()

	checkBuffer(t, outBuf.Bytes())
	t.Logf("written bytes: %d, write count: %d", outBuf.Len(), outBuf.count)
}

var errFail = errors.New("fail")

type failingWriter struct {
	bytes.Buffer
	failCount int
	count     int
}

func (w *failingWriter) Write(buf []byte) (int, error) {
	// Blocking the write for a little bit causes more batching.
	time.Sleep(time.Microsecond)
	w.count++
	if w.Len() > w.failCount {
		return 0, errFail
	}
	return w.Buffer.Write(buf)
}

func TestWriterError(t *testing.T) {
	const threads = 8
	const loops = 1000
	const maxPayload = 1024

	outBuf := &failingWriter{failCount: 1234567}
	w := NewWriter(outBuf)

	var wg sync.WaitGroup
	wg.Add(threads)
	for i := 0; i < threads; i++ {
		go func() {
			defer wg.Done()
			expectFail := false
			for j := 0; j < loops; j++ {
				buf := generatePacket(maxPayload)
				_, err := w.Write(buf)
				if expectFail && err == nil {
					t.Errorf("expected failure")
				} else if err != nil {
					expectFail = true
				}
			}
		}()
	}
	wg.Wait()

	checkBuffer(t, outBuf.Bytes())
	t.Logf("written bytes: %d, write count: %d", outBuf.Len(), outBuf.count)
}
