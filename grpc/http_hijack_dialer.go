package grpc

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"
)

type Dialer func(string, time.Duration) (net.Conn, error)

func NewDialerInsecure(urlStr string) (Dialer, error) {
	return NewDialer(urlStr, &tls.Config{InsecureSkipVerify: true})
}

func NewDialer(urlStr string, conf *tls.Config) (Dialer, error) {
	_, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	return func(addr string, timeout time.Duration) (net.Conn, error) {
		return dialAndHijack(urlStr, timeout, conf)
	}, nil
}

func dialAndHijack(urlStr string, timeout time.Duration, conf *tls.Config) (net.Conn, error) {
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	var conn net.Conn
	if parsed.Scheme == "https" {
		conn, err = tls.Dial("tcp", parsed.Host, conf)
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

	bufr := bufio.NewReader(conn)
	resp, err := http.ReadResponse(bufr, req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Unexpected HTTP status in hijack response: %d", resp.StatusCode)
	}

	hc, _ := newHijackedConn(conn, bufr, nil)
	return hc, nil
}
