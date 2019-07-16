package smppth

import (
	"net"
	"smpp"
	"testing"
)

func TestSmscPeerMessageHandler(t *testing.T) {
	parentSMSC := NewSMSC("testSmsc", net.ParseIP("127.0.0.1"), 2772)
	mockRemotePeerConnection := newFakeNetConn()

	handler := newSmscPeerMessageHandler(parentSMSC, mockRemotePeerConnection)

	eventMsgChannel := make(chan *AgentEvent)

	mockRemotePeerConnection.nextReadValue = testSmppMsgBindTransceiver01()

	go handler.startHandlingPeerConnection(eventMsgChannel)

	nextEvent := <-eventMsgChannel
	err := validateEventMessage(nextEvent, ReceivedBind, "foo")

	if err != nil {
		t.Errorf("On expected ReceivedBind event from 'foo': %s", err)
	}

	nextEvent = <-eventMsgChannel
	err = validateEventMessage(nextEvent, CompletedBind, "foo")

	if err != nil {
		t.Errorf("On expected CompletedBind from 'foo': %s", err)
	}

	if nextEvent.SmppPDU.CommandID != smpp.CommandBindTransceiverResp {
		t.Errorf("On completion of bind, expected bind-transceiver-resp, got %s", nextEvent.SmppPDU.CommandName())
	}

	bindRespPDU, err := smpp.DecodePDU(mockRemotePeerConnection.lastWriteValue)

	if err != nil {
		t.Errorf("Failed to decode transciever-bind-resp written by handler: %s", err)
	} else {
		if bindRespPDU.CommandID != smpp.CommandBindTransceiverResp {
			t.Errorf("handler should have sent bind-transceiver-resp on socket, but sent %s", bindRespPDU.CommandName())
		}
	}

	err = parentSMSC.SendMessageToPeer(&MessageDescriptor{NameOfSourcePeer: "testSmsc", NameOfRemotePeer: "foo", PDU: testSmppPDUEnquireLink01()})

	if err != nil {
		t.Errorf("Received error on sending enquire-link to peer 'foo': %s", err)
	} else {
		enquireLinkPDU, err := smpp.DecodePDU(mockRemotePeerConnection.lastWriteValue)

		if err != nil {
			t.Errorf("Received error trying to deocde written enquire-link PDU: %s", err)
		} else {
			if enquireLinkPDU.CommandID != smpp.CommandEnquireLink {
				t.Errorf("Decoded enquire-link received from SMSC agent expected enquire-link, got %s", enquireLinkPDU.CommandName())
			}
		}
	}
}
