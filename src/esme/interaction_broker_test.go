package main

import (
	"testing"
)

func TestBasicInteraction(t *testing.T) {
	promptWriter := newMockWriter()
	outputWriter := newMockWriter()
	inputReader := newMockReader()

	broker := newInteractionBroker().setInputPromptStream(promptWriter).setInputReader(inputReader).setOutputWriter(outputWriter)
	msgSendChannel := broker.retrieveSendMessageChannel()

	inputReader.setNextReadValue([]byte("esme01: send enquire-link to smsc01\n"))

	wasPrompted := false
	promptWriter.setOnWriteCallback(func(writtenBytes []byte, writeLength int, err error) {
		wasPrompted = true
		if string(writtenBytes) != "> " {
			t.Errorf("Expected promptWriter to receive prompt (> ), got (%s)", string(writtenBytes))
		}
	})

	go broker.beginInteractiveSession()

	nextSendMessage := <-msgSendChannel

	if nextSendMessage == nil {
		t.Errorf("Expected enquire-link message emitted by broker, got nil")
	}

	if !wasPrompted {
		t.Errorf("Expected to be prompted, but was not")
	}
}
