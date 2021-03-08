package smppth

import (
	"fmt"
	"net"
	"time"

	"github.com/blorticus/smpp"
)

func testSmppMsgBindTransceiver01() []byte {
	return []byte{
		0, 0, 0, 0x14, // len = 20
		0x00, 0, 0, 0x09, // command = bind_transceiver_resp
		0, 0, 0, 0x00, // status code = 0
		0, 0, 0, 0x01, // seq number = 1
		0x66, 0x6f, 0x6f, 0, // systemID = 'foo'
		0x62, 0x61, 0x72, 0, // password = 'bar'
		0x66, 0x6f, 0x6f, 0, // systemType = 'boo'
		0x34, // interface_version
	}
}

func testSmppMsgTransceiverResp01() []byte {
	return []byte{
		0, 0, 0, 0x14, // len = 20
		0x80, 0, 0, 0x09, // command = bind_trasceiver_resp
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
		0, 0, 0, 0x15, // command = enquire_link
		0, 0, 0, 0x00, // status code = 0
		0, 0, 0, 0x02, // seq number = 2
	}
}

func testSmppPDUEnquireLink01() *smpp.PDU {
	pdu, _ := smpp.DecodePDU(testSmppMsgEnquireLink01())
	return pdu
}

func testSmppMsgSubmitSm01() []byte {
	return []byte{
		0, 0, 0, 0x25, // len = 16
		0, 0, 0, 0x04, // command = submit_sm
		0, 0, 0, 0x00, // status code = 0
		0, 0, 0, 0x03, // seq number = 2
		0x0,                    // service_type
		0x0,                    // source_addr_ton
		0x0,                    // source_addr_npi,
		0x0,                    // source_addr
		0x0,                    // dest_addr_ton
		0x0,                    // dest_addr_npi,
		0x0,                    // destination_addr
		0x0,                    // esm_class
		0x0,                    // protocol_id
		0x0,                    // priority_flag
		0x0,                    // schedule_delivery_time
		0x0,                    // validity_period
		0x0,                    // registered_delivery
		0x0,                    // replace_if_present_flag
		0x0,                    // data_coding
		0x0,                    // sm_default_msg_id
		0x04,                   // sm_length
		0x54, 0x45, 0x53, 0x54, // short_message ("TEST")
	}
}

func testSmppPDUSubmitSm01() *smpp.PDU {
	pdu, _ := smpp.DecodePDU(testSmppMsgSubmitSm01())
	return pdu
}

type fakeNetConn struct {
	nextReadValue []byte
	nextReadError error

	lastWriteValue []byte
	nextWriteError error
}

func eventTypeToString(eventType AgentEventType) string {
	switch eventType {
	case SentPDU:
		return "SentPDU"
	case ReceivedPDU:
		return "ReceivedPDU"
	case CompletedBind:
		return "CompletedBind"
	default:
		return "<unknown>"
	}

}

func eventChannelTypeCheck(eventChannel <-chan *AgentEvent, expectingEventType AgentEventType) (*AgentEvent, error) {
	select {
	case nextEvent := <-eventChannel:
		if nextEvent.Type != expectingEventType {
			return nextEvent, fmt.Errorf("On received event, expected %s (%d), got %s (%d)",
				eventTypeToString(expectingEventType),
				int(expectingEventType),
				eventTypeToString(nextEvent.Type),
				int(nextEvent.Type),
			)
		}

		return nextEvent, nil
	case <-time.After(time.Second * 2):
		return nil, fmt.Errorf("Timed out waiting for first event from esmeEventChannel")
	}
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

func validateEventMessage(eventMessage *AgentEvent, expectedType AgentEventType, expectedSenderName string) error {
	if eventMessage == nil {
		return fmt.Errorf("expected valid event message, got nil")
	}

	if eventMessage.Type != expectedType {
		return fmt.Errorf("expected Type = %d, got = %d", int(expectedType), int(eventMessage.Type))
	}

	if eventMessage.RemotePeerName != expectedSenderName {
		return fmt.Errorf("expected nameOfSender = (%s), got = (%s)", expectedSenderName, eventMessage.RemotePeerName)
	}

	return nil
}
