package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"smppth"
)

// smpp-test-agent run esmes <yaml_file> [<output_file>]
// smpp-test-agent run smscs <yaml_file> [<output_file>]
//
func main() {
	app := newSmppTestAgentApplication()

	agentType, yamlFileName, eventOutputFileName, err := app.parseCommandLine()
	app.dieIfError(err)

	yamlReader := smppth.NewApplicationConfigYamlReader()

	var agents []smppth.Agent

	if agentType == "esmes" {
		esmes, _, err := yamlReader.ParseFile(yamlFileName)
		app.dieIfError(err)
		agents = make([]smppth.Agent, len(esmes))
		for i, v := range esmes {
			agents[i] = v
		}
	} else {
		_, smscs, err := yamlReader.ParseFile(yamlFileName)
		app.dieIfError(err)
		agents = make([]smppth.Agent, len(smscs))
		for i, v := range smscs {
			agents[i] = v
		}
	}

	sharedEventChannel := make(chan *smppth.AgentEvent, len(agents))

	for _, agent := range agents {
		app.rememberAgentObjectByName(agent.Name(), agent)
		go agent.StartEventLoop(sharedEventChannel)
	}

	fileWriterStream, err := app.getIoWriterStreamHandleForFileNamed(eventOutputFileName)
	app.dieIfError(err)

	interactionBroker := smppth.NewInteractionBroker().SetInputPromptStream(os.Stdout).SetInputReader(os.Stdin).SetOutputWriter(fileWriterStream)
	channelOfMessagesToSend := interactionBroker.RetrieveSendMessageChannel()
	go interactionBroker.BeginInteractiveSession()

	for {
		select {
		case event := <-sharedEventChannel:
			switch event.Type {
			case smppth.ReceivedBind:
				interactionBroker.NotifyThatSmppPduWasReceived(event.SmppPDU, event.SourceAgent.Name(), event.RemotePeerName)

			case smppth.SentBind:
				interactionBroker.NotifyThatSmppPduWasSentToPeer(event.SmppPDU, event.SourceAgent.Name(), event.RemotePeerName)

			case smppth.CompletedBind:
				interactionBroker.NotifyThatBindWasCompletedWithPeer(event.SourceAgent.Name(), event.RemotePeerName)

			case smppth.ReceivedMessage:
				interactionBroker.NotifyThatSmppPduWasReceived(event.SmppPDU, event.SourceAgent.Name(), event.RemotePeerName)

			case smppth.SentMessage:
				interactionBroker.NotifyThatSmppPduWasSentToPeer(event.SmppPDU, event.SourceAgent.Name(), event.RemotePeerName)
			}

		case descriptorOfMessageToSend := <-channelOfMessagesToSend:
			agent := app.retrieveEsmeObjectByName(descriptorOfMessageToSend.NameOfSourcePeer)

			if agent == nil {
				interactionBroker.NotifyOfPduSendAttemptFromUnknownAgent(descriptorOfMessageToSend.NameOfSourcePeer)
				continue
			}

			agent.SendMessageToPeer(descriptorOfMessageToSend)
			interactionBroker.NotifyThatSmppPduWasSentToPeer(descriptorOfMessageToSend.PDU, descriptorOfMessageToSend.NameOfSourcePeer, descriptorOfMessageToSend.NameOfRemotePeer)
		}
	}
}

type smppTestAgentApplication struct {
	agentObjectByAgentName map[string]smppth.Agent
}

func newSmppTestAgentApplication() *smppTestAgentApplication {
	return &smppTestAgentApplication{agentObjectByAgentName: make(map[string]smppth.Agent)}
}

func (app *smppTestAgentApplication) dieIfError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func (app *smppTestAgentApplication) parseCommandLine() (agentRunType string, yamlFileName string, writeOutputFileName string, err error) {
	if len(os.Args) < 4 || os.Args[1] != "run" || (os.Args[2] != "smscs" && os.Args[2] != "esmes") {
		return "", "", "", errors.New(app.syntaxString())
	}

	agentRunType = os.Args[2]
	yamlFileName = os.Args[3]

	switch len(os.Args) {
	case 4:
		writeOutputFileName = fmt.Sprintf("/tmp/%s-output.log", agentRunType)
	case 5:
		writeOutputFileName = os.Args[4]
	default:
		return "", "", "", errors.New(app.syntaxString())
	}

	return agentRunType, yamlFileName, writeOutputFileName, nil
}

func (app *smppTestAgentApplication) syntaxString() string {
	return fmt.Sprintf("%s run esmes|smscs <yaml_file> [<event_output_file>]", path.Base(os.Args[0]))
}

func (app *smppTestAgentApplication) rememberAgentObjectByName(agentName string, agentObject smppth.Agent) {
	app.agentObjectByAgentName[agentName] = agentObject
}

func (app *smppTestAgentApplication) retrieveEsmeObjectByName(agentName string) smppth.Agent {
	return app.agentObjectByAgentName[agentName]
}

func (app *smppTestAgentApplication) getIoWriterStreamHandleForFileNamed(fileName string) (io.Writer, error) {
	return os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY, 0640)
}
