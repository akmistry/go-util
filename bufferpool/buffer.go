package bufferpool

type Buffer struct {
	buf     []byte
	donated bool
}

func NewBuffer(buf []byte) *Buffer {
	return &Buffer{
		// Having cap(buf) == len(buf) prevents the original buffer from being
		// modified when new data is appended.
		buf:     buf[0:len(buf):len(buf)],
		donated: true,
	}
}

func (b *Buffer) growIfNecessary(n int) {
	if b.donated || len(b.buf)+n > cap(b.buf) {
		if b.buf == nil && n < MinBufferSize {
			n = MinBufferSize
		}
		newBuf := Get(len(b.buf) + n)[:0]
		newBuf = append(newBuf, b.buf...)
		if b.buf != nil && !b.donated {
			Put(b.buf)
		}
		b.buf = newBuf
		b.donated = false
	}
}

func (b *Buffer) Write(p []byte) (int, error) {
	b.growIfNecessary(len(p))
	b.buf = append(b.buf, p...)
	return len(p), nil
}

func (b *Buffer) Len() int {
	return len(b.buf)
}

func (b *Buffer) Bytes() []byte {
	return b.buf
}

func (b *Buffer) Reset() {
	if b.buf != nil && !b.donated {
		Put(b.buf)
	}
	b.buf = nil
}
