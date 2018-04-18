package parallelwriter

import (
	"io"
	"sync"

	"github.com/akmistry/go-util/bufferpool"
)

// Writer serialises and batches concurrent writes, to minimise Write calls to
// the underlying io.Writer. This is useful when the underlying io.Writer has
// properties that make each Write expensive, such as sending across a network
// and waiting for an ack, or flushing after every write. If your io.Writer
// does not have this property, you should instead synchronize using a
// sync.Mutex.
type Writer struct {
	w io.Writer

	buf  *bufferpool.Buffer
	done chan struct{}
	err  error
	lock sync.Mutex

	// Ensures only one goroutine is calling w.Write.
	writeLock sync.Mutex
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{w: w}
}

func (w *Writer) Write(b []byte) (int, error) {
	w.lock.Lock()
	defer w.lock.Unlock()

	if w.err != nil {
		return 0, w.err
	}

	if w.buf != nil {
		w.buf.Write(b)
		done := w.done

		w.lock.Unlock()
		<-done
		w.lock.Lock()

		if w.err != nil {
			return 0, w.err
		}
		return len(b), nil
	}

	buf := bufferpool.NewBuffer(b)
	defer buf.Reset()
	done := make(chan struct{})
	defer close(done)

	w.buf = buf
	w.done = done

	w.lock.Unlock()
	w.writeLock.Lock()
	defer w.writeLock.Unlock()
	w.lock.Lock()

	w.buf = nil
	w.done = nil

	if w.err != nil {
		return 0, w.err
	}

	w.lock.Unlock()
	_, err := w.w.Write(buf.Bytes())
	w.lock.Lock()

	w.err = err
	if err != nil {
		return 0, err
	}
	return len(b), nil
}
