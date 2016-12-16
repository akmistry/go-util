package grpc

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"time"
)

type Dialer func(string, time.Duration) (net.Conn, error)

func NewDialer(urlStr string) (Dialer, error) {
	_, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	return func(addr string, timeout time.Duration) (net.Conn, error) {
		return dialAndHijack(urlStr, timeout)
	}, nil
}

func dialAndHijack(urlStr string, timeout time.Duration) (net.Conn, error) {
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	var conn net.Conn
	if parsed.Scheme == "https" {
		conn, err = tls.Dial("tcp", parsed.Host, &tls.Config{InsecureSkipVerify: true})
	} else {
		panic("Non-TLS hijacking dialer not supported.")
	}
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return nil, err
	}
	err = req.Write(conn)
	if err != nil {
		return nil, err
	}

	/*
		// Wait for HTTP response???
		bufr := bufio.NewReader(conn)
		resp, err := http.ReadResponse(bufr, req)
		if err != nil {
			return nil, err
		}
	*/

	hc, _ := newHijackedConn(conn, nil, nil)
	return hc, nil
}
