package smppth

import (
	"fmt"
	"smpp"
	"testing"
	"time"
)

func TestBasicInteraction(t *testing.T) {
	promptWriter := newMockWriter("prompt")
	outputWriter := newMockWriter("output")
	inputReader := newMockReader("input")

	broker := NewTextInteractionBroker().SetInputPromptStream(promptWriter).SetInputReader(inputReader).SetOutputWriter(outputWriter)
	msgSendChannel := broker.RetrieveSendMessageChannel()

	go broker.BeginInteractiveSession()

	err := expectValueOnMockWriter(promptWriter, []byte("> "))

	if err != nil {
		t.Errorf("[1] Failed to receive prompt value: %s", err)
	}

	inputReader.setNextReadValue([]byte("esme01: send enquire-link to smsc01\n"))

	nextSendMessageDescritor := <-msgSendChannel

	err = expectValuesForSendMessageDescriptor(nextSendMessageDescritor, "esme01", "smsc01", smpp.CommandEnquireLink)

	if err != nil {
		t.Errorf("[1] For received messageDescriptor: %s", err)
	}

	err = expectValueOnMockWriter(promptWriter, []byte("> "))

	if err != nil {
		t.Errorf("[2] Failed to receive prompt value: %s", err)
	}

	inputReader.setNextReadValue([]byte("esme-1-1: send submit-sm to cluster-smsc-01 short_message=\"This is a test\"\n"))

	nextSendMessageDescritor = <-msgSendChannel

	err = expectValuesForSendMessageDescriptor(nextSendMessageDescritor, "esme-1-1", "cluster-smsc-01", smpp.CommandSubmitSm)

	if err != nil {
		t.Errorf("[2] For received messageDescriptor: %s", err)
	}

	if string(nextSendMessageDescritor.PDU.MandatoryParameters[17].Value.([]byte)) != "This is a test" {
		t.Errorf("[2] For received PDU, expected short_message = (This is a test), got = (%s)", string(nextSendMessageDescritor.PDU.MandatoryParameters[17].Value.([]byte)))
	}
}

func expectValueOnMockWriter(writer *mockWriter, expectedData []byte) error {
	timedOutWaitingForWrite := false
	var writeDataReceived []byte

	select {
	case writeDataReceived = <-writer.channelOfWrittenData:
		break
	case <-time.After(time.Second * 3):
		timedOutWaitingForWrite = true
	}

	if timedOutWaitingForWrite {
		return fmt.Errorf("timed out waiting for write input")
	}

	if len(writeDataReceived) != len(expectedData) {
		return fmt.Errorf("expected (%d) bytes of written data, received (%d)", len(expectedData), len(writeDataReceived))
	}

	for i, v := range expectedData {
		if writeDataReceived[i] != v {
			return fmt.Errorf("expected written data begins differing from received data at byte (%d)", i)
		}
	}

	return nil
}

func expectValuesForSendMessageDescriptor(messageDescriptor *MessageDescriptor, expectedNameOfSourcePeer string, expectedNameOfRemotePeer string, expectedPduCommandID smpp.CommandIDType) error {
	if messageDescriptor == nil {
		return fmt.Errorf("Received messageDescriptor is nil")
	}

	if messageDescriptor.NameOfSourcePeer != expectedNameOfSourcePeer {
		return fmt.Errorf("Expected NameOfSourcePeer = (%s), got = (%s)", expectedNameOfSourcePeer, messageDescriptor.NameOfSourcePeer)
	}

	if messageDescriptor.NameOfRemotePeer != expectedNameOfRemotePeer {
		return fmt.Errorf("Expected SendToSmscNamed = (%s), got = (%s)", expectedNameOfRemotePeer, messageDescriptor.NameOfRemotePeer)
	}

	if messageDescriptor.PDU == nil {
		return fmt.Errorf("messageDescriptor contains nil PDU value")
	}

	if messageDescriptor.PDU.CommandID != expectedPduCommandID {
		return fmt.Errorf("Expected PDU commandID = (%s), got (%s)", smpp.CommandName(expectedPduCommandID), messageDescriptor.PDU.CommandName())
	}

	return nil
}
