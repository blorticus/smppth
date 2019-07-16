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
	channelOfWrittenData chan []byte
	dontBlockOnNextWrite bool
	name                 string
}

func newMockWriter(writerName string) *mockWriter {
	return &mockWriter{channelOfWrittenData: make(chan []byte), dontBlockOnNextWrite: false, name: writerName}
}

func (writer *mockWriter) Write(bytesToWrite []byte) (int, error) {
	writer.channelOfWrittenData <- bytesToWrite
	return len(bytesToWrite), nil
}

func (writer *mockWriter) ignoreNextWrite() {
	<-writer.channelOfWrittenData
}

type mockReader struct {
	channelOfDataToRead      chan []byte
	leftOverDataFromLastRead []byte
	name                     string
}

func newMockReader(readerName string) *mockReader {
	return &mockReader{channelOfDataToRead: make(chan []byte), leftOverDataFromLastRead: []byte{}, name: readerName}
}

func (reader *mockReader) setNextReadValue(nextReadValue []byte) {
	reader.channelOfDataToRead <- nextReadValue
}

func (reader *mockReader) Read(readBuffer []byte) (int, error) {
	nextReadValue := reader.leftOverDataFromLastRead

	if len(nextReadValue) == 0 {
		nextReadValue = <-reader.channelOfDataToRead
	}

	readLength := 0
	if len(readBuffer) < len(nextReadValue) {
		copy(readBuffer, nextReadValue[:len(readBuffer)])
		reader.leftOverDataFromLastRead = nextReadValue[len(readBuffer):]
		readLength = len(readBuffer)
	} else {
		copy(readBuffer, nextReadValue)
		readLength = len(nextReadValue)
		reader.leftOverDataFromLastRead = []byte{}
	}

	return readLength, nil
}
