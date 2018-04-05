package listeners

import (
	"net"
)

type filteredListener struct {
	net.Listener
	f func(net.Conn) bool
}

func NewFiltered(l net.Listener, f func(net.Conn) bool) net.Listener {
	return &filteredListener{
		Listener: l,
		f:        f,
	}
}

func (l *filteredListener) Accept() (net.Conn, error) {
	for {
		conn, err := l.Listener.Accept()
		if err != nil {
			return conn, err
		}
		if !l.f(conn) {
			conn.Close()
			continue
		}
		return conn, nil
	}
}
