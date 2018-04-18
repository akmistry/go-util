package parallelwriter

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/fnv"
	"math/rand"
	"sync"
	"sync/atomic"
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
	if len(buf) < 4 {
		// Incomplete packet, not an error.
		return len(buf), nil
	}
	length := binary.LittleEndian.Uint32(buf)
	if len(buf) < int(length) {
		// Incomplete packet, not an error.
		return len(buf), nil
	}
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

	var written int32
	var wg sync.WaitGroup
	wg.Add(threads)
	for i := 0; i < threads; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < loops; j++ {
				buf := generatePacket(maxPayload)
				n, err := w.Write(buf)
				if err != nil {
					t.Errorf("Error writing: %v", err)
				}
				atomic.AddInt32(&written, int32(n))
			}
		}()
	}
	wg.Wait()

	if int(written) != outBuf.Len() {
		t.Errorf("written %d != output %d", written, outBuf.Len())
	}

	checkBuffer(t, outBuf.Bytes())
	t.Logf("written bytes: %d, write count: %d", outBuf.Len(), outBuf.count)
}

var errFail = errors.New("fail")

type failingWriter struct {
	bytes.Buffer
	failCount int
	count     int
	failed    bool
}

func (w *failingWriter) Write(buf []byte) (int, error) {
	if w.failed {
		panic("Write call after failure")
	}
	// Blocking the write for a little bit causes more batching.
	time.Sleep(time.Microsecond)
	w.count++
	if w.Len() > w.failCount {
		w.failed = true
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

	var written int32
	var wg sync.WaitGroup
	wg.Add(threads)
	for i := 0; i < threads; i++ {
		go func() {
			defer wg.Done()
			expectFail := false
			for j := 0; j < loops; j++ {
				buf := generatePacket(maxPayload)
				n, err := w.Write(buf)
				if expectFail && err == nil {
					t.Errorf("expected failure")
				} else if err != nil {
					expectFail = true
				}
				atomic.AddInt32(&written, int32(n))
			}
		}()
	}
	wg.Wait()

	if int(written) > outBuf.Len() {
		t.Errorf("written %d > output %d", written, outBuf.Len())
	} else if int(written) < outBuf.Len()-(threads*maxPayload) {
		t.Errorf("written %d < output-(threads*maxPayload) %d", written, outBuf.Len()-(threads*maxPayload))
	}

	checkBuffer(t, outBuf.Bytes())
	t.Logf("written bytes: %d, write count: %d", outBuf.Len(), outBuf.count)
}
