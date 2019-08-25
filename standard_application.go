package smppth

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"time"

	smpp "github.com/blorticus/smpp-go"
)

// StandardApplication provides a standard method for accepting user input commands (marshalled to UserCommand structs),
// generate smpp PDUs (if the command is 'SendPDU'), respond to known PDU types (e.g., send enquire-link-resp in response
// to an enquire-link), and produce standard text output to an io.Writer, based on a supplied OutputGenerator.  By default,
// the application uses a StandardOutputGenerate and a DefaultPduFactory, but these can be overridden.  Some of the output
// is based on events that arrive on a supplied AgentEvent channel.  If the creator of the StandardApplication would also
// like to see the AgentEvent messages, a call to AttachEventChannel() returns a channel to which the AgentEvent messages
// are proxied.
//
// Generally speaking, the application is used in this way:
//   app := NewStandardApplication().
//             SetOutputGenerator(someOutputGenerator).
//             SetEventOutputWriter(someIoWriter).
//             SetPduFactory(somePduFactory).
//             SetAgentGroup(someAgentGroup)
//   proxiedEventChannel := app.AttachEventChannel(underlyingEventChannel)
//   go app.Start()
//     // ... some time later when user input arrives ... //
//   app.ReceiveNextCommand(someUserCommand)
type StandardApplication struct {
	outputGenerator             OutputGenerator
	eventOutputWriter           io.Writer
	pduFactory                  PduFactory
	agentGroup                  *AgentGroup
	incomingSharedEventChannel  <-chan *AgentEvent
	proxiedOutgoingEventChannel chan *AgentEvent
	automaticResponsesEnabled   bool
	debugLogger                 *log.Logger
	shouldProxyAgentEvents      bool
	quitCommandCallback         func()
}

// NewStandardApplication creates a new StandardApplication
func NewStandardApplication() *StandardApplication {
	return &StandardApplication{
		outputGenerator:             NewStandardOutputGenerator(),
		eventOutputWriter:           os.Stdout,
		pduFactory:                  NewDefaultPduFactory(),
		agentGroup:                  NewAgentGroup([]Agent{}),
		incomingSharedEventChannel:  nil,
		proxiedOutgoingEventChannel: make(chan *AgentEvent),
		automaticResponsesEnabled:   true,
		debugLogger:                 log.New(ioutil.Discard, "", 0),
		shouldProxyAgentEvents:      true,
		quitCommandCallback:         func() {},
	}
}

// DisableAutomaticResponsesToRequestPdus disables automatic responses to incoming PDUs.  The PDUs can still
// be received by the creator of the StandardApplication by reading from the proxiedEventChannel (returned
// when AttachEventChannel is invoked).
func (app *StandardApplication) DisableAutomaticResponsesToRequestPdus() *StandardApplication {
	app.automaticResponsesEnabled = false
	return app
}

// SetOutputGenerator adds something implementing the OutputGenerator interface, which is then used
// by the StandardApplication to generate output message (written to the EventOutputWriter).  If this
// option isn't applied, the default is to use a StandardOutputGenerator.
func (app *StandardApplication) SetOutputGenerator(generator OutputGenerator) *StandardApplication {
	app.outputGenerator = generator
	return app
}

// SetEventOutputWriter sets the io.Writer to which output messages are written.  If this option isn't
// applies, STDOUT is used.
func (app *StandardApplication) SetEventOutputWriter(writer io.Writer) *StandardApplication {
	app.eventOutputWriter = writer
	return app
}

// SetPduFactory adds something implementing the PduFactory interfaces, which is used to generate smpp PDUs
// based on SendPDU commands, or in response to received request PDUs.
func (app *StandardApplication) SetPduFactory(factory PduFactory) *StandardApplication {
	app.pduFactory = factory
	return app
}

// SetAgentGroup adds a managed AgentGroup, which is used to direct send messages on a SendPDU command.
func (app *StandardApplication) SetAgentGroup(group *AgentGroup) *StandardApplication {
	app.agentGroup = group
	return app
}

// OnQuit adds a callback when the user command is "quit"
func (app *StandardApplication) OnQuit(quitCommandCallback func()) *StandardApplication {
	app.quitCommandCallback = quitCommandCallback
	return app
}

// AttachEventChannel attaches a shared AgentEvent channel, generally the one used by the associated AgentGroup.
// An AgentEvent channel is returned.  Any message that arrives on incoming AgentEvent channel is copied to the
// proxy channel.  If DisableAgentEventProxying() is called, then nothing is written to the proxy channel.  Otherwise,
// messages are written in a non-blocking way (but you shouldn't let them collect for too long).
func (app *StandardApplication) AttachEventChannel(channel <-chan *AgentEvent) (proxiedEventChannel <-chan *AgentEvent) {
	app.incomingSharedEventChannel = channel
	return app.proxiedOutgoingEventChannel
}

// DisableAgentEventProxying instructs the StandardApplication to not (or stop) writing AgentEvents to the proxiedEventChannel
// returned by AttachEventChannel()
func (app *StandardApplication) DisableAgentEventProxying() *StandardApplication {
	app.shouldProxyAgentEvents = false
	return app
}

