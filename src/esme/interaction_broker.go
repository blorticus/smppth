package main

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
	onlyWhitespaceMatcher                           *regexp.Regexp
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
		sendCommandMatcher:           regexp.MustCompile(`^(\S+?): send (\S+) to (\S+) *(.*)?$`),
		sendCommandParametersMatcher: regexp.MustCompile(`^ *short_message="(.+?)" *$`),
		onlyWhitespaceMatcher:        regexp.MustCompile(`^\s*$`),
		emptyParameterMatcher:        regexp.MustCompile(`^(\S+)=\s+`),
		emptyLastParameterMatcher:    regexp.MustCompile(`^(\S+)=$`),
		doubleQuotedParameterMatcher: regexp.MustCompile(`^(\S+)="(.+?)"\s*`),
		singleQuotedParameterMatcher: regexp.MustCompile(`^(\S+)='(.+?)'\s*`),
		unquotedParameterMatcher:     regexp.MustCompile(`^(\S+)=(\S+)\s*`),
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

func (matcher *interactionBrokerCommandMatcher) extractSendCommandParamtersFromParameterString(parameterString string) (shortMessage string, err error) {
	matchingGroups := matcher.sendCommandParametersMatcher.FindStringSubmatch(parameterString)

	if matchingGroups == nil {
		matcher.lastSendCommandParameterSetIncludedShortMessage = false

		if !matcher.onlyWhitespaceMatcher.MatchString(parameterString) {
			return "", fmt.Errorf("Unknown parameters provided to set command")
		}

		return "", nil
	}

	matcher.lastSendCommandParameterSetIncludedShortMessage = true

	return matchingGroups[1], nil
}

func (matcher *interactionBrokerCommandMatcher) saysThatSendCommandParametersContainedShortMessage() bool {
	return matcher.lastSendCommandParameterSetIncludedShortMessage
}

type interactionBroker struct {
	channelOfPdusToSend chan *messageDescriptor
	outputWriter        io.Writer
	inputReader         io.Reader
	inputPromptStream   io.Writer
	promptString        string
	inputByteStream     []byte
	commandMatcher      *interactionBrokerCommandMatcher
}

func newInteractionBroker() *interactionBroker {
	return &interactionBroker{
		channelOfPdusToSend: make(chan *messageDescriptor),
		outputWriter:        os.Stdout,
		inputReader:         os.Stdin,
		inputPromptStream:   os.Stdout,
		promptString:        "> ",
		inputByteStream:     make([]byte, 9000),
	}
}

func (broker *interactionBroker) setInputReader(reader io.Reader) *interactionBroker {
	broker.inputReader = reader
	return broker
}

func (broker *interactionBroker) setInputPromptStream(writer io.Writer) *interactionBroker {
	broker.inputPromptStream = writer
	return broker
}

func (broker *interactionBroker) setOutputWriter(writer io.Writer) *interactionBroker {
	broker.outputWriter = writer
	return broker
}

func (broker *interactionBroker) retrieveSendMessageChannel() <-chan *messageDescriptor {
	return broker.channelOfPdusToSend
}

func (broker *interactionBroker) beginInteractiveSession() {
	broker.commandMatcher = newInteractionBrokerCommandMatcher()

	for {
		nextCommand := broker.promptForNextCommand()

		if nextCommand == "help" {
			broker.writeOutHelp()
		} else {
			broker.commandMatcher.digestCommand(nextCommand)
			if broker.commandMatcher.saysThisIsAValidSendCommand() {
				sendCommandStruct := broker.commandMatcher.breakSendCommandIntoStruct()

				pdu, err := broker.createPduFromCommand(sendCommandStruct)

				if err != nil {
					broker.writeLine(err.Error())
				} else {
					broker.channelOfPdusToSend <- &messageDescriptor{pdu: pdu, sendFromEsmeNamed: sendCommandStruct.pduSenderName, sendToSmscNamed: sendCommandStruct.pduReceiverName}
				}
			} else {
				broker.writeLine("Command not understood")
			}
		}
	}
}

