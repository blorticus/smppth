package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path"
	"smpp"
)

type esmeEventType int

const (
	receivedMessage esmeEventType = iota
)

type messageDescriptor struct {
	sendFromEsmeNamed string
	sendToSmscNamed   string
	pdu               *smpp.PDU
}

type esmeListenerEvent struct {
	Type       esmeEventType
	sourceEsme *esme
	smppPDU    *smpp.PDU
	peerIP     net.IP
	peerPort   uint16
}

type esmeDescriptor struct {
}

type outputter struct {
}

func newOutputter() *outputter {
	return &outputter{}
}

func (outputter *outputter) sayThatMessageWasReceived(message *smpp.PDU, peerIP net.IP, peerPort uint16) {

}

func (outputter *outputter) dieIfError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

// esme-cluster <yaml_file>
//
func main() {
	app := newEsmeClusterApplication()
	outputter := newOutputter()

	yamlFileName, err := app.parseCommandLine()
	outputter.dieIfError(err)

	yamlReader := newApplicationConfigYamlReader()

	esmes, _, err := yamlReader.parseFile(yamlFileName)
	outputter.dieIfError(err)

	esmeEventListenerConcentrator := newEventChannelConcentrator()

	for _, esme := range esmes {
		app.mapEsmeNameToItsReceiverChannel(esme.name, esme.outgoingMessageChannel())
		esmeEventListenerConcentrator.addEventChannelForEsmeNamed(esme.name, esme.incomingEventChannel())
		go esme.startListening()
	}

	appControlBroker := newApplicationControlBroker()
	channelOfMessagesToSend := appControlBroker.retrieveSendMessageChannel()
	go appControlBroker.startListening()

	esmeEventListenerConcentrator.startReceivingEvents()
	eventConcentratorChannel := esmeEventListenerConcentrator.retrieveConcentrationChannel()

	for {
		select {
		case event := <-eventConcentratorChannel:
			if event.Type == receivedMessage {
				outputter.sayThatMessageWasReceived(event.smppPDU, event.peerIP, event.peerPort)
			}

		case messageToSendDescriptor := <-channelOfMessagesToSend:
			sendChannel := app.retrieveOutgoingMessageChannelFromEsmeNamed(messageToSendDescriptor.sendFromEsmeNamed)
			sendChannel <- messageToSendDescriptor
		}
	}
}

type esmeClusterApplication struct {
}

func newEsmeClusterApplication() *esmeClusterApplication {
	return &esmeClusterApplication{}
}

func (app *esmeClusterApplication) parseCommandLine() (string, error) {
	if len(os.Args) != 2 {
		return "", errors.New(app.syntaxString())
	}

	return os.Args[1], nil
}

func (app *esmeClusterApplication) syntaxString() string {
	return fmt.Sprintf("%s <yaml_file>", path.Base(os.Args[0]))
}

func (app *esmeClusterApplication) addEsmePeer(emse *esme) {

}

func (app *esmeClusterApplication) mapEsmeNameToItsReceiverChannel(esmeName string, messageChannel chan<- *messageDescriptor) {

}

func (app *esmeClusterApplication) retrieveOutgoingMessageChannelFromEsmeNamed(esmeName string) chan<- *messageDescriptor {
	return nil
}