// EnableAgentEventProxying instructs the StandardApplication to start writing AgentEvents to the proxiedEventChannel
// returned by AttachEventChannel() when events arrive on the attached event channel
func (app *StandardApplication) EnableAgentEventProxying() *StandardApplication {
	app.shouldProxyAgentEvents = true
	return app
}

// EnableDebugMessages instructs the StandardApplication to output any debugging message embedded in the code.  Messages
// are written to the provided writer.
func (app *StandardApplication) EnableDebugMessages(writer io.Writer) {
	app.debugLogger = log.New(writer, "(StandardApplication): ", 0)
}

// DisableDebugMessages instructs the StandardApplication to stop sending any output for debugging messages embedded
// in the code.
func (app *StandardApplication) DisableDebugMessages() {
	app.debugLogger = log.New(ioutil.Discard, "", 0)
}

// Start begins the running application.  This method never returns, so it may be appropriate to launch it as
// a go routine.
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

		case PeerTransportClosed:
			app.respondToPeerTransportClosedEvent(nextAgentEvent)

		case TransportError:
			app.respondToTransportErrorEvent(nextAgentEvent)

		case ApplicationError:
			app.respondToApplicationErrorEvent(nextAgentEvent)
		}

		if app.shouldProxyAgentEvents {
			go app.writeToProxiedEventChannelWithoutBlockingThisFunction(nextAgentEvent)
		}
	}
}

// ReceiveNextCommand sends a structured UserCommand to the StandardApplication.  If the type is SendPDU,
// and the Details in the UserCommand are for a PDU type known to the StandardApplication, then the identified
// agent sends the message to the identified peer.  If there is an error (e.g., if the identified sending agent
// is not under management by the associated AgentGroup), the error text is written to the EventOutputWriter.
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

	case Help:
		fmt.Fprintf(app.eventOutputWriter, app.helpText())

	case Quit:
		app.quitCommandCallback()
	}
}

func (app *StandardApplication) respondToReceivedPduEvent(event *AgentEvent) {
	fmt.Fprintf(app.eventOutputWriter, app.outputGenerator.SayThatAPduWasReceivedByAnAgent(event.RemotePeerName, event.SourceAgent.Name(), event.SmppPDU))

	if app.automaticResponsesEnabled {
		switch event.SmppPDU.CommandID {
		case smpp.CommandEnquireLink:
			event.SourceAgent.SendMessageToPeer(&MessageDescriptor{
				NameOfSendingPeer:   event.SourceAgent.Name(),
				NameOfReceivingPeer: event.RemotePeerName,
				PDU:                 app.pduFactory.CreateEnquireLinkRespFromRequest(event.SmppPDU),
			})

		case smpp.CommandSubmitSm:
			event.SourceAgent.SendMessageToPeer(&MessageDescriptor{
				NameOfSendingPeer:   event.SourceAgent.Name(),
				NameOfReceivingPeer: event.RemotePeerName,
				PDU:                 app.pduFactory.CreateSubmitSmRespFromRequest(event.SmppPDU, event.SourceAgent.Name()),
			})
		}
	}
}

func (app *StandardApplication) respondToSentPduEvent(event *AgentEvent) {
	fmt.Fprintf(app.eventOutputWriter, app.outputGenerator.SayThatAPduWasSentByAnAgent(event.SourceAgent.Name(), event.RemotePeerName, event.SmppPDU))
}

func (app *StandardApplication) respondToCompletedBindEvent(event *AgentEvent) {
	fmt.Fprintf(app.eventOutputWriter, app.outputGenerator.SayThatATransceiverBindWasCompletedByAnAgent(event.SourceAgent.Name(), event.RemotePeerName))

}

func (app *StandardApplication) respondToPeerTransportClosedEvent(event *AgentEvent) {
	fmt.Fprintf(app.eventOutputWriter, app.outputGenerator.SayThatTheTransportForAPeerClosed(event.SourceAgent.Name(), event.RemotePeerName))
}

func (app *StandardApplication) respondToTransportErrorEvent(event *AgentEvent) {
	fmt.Fprintf(app.eventOutputWriter, app.outputGenerator.SayThatATransportErrorWasThrown(event.SourceAgent.Name(), event.RemotePeerName, event.Error))
}

func (app *StandardApplication) respondToApplicationErrorEvent(event *AgentEvent) {
	fmt.Fprintf(app.eventOutputWriter, app.outputGenerator.SayThatAnApplicationErrorWasThrown(event.SourceAgent.Name(), event.Error))
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

func (app *StandardApplication) helpText() string {
	return "<sending_agent_name>: send enquire-link to <peer_name>\n" +
		"<sending_agent_name>: send submit-sm to <peer_name> [params]\n" +
		"  params: [source_addr_ton=<ton_int>] [source_addr=<addr>] [dest_addr_ton=<ton_int>] [destination_addr=<addr>] [short_message=<message>]"
}