func (broker *interactionBroker) createPduFromCommand(commandDetails *interactionBrokerSendCommand) (*smpp.PDU, error) {
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

func (broker *interactionBroker) attemptToMakeEnquireLinkPdu(commandParameterMap map[string]string) (*smpp.PDU, error) {
	if len(commandParameterMap) > 0 {
		return nil, fmt.Errorf("When sending an enquire-link PDU, no additional parameters are allowed")
	}

	return smpp.NewPDU(smpp.CommandEnquireLink, 0, 1, []*smpp.Parameter{}, []*smpp.Parameter{}), nil
}

func (broker *interactionBroker) attemptToMakeSubmitSmPdu(commandParameterMap map[string]string) (*smpp.PDU, error) {
	shortMessage, shortMessageIsInMap := commandParameterMap["short_message"]

	if !shortMessageIsInMap {
		shortMessage = "Sample Short Message"
	}

	if len(shortMessage) > 255 {
		shortMessage = shortMessage[:255]
	}

	return smpp.NewPDU(smpp.CommandSubmitSm, 0, 1, []*smpp.Parameter{
		smpp.NewFLParameter(uint8(0)),     // service_type
		smpp.NewFLParameter(uint8(0)),     // source_addr_ton
		smpp.NewFLParameter(uint8(0)),     // source_addr_npi
		smpp.NewCOctetStringParameter(""), // source_addr
		smpp.NewFLParameter(uint8(0)),     // dest_addr_ton
		smpp.NewFLParameter(uint8(0)),     // dest_addr_npi
		smpp.NewCOctetStringParameter(""), // destination_addr
		smpp.NewFLParameter(uint8(0)),     // esm_class
		smpp.NewFLParameter(uint8(0)),     // protocol_id
		smpp.NewFLParameter(uint8(0)),     // priority_flag
		smpp.NewFLParameter(uint8(0)),     // scheduled_delivery_time
		smpp.NewFLParameter(uint8(0)),     // validity_period
		smpp.NewFLParameter(uint8(0)),     // registered_delivery
		smpp.NewFLParameter(uint8(0)),     // replace_if_present_flag
		smpp.NewFLParameter(uint8(0)),     // data_coding
		smpp.NewFLParameter(uint8(0)),     // sm_defalt_msg_id
		smpp.NewFLParameter(uint8(len(shortMessage))),
		smpp.NewOctetStringFromString(shortMessage),
	}, []*smpp.Parameter{}), nil
}

func (broker *interactionBroker) promptForNextCommand() string {
	if broker.inputPromptStream != nil {
		broker.inputPromptStream.Write([]byte(broker.promptString))
	}

	input := broker.read()

	if broker.inputByteStream[len(input)-1] != byte('\n') {
		broker.writeLine("[ERROR] Command line too long.  Ignored.\n")
		broker.discardInputUntilNewline()

		return broker.promptForNextCommand()
	}

	return strings.TrimRight(string(input), "\n")
}

func (broker *interactionBroker) read() string {
	bytesRead, err := broker.inputReader.Read(broker.inputByteStream)
	broker.panicIfError(err)

	return string(broker.inputByteStream[:bytesRead])
}

func (broker *interactionBroker) write(outputString string) {
	_, err := broker.outputWriter.Write([]byte(outputString))
	broker.panicIfError(err)
}

func (broker *interactionBroker) writeLine(outputString string) {
	broker.write(outputString)
	broker.write("\n")
}

func (broker *interactionBroker) discardInputUntilNewline() {
	for input := broker.read(); input[len(input)-1] != byte('\n'); {
	}
}
func (broker *interactionBroker) notifyThatSmppPduWasReceived(pdu *smpp.PDU, nameOfReceivingEsme string, nameOfRemoteSender string) {
	switch pdu.CommandID {
	case smpp.CommandSubmitSm:
		broker.outputWriter.Write([]byte(fmt.Sprintf("(%s) received submit-sm from %s, SeqNum=%d, short_message=\"%s\"\n", nameOfReceivingEsme, nameOfRemoteSender, pdu.SequenceNumber, pdu.MandatoryParameters[17].Value.(string))))
	default:
		broker.outputWriter.Write([]byte(fmt.Sprintf("(%s) received %s from %s, SeqNum=%d\n", nameOfReceivingEsme, pdu.CommandName(), nameOfRemoteSender, pdu.SequenceNumber)))
	}
}

func (broker *interactionBroker) notifyThatBindWasCompletedWithPeer(nameOfBindingEsme string, nameOfBoundPeer string) {
	broker.outputWriter.Write([]byte(fmt.Sprintf("(%s) completed transceiver-bind with %s\n", nameOfBindingEsme, nameOfBoundPeer)))
}

func (broker *interactionBroker) notifyOfPduSendAttemptFromUnknownEsme(nameOfNonExistantEsme string) {
	broker.outputWriter.Write([]byte(fmt.Sprintf("[ERROR] Attempt to send message from unknown ESME named (%s)", nameOfNonExistantEsme)))
}

func (broker *interactionBroker) writeOutHelp() {
	helpText := `
$esme_name: send submit-sm to $smsc_name short_message="$message"
$esme_name: send enquire-link to $smsc_name
	`
	broker.outputWriter.Write([]byte(helpText))
}

func (broker *interactionBroker) panicIfError(err error) {
	if err != nil {
		panic(err)
	}
}
