package smppth

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"smpp"
	"strings"
)

type interactionBrokerSendCommand struct {
	pduSenderName        string
	pduReceiverName      string
	commandTypeName      string
	commandParametersMap map[string]string
}

type interactionBrokerCommandMatcher struct {
	digestingCommand                                string
	sendCommandMatcher                              *regexp.Regexp
	sendCommandParametersMatcher                    *regexp.Regexp
	emptyParameterMatcher                           *regexp.Regexp
	emptyLastParameterMatcher                       *regexp.Regexp
	doubleQuotedParameterMatcher                    *regexp.Regexp
	singleQuotedParameterMatcher                    *regexp.Regexp
	unquotedParameterMatcher                        *regexp.Regexp
	lastMatchStringSet                              []string
	lastSendCommandParameterSetIncludedShortMessage bool
}

func newInteractionBrokerCommandMatcher() *interactionBrokerCommandMatcher {
	return &interactionBrokerCommandMatcher{
		digestingCommand:                                "",
		sendCommandMatcher:                              regexp.MustCompile(`^(\S+?): send (\S+) to (\S+) *(.*)?$`),
		sendCommandParametersMatcher:                    regexp.MustCompile(`^ *short_message="(.+?)" *$`),
		emptyParameterMatcher:                           regexp.MustCompile(`^(\S+)=\s+`),
		emptyLastParameterMatcher:                       regexp.MustCompile(`^(\S+)=$`),
		doubleQuotedParameterMatcher:                    regexp.MustCompile(`^(\S+)="(.+?)"\s*`),
		singleQuotedParameterMatcher:                    regexp.MustCompile(`^(\S+)='(.+?)'\s*`),
		unquotedParameterMatcher:                        regexp.MustCompile(`^(\S+)=(\S+)\s*`),
		lastMatchStringSet:                              []string{},
		lastSendCommandParameterSetIncludedShortMessage: false,
	}
}

func (matcher *interactionBrokerCommandMatcher) digestCommand(command string) {
	matcher.digestingCommand = command
}

func (matcher *interactionBrokerCommandMatcher) saysThisIsAValidSendCommand() bool {
	matcher.lastMatchStringSet = matcher.sendCommandMatcher.FindStringSubmatch(matcher.digestingCommand)

	if len(matcher.lastMatchStringSet) > 0 {
		return true
	}

	return false
}

func (matcher *interactionBrokerCommandMatcher) breakSendCommandIntoStruct() *interactionBrokerSendCommand {
	return &interactionBrokerSendCommand{
		pduSenderName:        matcher.lastMatchStringSet[1],
		pduReceiverName:      matcher.lastMatchStringSet[3],
		commandTypeName:      matcher.lastMatchStringSet[2],
		commandParametersMap: matcher.breakParametersIntoMap(matcher.lastMatchStringSet[4]),
	}
}

func (matcher *interactionBrokerCommandMatcher) breakParametersIntoMap(parameterString string) map[string]string {
	parameterMap := make(map[string]string)

	for len(parameterString) > 0 {
		foundSomeTypeOfMatch := false

		for _, compiledMatcher := range []*regexp.Regexp{matcher.emptyLastParameterMatcher, matcher.emptyParameterMatcher, matcher.doubleQuotedParameterMatcher, matcher.singleQuotedParameterMatcher, matcher.unquotedParameterMatcher} {
			itDoesMatch, parameterName, parameterValue, parameterStringLength := matcher.extractMappableValueAndMatchingLengthFromMatcher(compiledMatcher, parameterString)

			if itDoesMatch {
				parameterMap[parameterName] = parameterValue
				parameterString = parameterString[parameterStringLength:]
				foundSomeTypeOfMatch = true
				break
			}
		}

		if !foundSomeTypeOfMatch {
			break
		}
	}

	return parameterMap
}

func (matcher *interactionBrokerCommandMatcher) extractMappableValueAndMatchingLengthFromMatcher(compiledRegexp *regexp.Regexp, parseString string) (doesMatch bool, name string, value string, matchLen int) {
	groups := compiledRegexp.FindStringSubmatch(parseString)

	if groups == nil {
		return false, "", "", 0
	}

	if len(groups) == 2 {
		return true, groups[1], "", len(groups[0])
	}

	return true, groups[1], groups[2], len(groups[0])
}

