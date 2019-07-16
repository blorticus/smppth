package main

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

	broker := newInteractionBroker().setInputPromptStream(promptWriter).setInputReader(inputReader).setOutputWriter(outputWriter)
	msgSendChannel := broker.retrieveSendMessageChannel()

	go broker.beginInteractiveSession()

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

	if string(nextSendMessageDescritor.pdu.MandatoryParameters[17].Value.(string)) != "This is a test" {
		t.Errorf("[2] For received PDU, expected short_message = (This is a test), got = (%s)", string(nextSendMessageDescritor.pdu.MandatoryParameters[17].Value.(string)))
	}
}

func expectValueOnMockWriter(writer *mockWriter, expectedData []byte) error {
	timedOutWaitingForWrite := false
	var writeDataReceived []byte

	fmt.Println("Entering select")

	select {
	case writeDataReceived = <-writer.channelOfWrittenData:
		break
	case <-time.After(time.Second * 3):
		timedOutWaitingForWrite = true
	}

	fmt.Printf("Exited Select: %t\b", timedOutWaitingForWrite)

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

func expectValuesForSendMessageDescriptor(messageDescriptor *messageDescriptor, expectedSendFromEsmeNamed string, expectedSendToSmscNamed string, expectedPduCommandID smpp.CommandIDType) error {
	if messageDescriptor == nil {
		return fmt.Errorf("Received messageDescriptor is nil")
	}

	if messageDescriptor.sendFromEsmeNamed != expectedSendFromEsmeNamed {
		return fmt.Errorf("Expected sendFromEsmeNamed = (%s), got = (%s)", expectedSendFromEsmeNamed, messageDescriptor.sendFromEsmeNamed)
	}

	if messageDescriptor.sendToSmscNamed != expectedSendToSmscNamed {
		return fmt.Errorf("Expected sendToSmscNamed = (%s), got = (%s)", expectedSendToSmscNamed, messageDescriptor.sendToSmscNamed)
	}

	if messageDescriptor.pdu == nil {
		return fmt.Errorf("messageDescriptor contains nil PDU value")
	}

	if messageDescriptor.pdu.CommandID != expectedPduCommandID {
		return fmt.Errorf("Expected PDU commandID = (%s), got (%s)", smpp.CommandName(expectedPduCommandID), messageDescriptor.pdu.CommandName())
	}

	return nil
}
