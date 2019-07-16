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

	esmeEventChannel := make(chan *smppth.AgentEvent, len(esmes))

	for _, esme := range esmes {
		app.rememberEsmeObjectByName(esme.Name(), esme)
		go esme.StartEventLoop(esmeEventChannel)
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
				interactionBroker.NotifyThatSmppPduWasReceived(event.SmppPDU, event.SourceEsme.Name(), event.NameOfMessageSender)

			case smppth.CompletedBind:
				interactionBroker.NotifyThatBindWasCompletedWithPeer(event.SourceEsme.Name(), event.BoundPeerName)
			}

		case descriptorOfMessageToSend := <-channelOfMessagesToSend:
			esme := app.retrieveEsmeObjectByName(descriptorOfMessageToSend.SendFromEsmeNamed)

			if esme == nil {
				interactionBroker.NotifyOfPduSendAttemptFromUnknownEsme(descriptorOfMessageToSend.SendFromEsmeNamed)
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

func (app *esmeClusterApplication) rememberEsmeObjectByName(esmeName string, esmeObject *smppth.ESME) {
	app.esmeObjectByEsmeName[esmeName] = esmeObject
}

func (app *esmeClusterApplication) retrieveEsmeObjectByName(esmeName string) *smppth.ESME {
	return app.esmeObjectByEsmeName[esmeName]
}

func (app *esmeClusterApplication) getIoWriterStreamHandleForFileNamed(fileName string) (io.Writer, error) {
	return os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY, 0640)
}
