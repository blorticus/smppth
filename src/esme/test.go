package main

import (
	"net"
	"smpp"
	"time"
)

func testSmppMsgTransceiverResp01() []byte {
	return []byte{
		0, 0, 0, 0x14, // len = 20
		0x80, 0, 0, 0x02, // command = bind_trasceiver_resp
		0, 0, 0, 0x00, // status code = 0
		0, 0, 0, 0x01, // seq number = 1
		0x66, 0x6f, 0x6f, 0, // systemID = 'foo'
	}
}

func testSmppPDUTransceiverResp01() *smpp.PDU {
	pdu, _ := smpp.DecodePDU(testSmppMsgTransceiverResp01())
	return pdu
}

func testSmppMsgEnquireLink01() []byte {
	return []byte{
		0, 0, 0, 0x10, // len = 16
		0, 0, 0, 0x15, // command = eqnuire_link
		0, 0, 0, 0x00, // status code = 0
		0, 0, 0, 0x02, // seq number = 2
	}
}

func testSmppPDUEnquireLink01() *smpp.PDU {
	pdu, _ := smpp.DecodePDU(testSmppMsgEnquireLink01())
	return pdu
}

type fakeNetConn struct {
	nextReadValue []byte
	nextReadError error

	lastWriteValue []byte
	nextWriteError error
}

func newFakeNetConn() *fakeNetConn {
	conn := &fakeNetConn{}

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

type mockWriterOnWriteCallbackFunc func(bytesWritten []byte, writeLength int, err error)

type mockWriter struct {
	writesSinceLastClear [][]byte
	onWriteCallback      mockWriterOnWriteCallbackFunc
}

func newMockWriter() *mockWriter {
	return &mockWriter{writesSinceLastClear: [][]byte{}, onWriteCallback: nil}
}

func (writer *mockWriter) Write(bytesToWrite []byte) (int, error) {
	writer.writesSinceLastClear = append(writer.writesSinceLastClear, bytesToWrite)

	if writer.onWriteCallback != nil {
		writer.onWriteCallback(bytesToWrite, len(bytesToWrite), nil)
	}

	return len(bytesToWrite), nil
}

func (writer *mockWriter) getLastWrittenValues() [][]byte {
	return writer.writesSinceLastClear
}

func (writer *mockWriter) clearStoresWrites() {
	writer.writesSinceLastClear = [][]byte{}
}

func (writer *mockWriter) setOnWriteCallback(callback mockWriterOnWriteCallbackFunc) {
	writer.onWriteCallback = callback
}

func (writer *mockWriter) clearOnWriteCallback() {
	writer.onWriteCallback = nil
}

type mockReader struct {
	nextReadValue []byte
}

func newMockReader() *mockReader {
	return &mockReader{nextReadValue: []byte{}}
}

func (reader *mockReader) setNextReadValue(nextReadValue []byte) {
	reader.nextReadValue = nextReadValue[:]
}

func (reader *mockReader) Read(readBuffer []byte) (int, error) {
	readLength := 0
	if len(readBuffer) < len(reader.nextReadValue) {
		copy(readBuffer, reader.nextReadValue[:len(readBuffer)])
		reader.nextReadValue = reader.nextReadValue[len(readBuffer):]
		readLength = len(readBuffer)
	} else {
		copy(readBuffer, reader.nextReadValue)
		readLength = len(reader.nextReadValue)
	}

	reader.nextReadValue = []byte{}

	return readLength, nil
}
