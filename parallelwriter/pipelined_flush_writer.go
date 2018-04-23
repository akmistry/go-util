package parallelwriter

import (
	"sync"
)

type pipelinedWriteAdapter struct {
	w *PipelinedFlushWriter
}

func (a *pipelinedWriteAdapter) Write(buf []byte) (int, error) {
	n, err := a.w.wf.Write(buf)
	if err != nil {
		return n, err
	}

	// Ensure that only one flush is running at a time, and block writes
	// from completing unless it can start a flush.
	a.w.flushLock.Lock()

	flushCh := make(chan struct{})
	a.w.lock.Lock()
	a.w.flushDone = flushCh
	a.w.lock.Unlock()

	go func() {
		defer a.w.flushLock.Unlock()
		err := a.w.wf.Flush()

		if err != nil {
			a.w.lock.Lock()
			a.w.flushErr = err
			a.w.lock.Unlock()
		}
		close(flushCh)
	}()
	return n, err
}

// PipelinedFlushWriter is a writer that serialises and batches parallel
// writes, similar to Writer, but with two additional properties:
// 1. Flush is performed after every Write.
// 2. Flush and Write are pipelined, so that a Flush may occur in parallel
//    with a Write, but Writes are serialised, and Flushes are serialised.
//
// This behaviour is particularly useful for write-ahead logs, where the caller
// wants to ensure writes are flushed to disk, but also wants to do many writes
// in parallel for high total throughput.
type PipelinedFlushWriter struct {
	wf WriteFlusher
	w  *Writer

	flushLock sync.Mutex

	flushDone chan struct{}
	flushErr  error
	lock      sync.RWMutex
}

func NewPipelinedFlushWriter(wf WriteFlusher) *PipelinedFlushWriter {
	pfw := &PipelinedFlushWriter{wf: wf}
	pfw.w = NewWriter(&pipelinedWriteAdapter{w: pfw})
	return pfw
}

// Write serialises, batches, and flushes writes to the underlying Writer.
// Parallel writes are atomic, and buffers are never interleaved or broken to
// the underlying Writer.
func (w *PipelinedFlushWriter) Write(buf []byte) (int, error) {
	n, err := w.w.Write(buf)
	if err != nil {
		return n, err
	}

	w.lock.RLock()
	defer w.lock.RUnlock()
	ch := w.flushDone

	w.lock.RUnlock()
	<-ch
	w.lock.RLock()

	err = w.flushErr
	if err != nil {
		return 0, err
	}
	return n, nil
}
