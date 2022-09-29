package tempfile

import (
	"io"
)

type File interface {
	io.ReaderAt
	io.WriterAt
	io.Closer
}
