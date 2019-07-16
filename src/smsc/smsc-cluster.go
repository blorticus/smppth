package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"smppth"
)

// smsc-cluster <yaml_file>
//
func main() {
	app := newSmscClusterApplication()

	yamlFileName, err := app.parseCommandLine()
	app.dieIfError(err)

	yamlReader := smppth.NewApplicationConfigYamlReader()

	_, smscs, err := yamlReader.ParseFile(yamlFileName)
	app.dieIfError(err)

	agentEventChannel := make(chan *smppth.AgentEvent, len(smscs))

	for _, smsc := range smscs {
		app.mapAgentNameToItsReceiverChannel(smsc.Name, smsc.ChannelOfMessagesForAgentToSend())
		go smsc.StartListening(agentEventChannel)
	}

	fileWriterStream, err := app.getIoWriterStreamHandleForFileNamed("foo.out")
	app.dieIfError(err)

	interactionBroker := smppth.NewInteractionBroker().SetInputPromptStream(os.Stdout).SetInputReader(os.Stdin).SetOutputWriter(fileWriterStream)
	channelOfMessagesToSend := interactionBroker.RetrieveSendMessageChannel()
	go interactionBroker.BeginInteractiveSession()

	for {
		select {
		case event := <-agentEventChannel:
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

type smscClusterApplication struct {
	esmeReceiveChannelByEsmeName map[string]chan *smppth.MessageDescriptor
}

func newSmscClusterApplication() *smscClusterApplication {
	return &smscClusterApplication{}
}

func (app *smscClusterApplication) dieIfError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func (app *smscClusterApplication) parseCommandLine() (string, error) {
	if len(os.Args) != 2 {
		return "", errors.New(app.syntaxString())
	}

	return os.Args[1], nil
}

func (app *smscClusterApplication) syntaxString() string {
	return fmt.Sprintf("%s <yaml_file>", path.Base(os.Args[0]))
}

func (app *smscClusterApplication) mapAgentNameToItsReceiverChannel(esmeName string, messageChannel chan *smppth.MessageDescriptor) {
	app.esmeReceiveChannelByEsmeName[esmeName] = messageChannel
}

func (app *smscClusterApplication) retrieveOutgoingMessageChannelFromEsmeNamed(esmeName string) chan *smppth.MessageDescriptor {
	return app.esmeReceiveChannelByEsmeName[esmeName]
}

func (app *smscClusterApplication) getIoWriterStreamHandleForFileNamed(fileName string) (io.Writer, error) {
	return os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY, 0640)
}
