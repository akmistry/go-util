package ringbuffer

import (
	"errors"
)

var (
	ErrBufferFull = errors.New("ringbuffer: buffer full")
)

// RingBuffer is an unsynchronized ring buffer. Any concurrent use MUST be
// externally synchronized.
type RingBuffer struct {
	buf           []byte
	start, length int
}

func NewRingBuffer(buf []byte) *RingBuffer {
	if len(buf) < 1 {
		panic("buffer length too small")
	}
	return &RingBuffer{buf: buf}
}

func (r *RingBuffer) Len() int {
	return r.length
}

func (r *RingBuffer) Cap() int {
	return len(r.buf)
}

func (r *RingBuffer) Free() int {
	return len(r.buf) - r.length
}

func (r *RingBuffer) writeBlock(b []byte) int {
	end := (r.start + r.length) % len(r.buf)
	rem := len(r.buf) - r.length
	copyLen := len(b)
	if copyLen > rem {
		copyLen = rem
	}
	if end+copyLen > len(r.buf) {
		copyLen = len(r.buf) - end
	}
	copied := copy(r.buf[end:end+copyLen], b)
	if copied != copyLen {
		panic("copied != copyLen")
	}
	r.length += copied
	return copied
}

func (r *RingBuffer) Append(b []byte) (int, error) {
	n := 0
	for n < len(b) {
		written := r.writeBlock(b[n:])
		n += written
		if written == 0 {
			break
		}
	}
	if n != len(b) && r.Free() > 0 {
		panic("unexpected short write")
	}
	if n < len(b) {
		return n, ErrBufferFull
	}
	return n, nil
}

func (r *RingBuffer) Peek(off int) []byte {
	if off > r.length {
		panic("off > length")
	}
	peekStart := (r.start + off) % len(r.buf)
	peekLen := r.length - off
	if peekStart+peekLen > len(r.buf) {
		peekLen = len(r.buf) - peekStart
	}
	return r.buf[peekStart : peekStart+peekLen]
}

func (r *RingBuffer) Fetch(b []byte, off int) int {
	if off > r.length {
		panic("off > length")
	}
	fetchStart := (r.start + off) % len(r.buf)
	fetchLen := len(b)
	if fetchLen > (r.length - off) {
		fetchLen = r.length - off
	}

	// Data before the wrap around
	n := fetchLen
	if (fetchStart + n) > len(r.buf) {
		n = len(r.buf) - fetchStart
	}
	if copy(b, r.buf[fetchStart:fetchStart+n]) != n {
		panic("copied bytes != n")
	}

	// And after the wrap around
	if copy(b[n:], r.buf[:fetchLen-n]) != (fetchLen - n) {
		panic("copied bytes != (fetchLen - n)")
	}

	return fetchLen
}

func (r *RingBuffer) Consume(count int) {
	if count > r.length {
		panic("consume count > length")
	}
	r.start = (r.start + count) % len(r.buf)
	r.length -= count
	if r.length == 0 {
		// If the ring is enpty, reset the head so that all bytes are contiguous.
		// This is an optimisation and not necessary for correctness.
		r.start = 0
	}
}
