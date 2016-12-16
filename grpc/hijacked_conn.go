package grpc

import (
	"bufio"
	"net"
	"sync"
	"time"
)

type hijackedConn struct {
	conn net.Conn

	lock sync.Mutex
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

	return &hijackedConn{conn: conn, bufr: bufr}, nil
}

func (c *hijackedConn) Read(b []byte) (n int, err error) {
	c.lock.Lock()
	if c.bufr != nil {
		if c.bufr.Buffered() > 0 {
			defer c.lock.Unlock()
			readLen := len(b)
			if readLen > c.bufr.Buffered() {
				readLen = c.bufr.Buffered()
			}
			return c.bufr.Read(b[:readLen])
		}
		c.bufr = nil
	}
	c.lock.Unlock()

	return c.conn.Read(b)
}

func (c *hijackedConn) Write(b []byte) (n int, err error) {
	return c.conn.Write(b)
}

func (c *hijackedConn) Close() error {
	return c.conn.Close()
}

func (c *hijackedConn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *hijackedConn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *hijackedConn) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

func (c *hijackedConn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

func (c *hijackedConn) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}
