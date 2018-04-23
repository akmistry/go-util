package parallelwriter

import (
	"sync"
	"sync/atomic"
	"testing"
)

// TODO: Test flush error.
func TestPipelinedFlushWriter(t *testing.T) {
	outBuf := new(flushWriter)
	w := NewPipelinedFlushWriter(outBuf)

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

	if outBuf.flushCount != outBuf.count {
		t.Errorf("flush count %d != write count %d", outBuf.flushCount, outBuf.count)
	}

	checkBuffer(t, outBuf.Bytes())
	t.Logf("written bytes: %d, write count: %d, flush count: %d",
		outBuf.Len(), outBuf.count, outBuf.flushCount)
}

func TestPipelinedFlushWriterError(t *testing.T) {
	outBuf := &failingWriter{failCount: 1234567}
	w := NewPipelinedFlushWriter(NopFlusher(outBuf))

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
