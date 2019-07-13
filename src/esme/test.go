package main

import (
	"net"
	"time"
)

type fakeNetConn struct {
	nextReadValue []byte
	nextReadError error

	lastWriteValue []byte
	nextWriteError error

	bindTrasceiverResp01Msg []byte
	enquireLink01Msg        []byte
}

func newFakeNetConn() *fakeNetConn {
	conn := &fakeNetConn{
		bindTrasceiverResp01Msg: []byte{
			0, 0, 0, 0x14, // len = 20
			0x80, 0, 0, 0x02, // command = bind_trasceiver_resp
			0, 0, 0, 0x00, // status code = 0
			0, 0, 0, 0x01, // seq number = 1
			0x66, 0x6f, 0x6f, 0, // systemID = 'foo'
		},

		enquireLink01Msg: []byte{
			0, 0, 0, 0x10, // len = 16
			0, 0, 0, 0x15, // command = eqnuire_link
			0, 0, 0, 0x00, // status code = 0
			0, 0, 0, 0x02, // seq number = 2
		},
	}

	return conn
}

func (conn *fakeNetConn) Read(b []byte) (int, error) {
	if conn.nextReadError != nil {
		return 0, conn.nextReadError
	}

	copy(b, conn.nextReadValue)

	return len(conn.nextReadValue), nil
}

func (conn *fakeNetConn) Write(b []byte) (n int, err error) {
	if conn.nextWriteError != nil {
		return 0, nil
	}

	conn.lastWriteValue = b
	return len(b), nil
}

func (conn *fakeNetConn) Close() error {
	return nil
}

func (conn *fakeNetConn) LocalAddr() net.Addr {
	return nil
}

func (conn *fakeNetConn) RemoteAddr() net.Addr {
	return nil
}

func (conn *fakeNetConn) SetDeadline(t time.Time) error {
	return nil
}

func (conn *fakeNetConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (conn *fakeNetConn) SetWriteDeadline(t time.Time) error {
	return nil
}
