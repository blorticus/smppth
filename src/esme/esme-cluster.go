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

	yamlFileName, eventOutputFileName, err := app.parseCommandLine()
	app.dieIfError(err)

	yamlReader := smppth.NewApplicationConfigYamlReader()

	esmes, _, err := yamlReader.ParseFile(yamlFileName)
	app.dieIfError(err)

	esmeEventChannel := make(chan *smppth.AgentEvent, len(esmes))

	for _, esme := range esmes {
		app.rememberEsmeObjectByName(esme.Name(), esme)
		go esme.StartEventLoop(esmeEventChannel)
	}

	fileWriterStream, err := app.getIoWriterStreamHandleForFileNamed(eventOutputFileName)
	app.dieIfError(err)

	interactionBroker := smppth.NewInteractionBroker().SetInputPromptStream(os.Stdout).SetInputReader(os.Stdin).SetOutputWriter(fileWriterStream)
	channelOfMessagesToSend := interactionBroker.RetrieveSendMessageChannel()
	go interactionBroker.BeginInteractiveSession()

	for {
		select {
		case event := <-esmeEventChannel:
			switch event.Type {
			case smppth.ReceivedMessage:
				interactionBroker.NotifyThatSmppPduWasReceived(event.SmppPDU, event.SourceAgent.Name(), event.RemotePeerName)

			case smppth.CompletedBind:
				interactionBroker.NotifyThatBindWasCompletedWithPeer(event.SourceAgent.Name(), event.RemotePeerName)
			}

		case descriptorOfMessageToSend := <-channelOfMessagesToSend:
			esme := app.retrieveEsmeObjectByName(descriptorOfMessageToSend.NameOfSourcePeer)

			if esme == nil {
				interactionBroker.NotifyOfPduSendAttemptFromUnknownAgent(descriptorOfMessageToSend.NameOfSourcePeer)
				continue
			}

			esme.SendMessageToPeer(descriptorOfMessageToSend)
		}
	}
}

type esmeClusterApplication struct {
	esmeObjectByEsmeName map[string]*smppth.ESME
}

func newEsmeClusterApplication() *esmeClusterApplication {
	return &esmeClusterApplication{esmeObjectByEsmeName: make(map[string]*smppth.ESME)}
}

func (app *esmeClusterApplication) dieIfError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func (app *esmeClusterApplication) parseCommandLine() (string, string, error) {
	switch len(os.Args) {
	case 2:
		return os.Args[1], "/tmp/esme-output.txt", nil

	case 3:
		return os.Args[1], os.Args[2], nil

	default:
		return "", "", errors.New(app.syntaxString())
	}
}

func (app *esmeClusterApplication) syntaxString() string {
	return fmt.Sprintf("%s <yaml_file> [<event_output_file>", path.Base(os.Args[0]))
}

func (app *esmeClusterApplication) rememberEsmeObjectByName(esmeName string, esmeObject *smppth.ESME) {
	app.esmeObjectByEsmeName[esmeName] = esmeObject
}

func (app *esmeClusterApplication) retrieveEsmeObjectByName(esmeName string) *smppth.ESME {
	return app.esmeObjectByEsmeName[esmeName]
}

func (app *esmeClusterApplication) getIoWriterStreamHandleForFileNamed(fileName string) (io.Writer, error) {
	return os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY, 0640)
}