func (matcher *interactionBrokerCommandMatcher) saysThatSendCommandParametersContainedShortMessage() bool {
	return matcher.lastSendCommandParameterSetIncludedShortMessage
}

// InteractionBroker connects to a reader, which accepts structured command messages, and a writer,
// which emits events on behalf of testharness agents that have been started.
type InteractionBroker struct {
	channelOfPdusToSend chan *MessageDescriptor
	outputWriter        io.Writer
	inputReader         io.Reader
	inputPromptStream   io.Writer
	promptString        string
	inputByteStream     []byte
	commandMatcher      *interactionBrokerCommandMatcher
}

// NewInteractionBroker creates an empty broker, where the prompt output stream is set to STDOUT,
// the command input stream is set to STDIN, and the event output writer is set to STDOUT.
func NewInteractionBroker() *InteractionBroker {
	return &InteractionBroker{
		channelOfPdusToSend: make(chan *MessageDescriptor),
		outputWriter:        os.Stdout,
		inputReader:         os.Stdin,
		inputPromptStream:   os.Stdout,
		promptString:        "> ",
		inputByteStream:     make([]byte, 9000),
	}
}

// SetInputReader sets the command input reader for the broker, and returns the broker so that
// this can be chained with other broker Set... commands.
func (broker *InteractionBroker) SetInputReader(reader io.Reader) *InteractionBroker {
	broker.inputReader = reader
	return broker
}

// SetInputPromptStream sets the command prompt output stream for the broker, and returns the broker so that
// this can be chained with other broker Set... commands.
func (broker *InteractionBroker) SetInputPromptStream(writer io.Writer) *InteractionBroker {
	broker.inputPromptStream = writer
	return broker
}

// SetOutputWriter sets the command event output stream for the broker, and returns the broker so that
// this can be chained with other broker Set... commands.
func (broker *InteractionBroker) SetOutputWriter(writer io.Writer) *InteractionBroker {
	broker.outputWriter = writer
	return broker
}

// RetrieveSendMessageChannel returns the MessageDescriptor channel, which is used by the broker
// to accept MessageDescriptors.  These are routed to the appropriate testharness agents based
// on the MessageDescriptor fields.
func (broker *InteractionBroker) RetrieveSendMessageChannel() <-chan *MessageDescriptor {
	return broker.channelOfPdusToSend
}

// BeginInteractiveSession instructs the broker to send the prompt to the prompt output stream,
// wait for a command on the command input stream, attempt to execute the command, and send any
// results to the event output stream.  This cycle (prompt, read, write) is repeated indefinitely.
func (broker *InteractionBroker) BeginInteractiveSession() {
	broker.commandMatcher = newInteractionBrokerCommandMatcher()

	for {
		nextCommand := broker.promptForNextCommand()

		if nextCommand == "help" {
			broker.WriteOutHelp()
		} else {
			broker.commandMatcher.digestCommand(nextCommand)
			if broker.commandMatcher.saysThisIsAValidSendCommand() {
				sendCommandStruct := broker.commandMatcher.breakSendCommandIntoStruct()

				pdu, err := broker.createPduFromCommand(sendCommandStruct)

				if err != nil {
					broker.writeLine(err.Error())
				} else {
					broker.channelOfPdusToSend <- &MessageDescriptor{PDU: pdu, NameOfSourcePeer: sendCommandStruct.pduSenderName, NameOfRemotePeer: sendCommandStruct.pduReceiverName}
				}
			} else {
				broker.writeLine("Command not understood")
			}
		}
	}
}

func (broker *InteractionBroker) createPduFromCommand(commandDetails *interactionBrokerSendCommand) (*smpp.PDU, error) {
	commandID, commandIDIsUnderstood := smpp.CommandIDFromString(commandDetails.commandTypeName)

	if !commandIDIsUnderstood {
		return nil, fmt.Errorf("Message command (%s) not understood", commandDetails.commandTypeName)
	}

	switch commandID {
	case smpp.CommandEnquireLink:
		return broker.attemptToMakeEnquireLinkPdu(commandDetails.commandParametersMap)

	case smpp.CommandSubmitSm:
		return broker.attemptToMakeSubmitSmPdu(commandDetails.commandParametersMap)
	default:
		return nil, fmt.Errorf("While (%s) is a valid command, I don't know how to generate a PDU of that type", commandDetails.commandTypeName)
	}
}

