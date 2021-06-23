package dummylistener

import (
	"bytes"
	"fmt"
	"net"
	"time"
)

type DummyListener struct {
	parent net.Listener
}

type DummyConn struct {
	parent   net.Conn
	startGet bool
}

func Listener(ln net.Listener) (net.Listener, error) {
	return &DummyListener{parent: ln}, nil
}

// Accept waits for and returns the next connection to the listener.
func (ln *DummyListener) Accept() (net.Conn, error) {
	cn, err := ln.parent.Accept()

	if err != nil {
		return nil, err
	} else {
		return &DummyConn{parent: cn, startGet: false}, nil
	}
}

// Close closes the listener.
// Any blocked Accept operations will be unblocked and return errors.
func (ln *DummyListener) Close() error {
	return ln.parent.Close()
}

// Addr returns the listener's network address.
func (ln *DummyListener) Addr() net.Addr {
	return ln.parent.Addr()
}

/////////

func (cn *DummyConn) Read(b []byte) (int, error) {
	n, err := cn.parent.Read(b)

	if err == nil {
		fmt.Printf("DEBUG Read hook. n=%d\n", n)
		readIdx := 0
		if cn.startGet == false {
			// i don't wanna think if the "GET" is separated...
			data := b[:n]
			idx := bytes.Index(data, []byte("GET"))
			if idx >= 0 {
				cn.startGet = true
				readIdx = idx
				fmt.Printf("DEBUG GET %s\n", b[idx:idx+10])
			} else {
				fmt.Printf("DEBUG GET not found\n")
			}
		}
		if cn.startGet == true {
			//Search \r\n
			data := b[readIdx:n]
			idx := bytes.Index(data, []byte("\r\n"))
			if idx >= 0 {
				cn.startGet = false

				idx = idx + readIdx

				//Request line is ended.
				fmt.Printf("DEBUG End found %d %d %s\n", readIdx, idx, b[readIdx:idx+2])
				if bytes.Compare(b[idx+2:idx+8], []byte("Host: ")) == 0 {
					fmt.Printf("DEBUG Host is avail. no inject.\n")
				} else {
					insertBytes := []byte("Host: dummy\r\n")
					insertLen := len(insertBytes)
					copy(b[idx+2+insertLen:n+insertLen], b[idx+2:n])
					copy(b[idx+2:idx+2+insertLen], insertBytes)

					fmt.Printf("DEBUG Inject %s\n", b[readIdx:idx+2+insertLen+10])

					n = n + insertLen
				}
			}
		}
	} else {
		fmt.Printf("DEBUG Read hook error  %s\n", err.Error())
	}
	return n, err
}

func (cn *DummyConn) Write(b []byte) (int, error) {
	return cn.parent.Write(b)
}

func (cn *DummyConn) Close() error {
	return cn.parent.Close()
}

func (cn *DummyConn) LocalAddr() net.Addr {
	return cn.parent.LocalAddr()
}
func (cn *DummyConn) RemoteAddr() net.Addr {
	return cn.parent.RemoteAddr()
}
func (cn *DummyConn) SetDeadline(t time.Time) error {
	return cn.parent.SetDeadline(t)
}

func (cn *DummyConn) SetReadDeadline(t time.Time) error {
	return cn.parent.SetReadDeadline(t)
}
func (cn *DummyConn) SetWriteDeadline(t time.Time) error {
	return cn.parent.SetWriteDeadline(t)
}
