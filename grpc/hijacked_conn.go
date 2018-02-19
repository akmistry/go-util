package grpc

import (
	"bufio"
	"net"
)

type hijackedConn struct {
	net.Conn

	bufr *bufio.Reader
}

func newHijackedConn(conn net.Conn, bufr *bufio.Reader, bufw *bufio.Writer) (_ *hijackedConn, err error) {
	defer func() {
		if err != nil {
			conn.Close()
		}
	}()
	if bufw != nil {
		err = bufw.Flush()
		if err != nil {
			return
		}
	}

	return &hijackedConn{Conn: conn, bufr: bufr}, nil
}

func (c *hijackedConn) Read(b []byte) (n int, err error) {
	if c.bufr != nil {
		if c.bufr.Buffered() > 0 {
			readLen := len(b)
			if readLen > c.bufr.Buffered() {
				// Don't read more than the buffered data to prevent bufr from re-filling
				// the buffer.
				readLen = c.bufr.Buffered()
			}
			return c.bufr.Read(b[:readLen])
		}
		c.bufr = nil
	}

	return c.Conn.Read(b)
}
