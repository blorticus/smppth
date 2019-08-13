package smppth

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"smpp"
	"time"
)

type StandardApplication struct {
	outputGenerator             OutputGenerator
	eventOutputWriter           io.Writer
	pduFactory                  PduFactory
	agentGroup                  *AgentGroup
	incomingSharedEventChannel  <-chan *AgentEvent
	proxiedOutgoingEventChannel chan *AgentEvent
	automaticResponsesEnabled   bool
	debugLogger                 *log.Logger
}

func NewStandardApplication() *StandardApplication {
	return &StandardApplication{
		outputGenerator:             NewStandardOutputGenerator(),
		eventOutputWriter:           os.Stdout,
		pduFactory:                  NewDefaultPduFactory(),
		agentGroup:                  NewAgentGroup([]Agent{}),
		incomingSharedEventChannel:  nil,
		proxiedOutgoingEventChannel: nil,
		automaticResponsesEnabled:   true,
		debugLogger:                 log.New(ioutil.Discard, "", 0),
	}
}

func (app *StandardApplication) DisableAutomaticResponsesToRequestPdus() *StandardApplication {
	app.automaticResponsesEnabled = false
	return app
}

func (app *StandardApplication) SetOutputGenerator(generator OutputGenerator) *StandardApplication {
	app.outputGenerator = generator
	return app
}

func (app *StandardApplication) SetEventOutputWriter(writer io.Writer) *StandardApplication {
	app.eventOutputWriter = writer
	return app
}

func (app *StandardApplication) SetPduFactory(factory PduFactory) *StandardApplication {
	app.pduFactory = factory
	return app
}

func (app *StandardApplication) SetAgentGroup(group *AgentGroup) *StandardApplication {
	app.agentGroup = group
	return app
}

func (app *StandardApplication) AttachEventChannel(channel <-chan *AgentEvent) (proxiedEventChannel <-chan *AgentEvent) {
	app.incomingSharedEventChannel = channel
	app.proxiedOutgoingEventChannel = make(chan *AgentEvent)
	return app.proxiedOutgoingEventChannel
}

func (app *StandardApplication) EnableDebugMessages(writer io.Writer) {
	app.debugLogger = log.New(writer, "(StandardApplication): ", 0)
}

func (app *StandardApplication) DisableDebugMessages() {
	app.debugLogger = log.New(ioutil.Discard, "", 0)
}

func (app *StandardApplication) Start() {
	for {
		if app.incomingSharedEventChannel == nil {
			<-time.After(time.Second)
		} else {
			break
		}
	}

	for {
		nextAgentEvent := <-app.incomingSharedEventChannel

		switch nextAgentEvent.Type {
		case ReceivedPDU:
			app.respondToReceivedPduEvent(nextAgentEvent)

		case SentPDU:
			app.respondToSentPduEvent(nextAgentEvent)

		case CompletedBind:
			app.respondToCompletedBindEvent(nextAgentEvent)
		}

		if app.proxiedOutgoingEventChannel != nil {
			go app.writeToProxiedEventChannelWithoutBlockingThisFunction(nextAgentEvent)
		}
	}
}

func (app *StandardApplication) ReceiveNextCommand(command *UserCommand) {
	switch command.Type {
	case SendPDU:
		commandDetails := command.Details.(*SendPduDetails)
		generatedPDU, err := app.tryToGeneratePDUFromUserCommandDetails(commandDetails)

		if err != nil {
			fmt.Fprintf(app.eventOutputWriter, err.Error())
		}

		err = app.agentGroup.RoutePduToAgentForSending(commandDetails.NameOfAgentThatWillSendPdu, commandDetails.NameOfPeerThatShouldReceivePdu, generatedPDU)
		if err != nil {
			fmt.Fprintf(app.eventOutputWriter, "Unable to send pdu (%s) from (%s) to (%s): %s", generatedPDU.CommandName(), commandDetails.NameOfAgentThatWillSendPdu, commandDetails.NameOfPeerThatShouldReceivePdu, err)
		}
	}
}

func (app *StandardApplication) respondToReceivedPduEvent(event *AgentEvent) {
	fmt.Fprintf(app.eventOutputWriter, app.outputGenerator.sayThatAPduWasReceivedByAnAgent(event.SourceAgent.Name(), event.RemotePeerName, event.SmppPDU))

	if app.automaticResponsesEnabled {
		switch event.SmppPDU.CommandID {
		case smpp.CommandEnquireLink:
			event.SourceAgent.SendMessageToPeer(&MessageDescriptor{
				NameOfRemotePeer: event.RemotePeerName,
				NameOfSourcePeer: event.SourceAgent.Name(),
				PDU:              app.pduFactory.CreateEnquireLinkRespFromRequest(event.SmppPDU),
			})

		case smpp.CommandSubmitSm:
			event.SourceAgent.SendMessageToPeer(&MessageDescriptor{
				NameOfRemotePeer: event.RemotePeerName,
				NameOfSourcePeer: event.SourceAgent.Name(),
				PDU:              app.pduFactory.CreateSubmitSmRespFromRequest(event.SmppPDU, event.SourceAgent.Name()),
			})
		}
	}
}

func (app *StandardApplication) respondToSentPduEvent(event *AgentEvent) {
	fmt.Fprintf(app.eventOutputWriter, app.outputGenerator.sayTheAPduWasSentByAnAgent(event.SourceAgent.Name(), event.RemotePeerName, event.SmppPDU))
}

func (app *StandardApplication) respondToCompletedBindEvent(event *AgentEvent) {
	fmt.Fprintf(app.eventOutputWriter, app.outputGenerator.sayTheATransceiverBindWasCompletedByAnAgent(event.SourceAgent.Name(), event.RemotePeerName))

}

func (app *StandardApplication) writeToProxiedEventChannelWithoutBlockingThisFunction(event *AgentEvent) {
	go func() { app.proxiedOutgoingEventChannel <- event }()
}

func (app *StandardApplication) tryToGeneratePDUFromUserCommandDetails(details *SendPduDetails) (*smpp.PDU, error) {
	switch details.TypeOfSmppPDU {
	case smpp.CommandSubmitSm:
		pdu, err := app.pduFactory.CreateSubmitSm(details.StringParametersMap)
		if err != nil {
			return nil, err
		}

		return pdu, nil

	case smpp.CommandEnquireLink:
		return app.pduFactory.CreateEnquireLink(), nil
	}

	return nil, fmt.Errorf("Don't know how to generate message of type (%s)", smpp.CommandName(details.TypeOfSmppPDU))
}
