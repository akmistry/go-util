package backgroundflush

import (
	"fmt"
	"io"
	"sync"
)

type Writer struct {
	w io.Writer

	writeBuf, flushBuf int
	flushActive        bool
	flushErr           error
	bufs               [2][]byte

	lock sync.Mutex
	cond *sync.Cond
}

func NewWriter(w io.Writer, size int) *Writer {
	wr := &Writer{
		w: w,
	}
	wr.cond = sync.NewCond(&wr.lock)
	for i := range wr.bufs {
		wr.bufs[i] = make([]byte, 0, size/len(wr.bufs))
	}
	return wr
}

func (w *Writer) doFlush() {
	w.lock.Lock()
	defer func() {
		w.flushActive = false
		w.lock.Unlock()
	}()

	for w.flushErr == nil && w.writeBuf > w.flushBuf {
		b := w.bufs[w.flushBuf%2]

		w.lock.Unlock()
		_, err := w.w.Write(b)
		w.lock.Lock()

		w.bufs[w.flushBuf%2] = b[:0]
		w.flushErr = err
		w.flushBuf++
		w.cond.Broadcast()
	}
}

func (w *Writer) startFlush(wait bool) {
	if w.writeBuf < w.flushBuf+2 && len(w.bufs[w.writeBuf%2]) > 0 {
		w.writeBuf++
	}

	target := w.writeBuf
	if !w.flushActive {
		w.flushActive = true
		go w.doFlush()
	}
	for wait && w.flushErr == nil && w.flushBuf < target {
		w.cond.Wait()
	}
}

func (w *Writer) Flush() error {
	w.lock.Lock()
	defer w.lock.Unlock()
	if w.flushErr != nil {
		return w.flushErr
	}
	w.startFlush(true)
	return w.flushErr
}

func (w *Writer) Write(b []byte) (int, error) {
	w.lock.Lock()
	defer w.lock.Unlock()

	n := 0
	for len(b) > 0 && w.flushErr == nil {
		if w.writeBuf > w.flushBuf+1 {
			if w.writeBuf > w.flushBuf+2 {
				panic(fmt.Sprintf("unexpected writeBuf %d, flushBuf %d", w.writeBuf, w.flushBuf))
			}
			w.cond.Wait()
			continue
		}

		wb := w.bufs[w.writeBuf%2]
		rem := cap(wb) - len(wb)
		writeLen := len(b)
		if writeLen > rem {
			writeLen = rem
		}
		w.bufs[w.writeBuf%2] = append(wb, b[:writeLen]...)
		n += writeLen
		b = b[writeLen:]
		if len(w.bufs[w.writeBuf%2]) == cap(w.bufs[w.writeBuf%2]) {
			w.startFlush(false)
		}
	}

	return n, w.flushErr
}
