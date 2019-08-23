package smppth

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"testing"

	smpp "github.com/blorticus/smpp-go"
)

func TestEsmePeerMessageListener(t *testing.T) {
	esme := NewEsme("test-esme", nil, 0)

	conn := newFakeNetConn()

	connector := newEsmePeerMessageListener("testSmsc01", esme, conn)
	connector.streamReader = smpp.NewNetworkStreamReader(conn)

	conn.nextReadValue = testSmppMsgTransceiverResp01()
	connector.completeTransceiverBindingTowardPeer("esme01", "system", "password")

	pdu, err := smpp.DecodePDU(conn.lastWriteValue)

	if err != nil {
		t.Errorf("completeTransceiverBindingTowardPeer() should have returned tranceiver_bind_resp, but Decode() on conn Write() generated error = (%s)", err)
	}

	if pdu.CommandID != 0x00000009 {
		t.Errorf("completeTransceiverBindingTowardPeer() should have Write()n bind-tranceiver, but message type = (%s)", pdu.CommandName())
	}

	eventMsgChannel := make(chan *AgentEvent, 10)

	conn.nextReadValue = testSmppMsgEnquireLink01()
	connector.parentESME.SetAgentEventChannel(eventMsgChannel)
	go connector.startListeningForIncomingMessagesFromPeer()

	eventMessage := <-eventMsgChannel

	validationError := validateEventMessage(eventMessage, ReceivedPDU, "testSmsc01")

	if validationError != nil {
		t.Errorf("On first enquire_link from peer, for received event message, %s", validationError)
	}

	if eventMessage.SmppPDU == nil {
		t.Errorf("On first enquire_link from peer, for received event message, SmppPDU should not be nil, but is")
	}

	if eventMessage.SmppPDU.CommandID != smpp.CommandEnquireLink {
		t.Errorf("On first enquire_link from peer, for received event message, SmppPDU CommandID should be enquire_link, but is (%s)", eventMessage.SmppPDU.CommandName())
	}
}

func TestEsmeOneSmscEndpoint(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	defer listener.Close()

	if err != nil {
		panic(fmt.Sprintf("Failed to create local listener for SMSC: %s", err))
	}

	portAsUint64, _ := strconv.ParseUint(strings.Split(listener.Addr().String(), ":")[1], 10, 16)
	smscListeningPort := uint16(portAsUint64)

	go smscSimulatedListener(listener)

	esme := NewEsme("testEsme01", net.ParseIP("127.0.0.1"), 0)
	esme.peerBinds = []smppBindInfo{
		smppBindInfo{
			remoteIP:   net.ParseIP("127.0.0.1"),
			remotePort: smscListeningPort,
			smscName:   "testSmsc01",
			password:   "password",
			systemID:   "esme01",
			systemType: "generic",
		},
	}

	esmeEventChannel := make(chan *AgentEvent, 10)
	esme.SetAgentEventChannel(esmeEventChannel)

	go esme.StartEventLoop()

	if _, err := eventChannelTypeCheck(esmeEventChannel, SentPDU); err != nil {
		t.Errorf("On StartEventLoop, first message, %s", err)
	}

	if _, err := eventChannelTypeCheck(esmeEventChannel, ReceivedPDU); err != nil {
		t.Errorf("On StartEventLoop, second message, %s", err)
	}

	if _, err := eventChannelTypeCheck(esmeEventChannel, CompletedBind); err != nil {
		t.Errorf("On StartEventLoop, third message, %s", err)
	}

	esme.SendMessageToPeer(&MessageDescriptor{
		NameOfSendingPeer:   "testEsme01",
		NameOfReceivingPeer: "testSmsc01",
		PDU:                 testSmppPDUEnquireLink01(),
	})

	nextEvent, err := eventChannelTypeCheck(esmeEventChannel, SentPDU)
	if err != nil {
		t.Errorf("After EnquireLink SendMessageToPeer, first AgentEvent, %s", err)
	} else {
		err = eventCheck(nextEvent, "testSmsc01", smpp.CommandEnquireLink)
		if err != nil {
			t.Errorf("After EnquireLink SendMessageToPeer, first AgentEvent, %s", err)
		}
	}

	nextEvent, err = eventChannelTypeCheck(esmeEventChannel, ReceivedPDU)
	if err != nil {
		t.Errorf("After EnquireLink SendMessageToPeer, second AgentEvent, %s", err)
	} else {
		err = eventCheck(nextEvent, "testSmsc01", smpp.CommandEnquireLinkResp)
		if err != nil {
			t.Errorf("After EnquireLink SendMessageToPeer, second AgentEvent, %s", err)
		}
	}

	esme.SendMessageToPeer(&MessageDescriptor{
		NameOfSendingPeer:   "testEsme01",
		NameOfReceivingPeer: "testSmsc01",
		PDU:                 testSmppPDUSubmitSm01(),
	})

	nextEvent, err = eventChannelTypeCheck(esmeEventChannel, SentPDU)
	if err != nil {
		t.Errorf("After SubmitSm SendMessageToPeer, first AgentEvent, %s", err)
	} else {
		err = eventCheck(nextEvent, "testSmsc01", smpp.CommandSubmitSm)
		if err != nil {
			t.Errorf("After SubmitSm SendMessageToPeer, first AgentEvent, %s", err)
		}
	}

	nextEvent, err = eventChannelTypeCheck(esmeEventChannel, ReceivedPDU)
	if err != nil {
		t.Errorf("After SubmitSm SendMessageToPeer, second AgentEvent, %s", err)
	} else {
		err = eventCheck(nextEvent, "testSmsc01", smpp.CommandSubmitSmResp)
		if err != nil {
			t.Errorf("After SubmitSm SendMessageToPeer, second AgentEvent, %s", err)
		}
	}
}

