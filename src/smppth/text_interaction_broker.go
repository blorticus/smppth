package smppth

import (
	"fmt"
	"io"
	"os"
	"smpp"
	"strings"
)

// TextInteractionBroker connects to a reader, which accepts structured command messages, and a writer,
// which emits events on behalf of testharness agents that have been started.
type TextInteractionBroker struct {
	outputWriter       io.Writer
	inputReader        io.Reader
	inputPromptStream  io.Writer
	promptString       string
	inputByteStream    []byte
	commandMatcher     *textInteractionBrokerCommandMatcher
	useBuiltInOuputs   bool
	proxyEventChannel  chan *AgentEvent
	managedAgentsGroup *AgentGroup
}

// NewTextInteractionBroker creates an empty broker, where the prompt output stream is set to STDOUT,
// the command input stream is set to STDIN, and the event output writer is set to STDOUT.  Built-in
// outputs are not used by default.
func NewTextInteractionBroker(agentGroup *AgentGroup) *TextInteractionBroker {
	return &TextInteractionBroker{
		managedAgentsGroup: agentGroup,
		outputWriter:       os.Stdout,
		inputReader:        os.Stdin,
		inputPromptStream:  os.Stdout,
		promptString:       "> ",
		inputByteStream:    make([]byte, 9000),
		useBuiltInOuputs:   false,
		proxyEventChannel:  make(chan *AgentEvent),
	}
}

// StartUsingBuiltInOutputs instructs the broker to generate the built-in outputs based on received
// events and send message calls
func (broker *TextInteractionBroker) StartUsingBuiltInOutputs() *TextInteractionBroker {
	broker.useBuiltInOuputs = true
	return broker
}

// SetInputReader sets the command input reader for the broker, and returns the broker so that
// this can be chained with other broker Set... commands.
func (broker *TextInteractionBroker) SetInputReader(reader io.Reader) *TextInteractionBroker {
	broker.inputReader = reader
	return broker
}

// SetInputPromptStream sets the command prompt output stream for the broker, and returns the broker so that
// this can be chained with other broker Set... commands.
func (broker *TextInteractionBroker) SetInputPromptStream(writer io.Writer) *TextInteractionBroker {
	broker.inputPromptStream = writer
	return broker
}

// SetOutputWriter sets the command event output stream for the broker, and returns the broker so that
// this can be chained with other broker Set... commands.
func (broker *TextInteractionBroker) SetOutputWriter(writer io.Writer) *TextInteractionBroker {
	broker.outputWriter = writer
	return broker
}

// InstuctAgentToSendMessage pushes a message for delivery from a managed peer to one of its remote peers.  Emits message
// on success or failure.  On failure, returns error from AgentGroup.
func (broker *TextInteractionBroker) InstuctAgentToSendMessage(messageDescriptor *MessageDescriptor) error {
	err := broker.managedAgentsGroup.RoutePduToAgentForSending(messageDescriptor.NameOfSourcePeer, messageDescriptor.NameOfRemotePeer, messageDescriptor.PDU)

	if err != nil {
		broker.NotifyThatErrorOccurredWhileTryingToSendMessage(err, messageDescriptor.PDU, messageDescriptor.NameOfSourcePeer, messageDescriptor.NameOfRemotePeer)
		return err
	}

	broker.NotifyThatSmppPduWasSentToPeer(messageDescriptor.PDU, messageDescriptor.NameOfSourcePeer, messageDescriptor.NameOfRemotePeer)

	return nil
}

// ProxyAgentEventChannel returns the MessageDescriptor channel, which is used by the broker
// to accept MessageDescriptors.  These are routed to the appropriate testharness agents based
// on the MessageDescriptor fields.
func (broker *TextInteractionBroker) ProxyAgentEventChannel() <-chan *AgentEvent {
	return broker.proxyEventChannel
}

// BeginInteractiveSession instructs the broker to send the prompt to the prompt output stream,
// wait for a command on the command input stream, attempt to execute the command, and send any
// results to the event output stream.  This cycle (prompt, read, write) is repeated indefinitely.
func (broker *TextInteractionBroker) BeginInteractiveSession() {
	broker.commandMatcher = newTextInteractionBrokerCommandMatcher()

	for {
	}
}

