package main

import (
	"net"
	"smpp"
)

type esmeEventType int

const (
	receivedMessage esmeEventType = iota
)

type smppBindInfo struct {
	remoteIP   net.IP
	remotePort uint16
	systemID   string
	password   string
	systemType string
}

type messageDescriptor struct {
}

type esmeListenerEvent struct {
	Type     esmeEventType
	smppPDU  *smpp.PDU
	peerIP   net.IP
	peerPort uint16
}

type esme struct {
	bindIP   net.IP
	bindPort uint16
}

func (esme *esme) performTranceiverBindTo(bindInfo *smppBindInfo) error {
	return nil
}

func (esme *esme) startListening() {
}

func (esme *esme) sendMessageToPeer(message *messageDescriptor) {

}

func (esme *esme) incomingEventChannel() <-chan *esmeListenerEvent {
	return nil
}

type esmeDefinitionYamlReader struct {
}

func newEsmeDefinitionYamlReader() *esmeDefinitionYamlReader {
	return &esmeDefinitionYamlReader{}
}

func (reader *esmeDefinitionYamlReader) parseFile(fileName string) (*esme, []*smppBindInfo, error) {
	return nil, nil, nil
}

type brokerForMessagesToSend struct {
}

func newBrokerForMessagesToSend() *brokerForMessagesToSend {
	return &brokerForMessagesToSend{}
}

func (broker *brokerForMessagesToSend) retrieveSendMessageChannel() <-chan *messageDescriptor {
	return nil
}

func (broker *brokerForMessagesToSend) startListening() {
}

type outputter struct {
}

func newOutputter() *outputter {
	return &outputter{}
}

func (outputter *outputter) sayThatMessageWasReceived(message *smpp.PDU, peerIP net.IP, peerPort uint16) {

}

func (outputter *outputter) dieIfError(err error) {

}

// esme-cluster <yaml_file>
//
func main() {
	outputter := newOutputter()

	yamlFileName, err := parseCommandLine()
	outputter.dieIfError(err)

	yamlReader := newEsmeDefinitionYamlReader()

	thisEsme, smscRemoteBinds, err := yamlReader.parseFile(yamlFileName)
	outputter.dieIfError(err)

	broker := newBrokerForMessagesToSend()
	channelOfMessagesToSend := broker.retrieveSendMessageChannel()
	go broker.startListening()

	for _, smsc := range smscRemoteBinds {
		err := thisEsme.performTranceiverBindTo(smsc)
		outputter.dieIfError(err)
	}

	go thisEsme.startListening()

	esmeEventChannel := thisEsme.incomingEventChannel()

	for {
		select {
		case messageDescriptor := <-channelOfMessagesToSend:
			thisEsme.sendMessageToPeer(messageDescriptor)

		case event := <-esmeEventChannel:
			switch event.Type {
			case receivedMessage:
				outputter.sayThatMessageWasReceived(event.smppPDU, event.peerIP, event.peerPort)
			}
		}
	}
}

func parseCommandLine() (string, error) {
	return "", nil
}
