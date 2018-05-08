package backgroundflush

import (
	"errors"
	"fmt"
	"math/rand"
	"testing"
)

type seqWriter struct {
	seq     int
	maxByte int
}

func (r *seqWriter) Write(buf []byte) (int, error) {
	for i, b := range buf {
		if byte(r.seq%r.maxByte) != b {
			return i, fmt.Errorf("invalid byte %02x at %d, expected %02x", b, i, byte(r.seq%r.maxByte))
		}
		r.seq++
	}
	return len(buf), nil
}

func TestWriter(t *testing.T) {
	// Both prime numbers.
	const maxByte = 127
	const bufSize = 1021

	w := NewWriter(&seqWriter{maxByte: maxByte}, bufSize)

	buf := make([]byte, 256)
	for i := 0; i < 1024*1024; {
		n := rand.Intn(cap(buf))
		buf = buf[:n]
		for k := 0; k < n; k++ {
			buf[k] = byte(i % maxByte)
			i++
		}
		written, err := w.Write(buf)
		if written != n {
			t.Errorf("i: %d, written %d != expected %d", i, written, n)
		}
		if err != nil {
			t.Errorf("i: %d, unexpected error %v", i, err)
		}
		if i%100 == 0 {
			// Call flush multiple times concurrently because this is intended to be
			// safe.
			go w.Flush()
			go w.Flush()
			go w.Flush()
		}
	}

	err := w.Flush()
	if err != nil {
		t.Errorf("unexpected flush error %v", err)
	}
}

type nopWriter struct{}

func (nopWriter) Write(buf []byte) (int, error) {
	return len(buf), nil
}

func TestWriterEmptyFlush(t *testing.T) {
	w := NewWriter(nopWriter{}, 1234)
	err := w.Flush()
	if err != nil {
		t.Errorf("unexpected flush error %v", err)
	}
}

type errWriter struct{}

func (errWriter) Write(buf []byte) (int, error) {
	return 0, errors.New("errWriter Writer error")
}

func TestWriterFlushError(t *testing.T) {
	w := NewWriter(errWriter{}, 1234)
	_, err := w.Write([]byte("hello world"))
	if err != nil {
		t.Errorf("unexpected write error %v", err)
	}
	err = w.Flush()
	if err == nil {
		t.Error("expected flush error")
	}
}

func TestWriterError(t *testing.T) {
	w := NewWriter(errWriter{}, 1000)
	for i := 0; i < 110; i++ {
		_, err := w.Write([]byte("0123456789"))
		if err != nil {
			t.Logf("i: %d, write error: %v", i, err)
			break
		}
	}
	_, err := w.Write([]byte("hello world"))
	if err == nil {
		t.Error("expected write error")
	}
	err = w.Flush()
	if err == nil {
		t.Error("expected flush error")
	}
}
