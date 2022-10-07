package compressedbuffer

import (
	"bytes"
	"io"
	"math/rand"
	"testing"
)

func checkWriteRead(t *testing.T, b *Buffer, size int) {
	t.Helper()

	buf := make([]byte, size)
	rand.Read(buf)
	writeOffset := b.Size()
	n, err := b.Write(buf)
	if err != nil {
		t.Errorf("Unexpected write error: %v", err)
	} else if n != size {
		t.Errorf("Write size %d != expected %d", n, size)
	}
	if b.Size() != writeOffset+int64(size) {
		t.Errorf("Buffer size %d != expected %d", b.Size(), writeOffset+int64(size))
	}

	readBuf := make([]byte, size)
	n, err = b.ReadAt(readBuf, writeOffset)
	if err != nil {
		t.Errorf("Unexpected read error: %v", err)
	} else if n != size {
		t.Errorf("Read size %d != expected %d", n, size)
	} else if !bytes.Equal(readBuf, buf) {
		t.Error("Bytes read != written")
	}

	t.Logf("Written size: %d, compressed size: %d", b.Size(), b.CompressedSize())
}

func TestBuffer(t *testing.T) {
	rand.Seed(1)

	var b Buffer
	checkWriteRead(t, &b, 1)
	checkWriteRead(t, &b, BlockSize-1)
	checkWriteRead(t, &b, BlockSize)
	checkWriteRead(t, &b, BlockSize+1)
	checkWriteRead(t, &b, 1234)
	checkWriteRead(t, &b, 4*BlockSize)
}

func TestBufferCompressible(t *testing.T) {
	const TestSize = 1234567
	const Period = 12345

	buf := make([]byte, TestSize)
	for i := range buf {
		buf[i] = byte(i % Period)
	}

	var b Buffer
	n, err := b.Write(buf)
	if err != nil {
		t.Errorf("Unexpected write error: %v", err)
	} else if n != TestSize {
		t.Errorf("Write size %d != expected %d", n, TestSize)
	}
	t.Logf("Written size: %d, compressed size: %d", b.Size(), b.CompressedSize())

	readBuf := make([]byte, TestSize)
	n, err = b.ReadAt(readBuf, 0)
	if err != nil {
		t.Errorf("Unexpected read error: %v", err)
	} else if n != TestSize {
		t.Errorf("Read size %d != expected %d", n, TestSize)
	} else if !bytes.Equal(readBuf, buf) {
		t.Error("Bytes read != written")
	}
}

func TestBufferStress(t *testing.T) {
	const Iterations = 1000
	const MaxOpSize = 2 * BlockSize

	rand.Seed(3)
	var writtenBuf bytes.Buffer

	var b Buffer
	for i := 0; i < Iterations; i++ {
		writeBuf := make([]byte, rand.Intn(MaxOpSize))
		rand.Read(writeBuf)
		writtenBuf.Write(writeBuf)

		n, err := b.Write(writeBuf)
		if err != nil {
			t.Errorf("Unexpected write error: %v", err)
		} else if n != len(writeBuf) {
			t.Errorf("Write size %d != expected %d", n, len(writeBuf))
		}

		readOff := rand.Int63n(b.Size())
		readLen := rand.Intn(MaxOpSize)
		readBuf := make([]byte, readLen)

		n, err = b.ReadAt(readBuf, readOff)
		if err == io.EOF && (readOff+int64(readLen)) < b.Size() {
			t.Errorf("Unexpected EOF for offset %d, len %d with buffer size %d",
				readOff, readLen, b.Size())
		} else if err != nil && err != io.EOF {
			t.Errorf("Unexpected read error: %v", err)
		}
		if n < readLen && err != io.EOF {
			t.Errorf("Expected EOF when read bytes %d < expected %d", n, readLen)
		}

		if !bytes.Equal(readBuf[:n], writtenBuf.Bytes()[int(readOff):int(readOff)+n]) {
			t.Error("Bytes read != written")
		}
	}

	readFull := make([]byte, writtenBuf.Len())
	_, err := b.ReadAt(readFull, 0)
	if err != nil && err != io.EOF {
		t.Errorf("Unexpected read error: %v", err)
	} else if !bytes.Equal(readFull, writtenBuf.Bytes()) {
		t.Error("Bytes read != written")
	}
}
