package parallelwriter

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const (
	threads    = 8
	loops      = 1000
	maxPayload = 1024
	bufferSize = 12345
)

type flushWriter struct {
	countingWriter
	flushCount int
}

func (w *flushWriter) Flush() error {
	w.flushCount++
	return nil
}

// TODO: Test flush error.
func TestBufferedWriter(t *testing.T) {
	const flushPeriod = 10

	outBuf := new(flushWriter)
	w := NewBufferedWriter(outBuf, bufferSize)

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
				if j%flushPeriod == 0 {
					err = w.Flush()
					if err != nil {
						t.Errorf("Error flushing: %v", err)
					}
				}
			}
		}()
	}
	wg.Wait()
	w.Flush()

	w.Close()
	buf := generatePacket(maxPayload)
	_, err := w.Write(buf)
	if err != ErrClosed {
		t.Errorf("Unexpected wrror writing: %v", err)
	}
	err = w.Flush()
	if err != ErrClosed {
		t.Errorf("Unexpected error flushing: %v", err)
	}

	if int(written) != outBuf.Len() {
		t.Errorf("written %d != output %d", written, outBuf.Len())
	}

	if outBuf.flushCount < loops/flushPeriod {
		t.Errorf("flush count %d < minimum expected %d", outBuf.flushCount, loops/flushPeriod)
	}

	checkBuffer(t, outBuf.Bytes())
	t.Logf("written bytes: %d, write count: %d, flush count: %d",
		outBuf.Len(), outBuf.count, outBuf.flushCount)
}

func TestBufferedWriterShortWrite(t *testing.T) {
	outBuf := new(flushWriter)
	w := NewBufferedWriter(outBuf, bufferSize)
	time.Sleep(10 * time.Millisecond)

	buf := generatePacket(maxPayload)
	written, err := w.Write(buf)
	if err != nil {
		t.Errorf("Error writing: %v", err)
	}

	// NOTE: Racy!
	start := time.Now()
	for outBuf.count == 0 {
		if time.Since(start) > time.Second {
			t.Fatal("waited too long for write")
		}
		time.Sleep(10 * time.Millisecond)
	}

	if written != outBuf.Len() {
		t.Errorf("written %d != output %d", written, outBuf.Len())
	}

	checkBuffer(t, outBuf.Bytes())
	t.Logf("written bytes: %d, write count: %d, flush count: %d",
		outBuf.Len(), outBuf.count, outBuf.flushCount)
}

func TestBufferedWriterError(t *testing.T) {
	outBuf := &failingWriter{failCount: 1234567}
	w := NewBufferedWriter(NopFlusher(outBuf), bufferSize)

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
