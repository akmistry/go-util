package parallelwriter

import (
	"container/list"
	"errors"
	"io"
	"sync"

	"github.com/akmistry/go-util/ringbuffer"
)

var (
	ErrClosed = errors.New("buffered_writer: closed")
)

type WriteFlusher interface {
	io.Writer
	Flush() error
}

type nopFlusher struct {
	io.Writer
}

func NopFlusher(w io.Writer) WriteFlusher {
	return &nopFlusher{w}
}

func (*nopFlusher) Flush() error { return nil }

type bufferedOp struct {
	length int
}

type flushOp struct {
	done chan struct{}
	err  error
}

type bufOp struct {
	buf  []byte
	done chan struct{}
	n    int
	err  error
}

type BufferedWriter struct {
	w WriteFlusher

	rb     *ringbuffer.RingBuffer
	closed bool
	err    error
	ops    list.List
	lock   sync.Mutex
	cond   *sync.Cond
}

func NewBufferedWriter(w WriteFlusher, bufferSize int) *BufferedWriter {
	b := &BufferedWriter{
		w: w,
	}
	if bufferSize > 0 {
		b.rb = ringbuffer.NewRingBuffer(make([]byte, bufferSize))
	}
	b.cond = sync.NewCond(&b.lock)
	go b.flusher()
	return b
}

func (b *BufferedWriter) doOp(op interface{}) error {
	switch o := op.(type) {
	case *bufferedOp:
		n := 0
		for n < o.length {
			buf := b.rb.Peek(0)
			writeSize := o.length - n
			if writeSize > len(buf) {
				writeSize = len(buf)
			}
			b.lock.Unlock()
			written, err := b.w.Write(buf[:writeSize])
			b.lock.Lock()
			n += written
			b.rb.Consume(written)
			if err != nil {
				return err
			}
		}
		return nil
	case *flushOp:
		b.lock.Unlock()
		o.err = b.w.Flush()
		b.lock.Lock()
		close(o.done)
		return o.err
	case *bufOp:
		b.lock.Unlock()
		o.n, o.err = b.w.Write(o.buf)
		b.lock.Lock()
		close(o.done)
		return o.err
	default:
		panic("unsupported op type")
	}
}

func (b *BufferedWriter) flusher() {
	b.lock.Lock()
	defer b.lock.Unlock()

	for b.err == nil && !b.closed {
		if b.ops.Len() == 0 {
			b.cond.Wait()
		}
		for b.ops.Len() > 0 && b.err == nil && !b.closed {
			op := b.ops.Remove(b.ops.Back())
			err := b.doOp(op)
			if err != nil {
				b.err = err
			}
		}
	}

	err := b.err
	if err == nil {
		err = ErrClosed
	}
	for b.ops.Len() > 0 {
		op := b.ops.Remove(b.ops.Back())
		switch o := op.(type) {
		case *flushOp:
			o.err = err
			defer close(o.done)
		case *bufOp:
			o.err = err
			defer close(o.done)
		}
	}
}

// Close closes the writer and prevents any new writes from succeeding. This
// does not close the underlying writer which must be closed to ensure any
// pending writes are canceled. It must be safe to close the underlying writer
// from a different goroutine than an active Write.
//
// Close does not block.
func (b *BufferedWriter) Close() error {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.closed = true
	b.cond.Signal()

	return b.err
}

func (b *BufferedWriter) writeBlocking(buf []byte) (int, error) {
	bo := &bufOp{buf: buf, done: make(chan struct{})}
	b.ops.PushFront(bo)
	b.cond.Signal()

	b.lock.Unlock()
	<-bo.done
	b.lock.Lock()

	return bo.n, bo.err
}

func (b *BufferedWriter) Write(buf []byte) (int, error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	if b.err != nil {
		return 0, b.err
	} else if b.closed {
		return 0, ErrClosed
	}

	if b.rb == nil || len(buf) > b.rb.Free() {
		return b.writeBlocking(buf)
	}

	var bo *bufferedOp
	head := b.ops.Front()
	if head != nil {
		bo, _ = head.Value.(*bufferedOp)
	}

	if bo == nil {
		bo = new(bufferedOp)
		b.ops.PushFront(bo)
	}
	bo.length += len(buf)
	n, err := b.rb.Write(buf)
	if err != nil {
		panic(err)
	}
	return n, err
}

func (b *BufferedWriter) Flush() error {
	b.lock.Lock()
	defer b.lock.Unlock()

	if b.err != nil {
		return b.err
	} else if b.closed {
		return ErrClosed
	}

	var fo *flushOp
	head := b.ops.Front()
	if head != nil {
		fo, _ = head.Value.(*flushOp)
	}

	if fo == nil {
		fo = &flushOp{done: make(chan struct{})}
		b.ops.PushFront(fo)
		b.cond.Signal()
	}

	b.lock.Unlock()
	<-fo.done
	b.lock.Lock()

	return fo.err
}
