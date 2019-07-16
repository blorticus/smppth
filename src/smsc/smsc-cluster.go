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

	yamlFileName, outputFileName, err := app.parseCommandLine()
	app.dieIfError(err)

	yamlReader := smppth.NewApplicationConfigYamlReader()

	_, smscs, err := yamlReader.ParseFile(yamlFileName)
	app.dieIfError(err)

	agentEventChannel := make(chan *smppth.AgentEvent, len(smscs))

	for _, smsc := range smscs {
		app.rememberSmscObjectByName(smsc.Name(), smsc)
		go smsc.StartEventLoop(agentEventChannel)
	}

	fileWriterStream, err := app.getIoWriterStreamHandleForFileNamed(outputFileName)
	app.dieIfError(err)

	interactionBroker := smppth.NewInteractionBroker().SetInputPromptStream(os.Stdout).SetInputReader(os.Stdin).SetOutputWriter(fileWriterStream)
	channelOfMessagesToSend := interactionBroker.RetrieveSendMessageChannel()
	go interactionBroker.BeginInteractiveSession()

	for {
		select {
		case event := <-agentEventChannel:
			switch event.Type {
			case smppth.ReceivedMessage:
				interactionBroker.NotifyThatSmppPduWasReceived(event.SmppPDU, event.SourceAgent.Name(), event.RemotePeerName)

			case smppth.CompletedBind:
				interactionBroker.NotifyThatBindWasCompletedWithPeer(event.SourceAgent.Name(), event.RemotePeerName)
			}

		case descriptorOfMessageToSend := <-channelOfMessagesToSend:
			smsc := app.retrieveSmscByItsName(descriptorOfMessageToSend.NameOfSourcePeer)

			if smsc == nil {
				interactionBroker.NotifyOfPduSendAttemptFromUnknownAgent(descriptorOfMessageToSend.NameOfSourcePeer)
				continue
			}

			smsc.SendMessageToPeer(descriptorOfMessageToSend)
		}
	}
}

type smscClusterApplication struct {
	smscObjectBySmscName map[string]*smppth.SMSC
}

func newSmscClusterApplication() *smscClusterApplication {
	return &smscClusterApplication{smscObjectBySmscName: make(map[string]*smppth.SMSC)}
}

func (app *smscClusterApplication) dieIfError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func (app *smscClusterApplication) parseCommandLine() (string, string, error) {
	switch len(os.Args) {
	case 2:
		return os.Args[1], "/tmp/smsc-output.txt", nil

	case 3:
		return os.Args[1], os.Args[2], nil

	default:
		return "", "", errors.New(app.syntaxString())
	}
}

func (app *smscClusterApplication) syntaxString() string {
	return fmt.Sprintf("%s <yaml_file>", path.Base(os.Args[0]))
}

func (app *smscClusterApplication) getIoWriterStreamHandleForFileNamed(fileName string) (io.Writer, error) {
	return os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY, 0640)
}

func (app *smscClusterApplication) rememberSmscObjectByName(smscName string, smsc *smppth.SMSC) {
	app.smscObjectBySmscName[smscName] = smsc
}

func (app *smscClusterApplication) retrieveSmscByItsName(smscName string) *smppth.SMSC {
	return app.smscObjectBySmscName[smscName]
}
