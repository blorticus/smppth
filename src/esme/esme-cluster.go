package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
)

// esme-cluster <yaml_file>
//
func main() {
	app := newEsmeClusterApplication()

	yamlFileName, err := app.parseCommandLine()
	app.dieIfError(err)

	yamlReader := newApplicationConfigYamlReader()

	esmes, _, err := yamlReader.parseFile(yamlFileName)
	app.dieIfError(err)

	esmeEventChannel := make(chan *esmeListenerEvent, len(esmes))

	for _, esme := range esmes {
		app.mapEsmeNameToItsReceiverChannel(esme.name, esme.outgoingMessageChannel())
		go esme.startListening(esmeEventChannel)
	}

	fileWriterStream, err := app.getIoWriterStreamHandleForFileNamed("foo.out")
	app.dieIfError(err)

	interactionBroker := newInteractionBroker().setInputPromptStream(os.Stdout).setInputReader(os.Stdin).setOutputWriter(fileWriterStream)
	channelOfMessagesToSend := interactionBroker.retrieveSendMessageChannel()
	go interactionBroker.beginInteractiveSession()

	for {
		select {
		case event := <-esmeEventChannel:
			switch event.Type {
			case receivedMessage:
				interactionBroker.notifyThatSmppPduWasReceived(event.smppPDU, event.sourceEsme.name, event.nameOfMessageSender)

			case completedBind:
				interactionBroker.notifyThatBindWasCompletedWithPeer(event.sourceEsme.name, event.boundPeerName)
			}

		case descriptorOfMessageToSend := <-channelOfMessagesToSend:
			sendChannel := app.retrieveOutgoingMessageChannelFromEsmeNamed(descriptorOfMessageToSend.sendFromEsmeNamed)

			if sendChannel == nil {
				interactionBroker.notifyOfPduSendAttemptFromUnknownEsme(descriptorOfMessageToSend.sendFromEsmeNamed)
				continue
			}

			sendChannel <- descriptorOfMessageToSend
		}
	}
}

type esmeClusterApplication struct {
	esmeReceiveChannelByEsmeName map[string]chan *messageDescriptor
}

func newEsmeClusterApplication() *esmeClusterApplication {
	return &esmeClusterApplication{}
}

func (app *esmeClusterApplication) dieIfError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
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
	return app.esmeReceiveChannelByEsmeName[esmeName]
}

func (app *esmeClusterApplication) getIoWriterStreamHandleForFileNamed(fileName string) (io.Writer, error) {
	return os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY, 0640)
}
