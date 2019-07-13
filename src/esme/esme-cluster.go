package main

import (
	"errors"
	"fmt"
	"os"
	"path"
	"smpp"
)

type messageDescriptor struct {
	sendFromEsmeNamed string
	sendToSmscNamed   string
	pdu               *smpp.PDU
}

type esmeDescriptor struct {
}

type outputter struct {
}

func newOutputter() *outputter {
	return &outputter{}
}

func (outputter *outputter) sayThatMessageWasReceived(message *smpp.PDU, nameOfSender string) {

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

	esmeEventChannel := make(chan *esmeListenerEvent, len(esmes))

	for _, esme := range esmes {
		app.mapEsmeNameToItsReceiverChannel(esme.name, esme.outgoingMessageChannel())
		go esme.startListening(esmeEventChannel)
	}

	appControlBroker := newApplicationControlBroker()
	channelOfMessagesToSend := appControlBroker.retrieveSendMessageChannel()
	go appControlBroker.startListening()

	for {
		select {
		case event := <-esmeEventChannel:
			if event.Type == receivedMessage {
				outputter.sayThatMessageWasReceived(event.smppPDU, event.nameOfMessageSender)
			}

		case messageToSendDescriptor := <-channelOfMessagesToSend:
			sendChannel := app.retrieveOutgoingMessageChannelFromEsmeNamed(messageToSendDescriptor.sendFromEsmeNamed)
			sendChannel <- messageToSendDescriptor
		}
	}
}

type esmeClusterApplication struct {
	esmeReceiveChannelByEsmeName map[string]chan *messageDescriptor
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

func (app *esmeClusterApplication) mapEsmeNameToItsReceiverChannel(esmeName string, messageChannel chan *messageDescriptor) {
	app.esmeReceiveChannelByEsmeName[esmeName] = messageChannel
}

func (app *esmeClusterApplication) retrieveOutgoingMessageChannelFromEsmeNamed(esmeName string) chan *messageDescriptor {
	return nil
}
