package io

import "io"

type readerAtAdapter struct {
        r   io.ReaderAt
        off int64
}

func NewReader(r io.ReaderAt) io.Reader {
        return &readerAtAdapter{r: r}
}

func (a *readerAtAdapter) Read(p []byte) (int, error) {
        n, err := a.r.ReadAt(p, a.off)
        a.off += int64(n)
        return n, err
}