func (broker *InteractionBroker) attemptToMakeEnquireLinkPdu(commandParameterMap map[string]string) (*smpp.PDU, error) {
	if len(commandParameterMap) > 0 {
		return nil, fmt.Errorf("When sending an enquire-link PDU, no additional parameters are allowed")
	}

	return smpp.NewPDU(smpp.CommandEnquireLink, 0, 1, []*smpp.Parameter{}, []*smpp.Parameter{}), nil
}

func (broker *InteractionBroker) attemptToMakeSubmitSmPdu(commandParameterMap map[string]string) (*smpp.PDU, error) {
	shortMessage, shortMessageIsInMap := commandParameterMap["short_message"]
	destAddr, destAddrIsInMap := commandParameterMap["dest_addr"]
	destAddrNpi := uint8(0)

	if !shortMessageIsInMap {
		shortMessage = "Sample Short Message"
	}

	if !destAddrIsInMap {
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

func (broker *InteractionBroker) promptForNextCommand() string {
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

func (broker *InteractionBroker) read() string {
	bytesRead, err := broker.inputReader.Read(broker.inputByteStream)
	broker.panicIfError(err)

	return string(broker.inputByteStream[:bytesRead])
}

func (broker *InteractionBroker) write(outputString string) {
	_, err := broker.outputWriter.Write([]byte(outputString))
	broker.panicIfError(err)
}

func (broker *InteractionBroker) writeLine(outputString string) {
	broker.write(outputString)
	broker.write("\n")
}

func (broker *InteractionBroker) discardInputUntilNewline() {
	for input := broker.read(); input[len(input)-1] != byte('\n'); {
	}
}

// NotifyThatSmppPduWasReceived instructs the broker to write an output event message indicating that an SMPP PDU was received
// from a peer.  The message type, sequence number, and -- depending on the message type -- other attributes are ouput
func (broker *InteractionBroker) NotifyThatSmppPduWasReceived(pdu *smpp.PDU, nameOfReceivingEsme string, nameOfRemoteSender string) {
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
func (broker *InteractionBroker) NotifyThatBindWasCompletedWithPeer(nameOfBindingEsme string, nameOfBoundPeer string) {
	broker.outputWriter.Write([]byte(fmt.Sprintf("(%s) completed transceiver-bind with %s\n", nameOfBindingEsme, nameOfBoundPeer)))
}

// NotifyThatSmppPduWasSentToPeer instructs the broker to write an output event message indicating that a local agent successfully sent a message to
// a remote peer
func (broker *InteractionBroker) NotifyThatSmppPduWasSentToPeer(pduSentToPeer *smpp.PDU, nameOfSendingPeer string, nameOfReceivingPeer string) {
	broker.outputWriter.Write([]byte(fmt.Sprintf("(%s) sent %s to %s, seqNum=(%d)\n", nameOfSendingPeer, pduSentToPeer.CommandName(), nameOfReceivingPeer, pduSentToPeer.SequenceNumber)))
}

// NotifyThatErrorOccurredWhileTryingToSendMessage instructs the broker to write an output event message indicating an attempt to send a message
// from an agent to a remote peer failed.
func (broker *InteractionBroker) NotifyThatErrorOccurredWhileTryingToSendMessage(err error, pduThatAgentAttemptedToSend *smpp.PDU, nameOfSendingPeer string, nameOfReceivingPeer string) {
	broker.outputWriter.Write([]byte(fmt.Sprintf("[ERROR] failed to send %s from %s to %s: %s\n", pduThatAgentAttemptedToSend.CommandName(), nameOfSendingPeer, nameOfReceivingPeer, err)))
}

// WriteOutHelp instructs the broker to write an output event message listing the various possible input commands and their
// parameters.
func (broker *InteractionBroker) WriteOutHelp() {
	helpText := `
$esme_name: send submit-sm to $smsc_name short_message="$message" dest_addr=$addr
$esme_name: send enquire-link to $smsc_name
`
	broker.outputWriter.Write([]byte(helpText))
}

func (broker *InteractionBroker) panicIfError(err error) {
	if err != nil {
		panic(err)
	}
}
