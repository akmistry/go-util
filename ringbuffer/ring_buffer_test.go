package ringbuffer

import (
	"bytes"
	"math/rand"
	"testing"
)

func TestRingBuffer(t *testing.T) {
	const bufSize = 1234
	const testSize = 1234567
	buf := make([]byte, bufSize)
	testBuf := make([]byte, testSize)
	rand.Read(testBuf)

	written := 0
	read := 0
	writes := 0
	fullErrors := 0
	rb := NewRingBuffer(buf)
	outBuf := new(bytes.Buffer)
	for written < len(testBuf) {
		writeSize := rand.Intn(rb.Cap())
		if written+writeSize > len(testBuf) {
			writeSize = len(testBuf) - written
		}
		writes++
		n, err := rb.Append(testBuf[written : written+writeSize])
		written += n
		if err == ErrBufferFull {
			fullErrors++
		} else if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		readBuf := rb.Peek(0)
		if len(readBuf) == 0 {
			t.Error("read len 0")
		}
		readSize := rand.Intn(len(readBuf) + 1)
		outBuf.Write(readBuf[:readSize])
		rb.Consume(readSize)
		read += readSize
	}

	if read+rb.Len() != testSize {
		t.Errorf("read %d + used %d != test size %d", read, rb.Len(), testSize)
	}

	if !bytes.Equal(testBuf[:read], outBuf.Bytes()) {
		t.Error("read data != written")
	}
	t.Logf("writes: %d, write errors: %d, read bytes: %d", writes, fullErrors, read)
}