func (broker *TextInteractionBroker) repeatedlyPromptTheUserForInput(userInputChannel chan<- *textInteractionBrokerValidUserInputCommand) {
	commandMatcher := newTextInteractionBrokerCommandMatcher()

	for {
		nextCommandString := broker.promptForNextCommand()

		digestedCommand := commandMatcher.digestCommandString(nextCommandString)

		if digestedCommand.isNotValid {
			broker.notifyThatUserProvidedCommandedIsInvalid(digestedCommand.reasonCommandIsNotValid)
		} else {
			userInputChannel <- digestedCommand.compiledUserInputCommand
		}
	}
}

func (broker *TextInteractionBroker) createPduFromCommand(commandDetails *textInteractionBrokerSendCommand) (*smpp.PDU, error) {
	commandID, commandIDIsUnderstood := smpp.CommandIDFromString(commandDetails.smppCommandTypeName)

	if !commandIDIsUnderstood {
		return nil, fmt.Errorf("PDU command (%s) not understood", commandDetails.smppCommandTypeName)
	}

	switch commandID {
	case smpp.CommandEnquireLink:
		return broker.attemptToMakeEnquireLinkPdu(commandDetails.commandParametersMap)

	case smpp.CommandSubmitSm:
		return broker.attemptToMakeSubmitSmPdu(commandDetails.commandParametersMap)
	default:
		return nil, fmt.Errorf("While (%s) is a valid command, I don't know how to generate a PDU of that type", commandDetails.smppCommandTypeName)
	}
}

func (broker *TextInteractionBroker) attemptToMakeEnquireLinkPdu(commandParameterMap map[string]string) (*smpp.PDU, error) {
	if len(commandParameterMap) > 0 {
		return nil, fmt.Errorf("When sending an enquire-link PDU, no additional parameters are allowed")
	}

	return smpp.NewPDU(smpp.CommandEnquireLink, 0, 1, []*smpp.Parameter{}, []*smpp.Parameter{}), nil
}

func (broker *TextInteractionBroker) attemptToMakeSubmitSmPdu(commandParameterMap map[string]string) (*smpp.PDU, error) {
	shortMessage, customShortMessageWasProvided := commandParameterMap["short_message"]
	destAddr, customDestAddrWasProvided := commandParameterMap["dest_addr"]
	destAddrNpi := uint8(0)

	if !customShortMessageWasProvided {
		shortMessage = "Sample Short Message"
	}

	if !customDestAddrWasProvided {
		destAddr = ""
	} else {
		destAddrNpi = uint8(9)
	}

	if len(shortMessage) > 255 {
		shortMessage = shortMessage[:255]
	}

	return smpp.NewPDU(smpp.CommandSubmitSm, 0, 1, []*smpp.Parameter{
		smpp.NewFLParameter(uint8(0)),           // service_type
		smpp.NewFLParameter(uint8(0)),           // source_addr_ton
		smpp.NewFLParameter(uint8(0)),           // source_addr_npi
		smpp.NewCOctetStringParameter(""),       // source_addr
		smpp.NewFLParameter(uint8(0)),           // dest_addr_ton
		smpp.NewFLParameter(uint8(destAddrNpi)), // dest_addr_npi
		smpp.NewCOctetStringParameter(destAddr), // destination_addr
		smpp.NewFLParameter(uint8(0)),           // esm_class
		smpp.NewFLParameter(uint8(0)),           // protocol_id
		smpp.NewFLParameter(uint8(0)),           // priority_flag
		smpp.NewFLParameter(uint8(0)),           // scheduled_delivery_time
		smpp.NewFLParameter(uint8(0)),           // validity_period
		smpp.NewFLParameter(uint8(0)),           // registered_delivery
		smpp.NewFLParameter(uint8(0)),           // replace_if_present_flag
		smpp.NewFLParameter(uint8(0)),           // data_coding
		smpp.NewFLParameter(uint8(0)),           // sm_defalt_msg_id
		smpp.NewFLParameter(uint8(len(shortMessage))),
		smpp.NewOctetStringFromString(shortMessage),
	}, []*smpp.Parameter{}), nil
}

func (broker *TextInteractionBroker) promptForNextCommand() string {
	if broker.inputPromptStream != nil {
		broker.inputPromptStream.Write([]byte(broker.promptString))
	}

	input := broker.read()

	if broker.inputByteStream[len(input)-1] != byte('\n') {
		broker.writeLine("[ERROR] Command contains no newline or is too long.\n")
		broker.discardInputUntilNewline()

		return broker.promptForNextCommand()
	}

	return strings.TrimRight(string(input), "\n")
}

func (broker *TextInteractionBroker) read() string {
	bytesRead, err := broker.inputReader.Read(broker.inputByteStream)
	broker.panicIfError(err)

	return string(broker.inputByteStream[:bytesRead])
}

