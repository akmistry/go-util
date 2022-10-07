// Package compressedbuffer implements a compressed, in-memory, random-access
// buffer, similar to bytes.Buffer.
package compressedbuffer

import (
	"bytes"
	"compress/zlib"
	"io"
	"log"
	"sync"
)

const (
	BlockSize = 4096
)

var (
	zWriterPool = sync.Pool{}
	zReaderPool = sync.Pool{}
)

// A Buffer is a variable-sized buffer, with Write and ReadAt methods (Read can
// be done using io.SectionReader). The zero value for Buffer is an empty
// buffer ready to use. Buffer contains internal synchronisation, allowing for
// concurrent use.
type Buffer struct {
	blocks         [][]byte
	size           int64
	compressedSize int64

	writeBuf bytes.Buffer

	lock sync.Mutex
}

func (b *Buffer) flushWriter() error {
	if b.writeBuf.Len() != BlockSize {
		log.Panicf("Invalid flush size %d", b.writeBuf.Len())
	}
	var compBuf bytes.Buffer
	var zw *zlib.Writer
	if zwi := zWriterPool.Get(); zwi != nil {
		zw = zwi.(*zlib.Writer)
		zw.Reset(&compBuf)
	} else {
		var err error
		zw, err = zlib.NewWriterLevel(&compBuf, zlib.BestSpeed)
		if err != nil {
			panic(err)
		}
	}
	defer zWriterPool.Put(zw)
	n, err := b.writeBuf.WriteTo(zw)
	if err != nil {
		return err
	} else if n != BlockSize {
		log.Panicf("Zlib written %d != buffer size %d", n, BlockSize)
	}
	err = zw.Close()
	if err != nil {
		return err
	}
	b.compressedSize += int64(compBuf.Len())
	b.blocks = append(b.blocks, compBuf.Bytes())
	b.writeBuf.Reset()
	return nil
}

func (b *Buffer) Write(p []byte) (int, error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	written := 0
	for len(p) > 0 {
		rem := BlockSize - b.writeBuf.Len()
		writeLen := len(p)
		if writeLen > rem {
			writeLen = rem
		}
		n, err := b.writeBuf.Write(p[:writeLen])
		written += n
		b.size += int64(n)
		p = p[writeLen:]
		if err != nil {
			return written, err
		}
		if b.writeBuf.Len() == BlockSize {
			err = b.flushWriter()
			if err != nil {
				return written, err
			}
		}
	}
	return written, nil
}

func (b *Buffer) Size() int64 {
	b.lock.Lock()
	defer b.lock.Unlock()
	return b.size
}

func (b *Buffer) CompressedSize() int64 {
	b.lock.Lock()
	defer b.lock.Unlock()
	return b.compressedSize + int64(b.writeBuf.Len())
}

func (b *Buffer) readBlock(i int) ([]byte, error) {
	var err error
	var zr io.ReadCloser
	if zri := zReaderPool.Get(); zri != nil {
		zr = zri.(io.ReadCloser)
		err = zr.(zlib.Resetter).Reset(bytes.NewReader(b.blocks[i]), nil)
	} else {
		zr, err = zlib.NewReader(bytes.NewReader(b.blocks[i]))
	}
	if err != nil {
		return nil, err
	}
	defer zReaderPool.Put(zr)
	defer zr.Close()

	buf := make([]byte, BlockSize)
	_, err = io.ReadFull(zr, buf)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func (b *Buffer) ReadAt(p []byte, off int64) (int, error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	bytesRead := 0
	var err error
	for len(p) > 0 {
		if off >= b.size {
			return bytesRead, io.EOF
		}
		block := int(off / BlockSize)
		blockOff := int(off % BlockSize)

		var blockBuf []byte
		if block == len(b.blocks) {
			// Block is currently being written, so use writeBuf as the source
			blockBuf = b.writeBuf.Bytes()
		} else {
			blockBuf, err = b.readBlock(block)
			if err != nil {
				break
			}
		}

		n := copy(p, blockBuf[blockOff:])
		bytesRead += n
		p = p[n:]
		off += int64(n)
	}
	return bytesRead, err
}
