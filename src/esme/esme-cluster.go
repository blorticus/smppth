package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"smppth"
)

// esme-cluster <yaml_file>
//
func main() {
	app := newEsmeClusterApplication()

	yamlFileName, err := app.parseCommandLine()
	app.dieIfError(err)

	yamlReader := smppth.NewApplicationConfigYamlReader()

	esmes, _, err := yamlReader.ParseFile(yamlFileName)
	app.dieIfError(err)

	esmeEventChannel := make(chan *smppth.EsmeListenerEvent, len(esmes))

	for _, esme := range esmes {
		app.mapEsmeNameToItsReceiverChannel(esme.Name, esme.OutgoingMessageChannel())
		go esme.StartListening(esmeEventChannel)
	}

	fileWriterStream, err := app.getIoWriterStreamHandleForFileNamed("foo.out")
	app.dieIfError(err)

	interactionBroker := smppth.NewInteractionBroker().SetInputPromptStream(os.Stdout).SetInputReader(os.Stdin).SetOutputWriter(fileWriterStream)
	channelOfMessagesToSend := interactionBroker.RetrieveSendMessageChannel()
	go interactionBroker.BeginInteractiveSession()

	for {
		select {
		case event := <-esmeEventChannel:
			switch event.Type {
			case smppth.ReceivedMessage:
				interactionBroker.NotifyThatSmppPduWasReceived(event.SmppPDU, event.SourceEsme.Name, event.NameOfMessageSender)

			case smppth.CompletedBind:
				interactionBroker.NotifyThatBindWasCompletedWithPeer(event.SourceEsme.Name, event.BoundPeerName)
			}

		case descriptorOfMessageToSend := <-channelOfMessagesToSend:
			sendChannel := app.retrieveOutgoingMessageChannelFromEsmeNamed(descriptorOfMessageToSend.SendFromEsmeNamed)

			if sendChannel == nil {
				interactionBroker.NotifyOfPduSendAttemptFromUnknownEsme(descriptorOfMessageToSend.SendFromEsmeNamed)
				continue
			}

			sendChannel <- descriptorOfMessageToSend
		}
	}
}

type esmeClusterApplication struct {
	esmeReceiveChannelByEsmeName map[string]chan *smppth.MessageDescriptor
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

func (app *esmeClusterApplication) mapEsmeNameToItsReceiverChannel(esmeName string, messageChannel chan *smppth.MessageDescriptor) {
	app.esmeReceiveChannelByEsmeName[esmeName] = messageChannel
}

func (app *esmeClusterApplication) retrieveOutgoingMessageChannelFromEsmeNamed(esmeName string) chan *smppth.MessageDescriptor {
	return app.esmeReceiveChannelByEsmeName[esmeName]
}

func (app *esmeClusterApplication) getIoWriterStreamHandleForFileNamed(fileName string) (io.Writer, error) {
	return os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY, 0640)
}