func (broker *TextInteractionBroker) write(outputString string) {
	_, err := broker.outputWriter.Write([]byte(outputString))
	broker.panicIfError(err)
}

func (broker *TextInteractionBroker) writeLine(outputString string) {
	broker.write(outputString)
	broker.write("\n")
}

func (broker *TextInteractionBroker) discardInputUntilNewline() {
	for input := broker.read(); input[len(input)-1] != byte('\n'); {
	}
}

// NotifyThatSmppPduWasReceived instructs the broker to write an output event message indicating that an SMPP PDU was received
// from a peer.  The message type, sequence number, and -- depending on the message type -- other attributes are ouput
func (broker *TextInteractionBroker) NotifyThatSmppPduWasReceived(pdu *smpp.PDU, nameOfReceivingEsme string, nameOfRemoteSender string) {
	switch pdu.CommandID {
	case smpp.CommandSubmitSm:
		shortMessageValueInterface := pdu.MandatoryParameters[17].Value
		shortMessageValue := shortMessageValueInterface.([]byte)
		broker.outputWriter.Write([]byte(fmt.Sprintf("(%s) received submit-sm from %s, SeqNum=%d, short_message=\"%s\", dest_addr=(%s)\n", nameOfReceivingEsme, nameOfRemoteSender, pdu.SequenceNumber, shortMessageValue, pdu.MandatoryParameters[6].Value.(string))))
	case smpp.CommandSubmitSmResp:
		broker.outputWriter.Write([]byte(fmt.Sprintf("(%s) received submit-sm-resp from %s, SeqNum=%d, message_id=(%s)\n", nameOfReceivingEsme, nameOfRemoteSender, pdu.SequenceNumber, pdu.MandatoryParameters[0].Value.(string))))
	default:
		broker.outputWriter.Write([]byte(fmt.Sprintf("(%s) received %s from %s, SeqNum=%d\n", nameOfReceivingEsme, pdu.CommandName(), nameOfRemoteSender, pdu.SequenceNumber)))
	}
}

// NotifyThatBindWasCompletedWithPeer instructs the broker to write an output event message indicating that a bind was completed
// with a remote peer.  The name of the peer with which the bind was completed is included in the output.
func (broker *TextInteractionBroker) NotifyThatBindWasCompletedWithPeer(nameOfBindingEsme string, nameOfBoundPeer string) {
	broker.outputWriter.Write([]byte(fmt.Sprintf("(%s) completed transceiver-bind with %s\n", nameOfBindingEsme, nameOfBoundPeer)))
}

// NotifyThatSmppPduWasSentToPeer instructs the broker to write an output event message indicating that a local agent successfully sent a message to
// a remote peer
func (broker *TextInteractionBroker) NotifyThatSmppPduWasSentToPeer(pduSentToPeer *smpp.PDU, nameOfSendingPeer string, nameOfReceivingPeer string) {
	broker.outputWriter.Write([]byte(fmt.Sprintf("(%s) sent %s to %s, seqNum=(%d)\n", nameOfSendingPeer, pduSentToPeer.CommandName(), nameOfReceivingPeer, pduSentToPeer.SequenceNumber)))
}

// NotifyThatErrorOccurredWhileTryingToSendMessage instructs the broker to write an output event message indicating an attempt to send a message
// from an agent to a remote peer failed.
func (broker *TextInteractionBroker) NotifyThatErrorOccurredWhileTryingToSendMessage(err error, pduThatAgentAttemptedToSend *smpp.PDU, nameOfSendingPeer string, nameOfReceivingPeer string) {
	broker.outputWriter.Write([]byte(fmt.Sprintf("[ERROR] failed to send %s from %s to %s: %s\n", pduThatAgentAttemptedToSend.CommandName(), nameOfSendingPeer, nameOfReceivingPeer, err)))
}

// WriteOutHelp instructs the broker to write an output event message listing the various possible input commands and their
// parameters.
func (broker *TextInteractionBroker) WriteOutHelp() {
	helpText := `
$esme_name: send submit-sm to $smsc_name short_message="$message" dest_addr=$addr
$esme_name: send enquire-link to $smsc_name
`
	broker.outputWriter.Write([]byte(helpText))
}

func (broker *TextInteractionBroker) notifyThatUserProvidedCommandedIsInvalid(reason string) {
	broker.writeLine(fmt.Sprintf("[ERROR] command not understand: %s", reason))
}

func (broker *TextInteractionBroker) panicIfError(err error) {
	if err != nil {
		panic(err)
	}
}