func eventCheck(event *AgentEvent, expectedRemotePeerName string, expectedSmppCommand smpp.CommandIDType) error {
	if event.RemotePeerName != expectedRemotePeerName {
		return fmt.Errorf("expected RemotePeerNAme = (%s), got = (%s)", expectedRemotePeerName, event.RemotePeerName)
	}

	if event.SmppPDU == nil {
		return fmt.Errorf("expected SmppPDU, got nil")
	}

	if event.SmppPDU.CommandID != expectedSmppCommand {
		return fmt.Errorf("expected event.Type = %s (%d), got = %s (%d)",
			smpp.CommandName(event.SmppPDU.CommandID),
			event.SmppPDU.CommandID,
			event.SmppPDU.CommandName(),
			event.SmppPDU.CommandID)
	}

	return nil
}

func smscSimulatedListener(listener net.Listener) {
	conn, err := listener.Accept()
	defer conn.Close()

	if err != nil {
		panic(fmt.Sprintf("Failed on simulated SMSC listener Accept(): %s", err))
	}

	lastReceivedPDU, err := simulatedSmscReceivePDUWithExpectations(conn, smpp.CommandBindTransceiver)

	if err != nil {
		panic(fmt.Sprintf("On wait for bind-transceiver from esme: %s", err))
	}

	bindRespPDU := smpp.NewPDU(smpp.CommandBindTransceiverResp, 0, lastReceivedPDU.SequenceNumber, []*smpp.Parameter{
		smpp.NewCOctetStringParameter("smsc01"),
	}, []*smpp.Parameter{})

	encodedPDU, _ := bindRespPDU.Encode()
	_, err = conn.Write(encodedPDU)

	if err != nil {
		panic(fmt.Sprintf("Failed on SMSC Write() of transceiver_bind_resp: %s", err))
	}

	lastReceivedPDU, err = simulatedSmscReceivePDUWithExpectations(conn, smpp.CommandEnquireLink)

	if err != nil {
		panic(fmt.Sprintf("On wait for first enquire-link: %s", err))
	}

	enquireLinkRespPDU := smpp.NewPDU(smpp.CommandEnquireLinkResp, 0, lastReceivedPDU.SequenceNumber, []*smpp.Parameter{}, []*smpp.Parameter{})

	encodedPDU, _ = enquireLinkRespPDU.Encode()
	_, err = conn.Write(encodedPDU)

	if err != nil {
		panic(fmt.Sprintf("Failed on SMSC Write() of enquire-link-resp: %s", err))
	}

	lastReceivedPDU, err = simulatedSmscReceivePDUWithExpectations(conn, smpp.CommandSubmitSm)

	if err != nil {
		panic(fmt.Sprintf("On wait for first submit-sm: %s", err))
	}

	SubmitSmRespPDU := smpp.NewPDU(smpp.CommandSubmitSmResp, 0, lastReceivedPDU.SequenceNumber, []*smpp.Parameter{
		smpp.NewCOctetStringParameter("resp"),
	}, []*smpp.Parameter{})

	encodedPDU, _ = SubmitSmRespPDU.Encode()
	_, err = conn.Write(encodedPDU)

	if err != nil {
		panic(fmt.Sprintf("Failed on SMSC Write() of submit-sm-resp: %s", err))
	}
}

func simulatedSmscReceivePDUWithExpectations(conn net.Conn, expectedCommandID smpp.CommandIDType) (*smpp.PDU, error) {
	readBuf := make([]byte, 65536)

	bytesRead, err := conn.Read(readBuf)

	if err != nil {
		return nil, fmt.Errorf("Failed on simulated SMSC listener Read(): %s", err)
	}

	readBuf = readBuf[:bytesRead]

	pdu, err := smpp.DecodePDU(readBuf)

	if err != nil {
		return nil, fmt.Errorf("Failed to decode initial PDU from peer: %s", err)
	}

	if pdu.CommandID != expectedCommandID {
		return pdu, fmt.Errorf("Received PDU from ESME.  Expect = (%s), got = (%s)", smpp.CommandName(expectedCommandID), pdu.CommandName())
	}

	return pdu, nil
}
