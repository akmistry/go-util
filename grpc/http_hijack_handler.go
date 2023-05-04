package grpc

import (
	"io"
	"log"
	"net"
	"net/http"
	"sync"
)

const (
	hijackedNet = "hijacked-http"
)

type hijackHandler struct {
	lock sync.Mutex

	done       <-chan bool
	closedDone chan<- bool

	connCh chan net.Conn
}

func NewHijackHandlerListener() (http.Handler, net.Listener) {
	done := make(chan bool)
	h := &hijackHandler{done: done, closedDone: done, connCh: make(chan net.Conn)}
	return h, h
}

// http.Handler methods.
func (h *hijackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.ProtoMajor > 1 {
		http.Error(w, "Hijacking not supported on HTTP >= 2.0", http.StatusInternalServerError)
		return
	}
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Server does not suport hijacking", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	conn, bufrw, err := hj.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	hc, err := newHijackedConn(conn, bufrw.Reader, bufrw.Writer)
	if err != nil {
		log.Println("Error hijacking connection:", err)
		return
	}

	select {
	case h.connCh <- hc:
	case <-h.done:
		hc.Close()
	}
}

// net.Listener methods.
func (h *hijackHandler) Accept() (net.Conn, error) {
	select {
	case <-h.done:
		return nil, &net.OpError{Op: "accept", Net: hijackedNet, Err: io.ErrClosedPipe}
	case conn := <-h.connCh:
		return conn, nil
	}
}

func (h *hijackHandler) Close() error {
	h.lock.Lock()
	ch := h.closedDone
	h.closedDone = nil
	h.lock.Unlock()
	if ch != nil {
		close(ch)
	}
	return nil
}

type hijackedAddr string

func (a hijackedAddr) Network() string {
	return hijackedNet
}

func (a hijackedAddr) String() string {
	return string(a)
}

func (h *hijackHandler) Addr() net.Addr {
	return hijackedAddr("hijacked-http:TODO")
}
