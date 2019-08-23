package smppth

import (
	"net"
	"testing"

	smpp "github.com/blorticus/smpp-go"
)

func TestSmscPeerMessageHandler(t *testing.T) {
	parentSMSC := NewSMSC("testSmsc", "testSmsc", net.ParseIP("127.0.0.1"), 2772)
	mockRemotePeerConnection := newFakeNetConn()

	handler := newSmscPeerMessageHandler(parentSMSC, mockRemotePeerConnection)

	eventMsgChannel := make(chan *AgentEvent, 10)
	parentSMSC.SetAgentEventChannel(eventMsgChannel)

	mockRemotePeerConnection.nextReadValue = testSmppMsgBindTransceiver01()

	go handler.startHandlingPeerConnection()

	if _, err := eventChannelTypeCheck(eventMsgChannel, ReceivedPDU); err != nil {
		t.Error(err)
	}

	if _, err := eventChannelTypeCheck(eventMsgChannel, SentPDU); err != nil {
		t.Error(err)
	}

	nextEvent, err := eventChannelTypeCheck(eventMsgChannel, CompletedBind)
	if err != nil {
		t.Error(err)
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

	err = parentSMSC.SendMessageToPeer(&MessageDescriptor{NameOfSendingPeer: "testSmsc", NameOfReceivingPeer: "foo", PDU: testSmppPDUEnquireLink01()})

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
