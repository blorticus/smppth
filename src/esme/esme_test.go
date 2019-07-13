package main

import (
	"fmt"
	"smpp"
	"testing"
)

func TestEsmePeerMessageListener(t *testing.T) {
	esme := &esme{}

	conn := newFakeNetConn()

	connector := newEsmePeerMessageListener("testSmsc01", esme, conn)
	connector.streamReader = smpp.NewNetworkStreamReader(conn)

	conn.nextReadValue = conn.bindTrasceiverResp01Msg
	connector.completeTransceiverBindingTowardPeer("esme01", "system", "password")

	pdu, err := smpp.DecodePDU(conn.lastWriteValue)

	if err != nil {
		t.Errorf("completeTransceiverBindingTowardPeer() should have returned tranceiver_bind_resp, but Decode() on conn Write() generated error = (%s)", err)
	}

	if pdu.CommandID != 0x00000009 {
		t.Errorf("completeTransceiverBindingTowardPeer() should have Write()n bind-tranceiver, but message type = (%s)", pdu.CommandName())
	}

	eventMsgChannel := make(chan *esmeListenerEvent)

	conn.nextReadValue = conn.enquireLink01Msg
	go connector.startListeningForIncomingMessagesFromPeer(eventMsgChannel)

	eventMessage := <-eventMsgChannel

	validationError := validateEventMessage(eventMessage, receivedMessage, "testSmsc01")

	if validationError != nil {
		t.Errorf("On first enquire_link from peer, for received event message, %s", validationError)
	}

	if eventMessage.smppPDU == nil {
		t.Errorf("On first enquire_link from peer, for received event message, smppPDU should not be nil, but is")
	}

	if eventMessage.smppPDU.CommandID != smpp.CommandEnquireLink {
		t.Errorf("On first enquire_link from peer, for received event message, smppPDU CommandID should be enquire_link, but is (%s)", eventMessage.smppPDU.CommandName())
	}
}

func validateEventMessage(eventMessage *esmeListenerEvent, expectedType esmeEventType, expectedSenderName string) error {
	if eventMessage == nil {
		return fmt.Errorf("expected valid event message, got nil")
	}

	if eventMessage.Type != receivedMessage {
		return fmt.Errorf("expected Type = %d, got = %d", int(expectedType), int(eventMessage.Type))
	}

	if eventMessage.nameOfMessageSender != expectedSenderName {
		return fmt.Errorf("expected nameOfSender = (%s), got = (%s)", expectedSenderName, eventMessage.nameOfMessageSender)
	}

	return nil
}
