package smppth

import (
	"fmt"
	"regexp"
	"smpp"
)

// TextCommandProcessor connects to a reader, which accepts structured command messages, and a writer,
// which emits events on behalf of testharness agents that have been started.
type TextCommandProcessor struct {
	helpCommandMatcher           *regexp.Regexp
	sendCommandMatcher           *regexp.Regexp
	sendCommandParametersMatcher *regexp.Regexp
	emptyParameterMatcher        *regexp.Regexp
	emptyLastParameterMatcher    *regexp.Regexp
	doubleQuotedParameterMatcher *regexp.Regexp
	singleQuotedParameterMatcher *regexp.Regexp
	unquotedParameterMatcher     *regexp.Regexp
	lastSetOfMatchGroupValues    []string
}

// NewTextCommandProcessor creates an empty broker, where the prompt output stream is set to STDOUT,
// the command input stream is set to STDIN, and the event output writer is set to STDOUT.  Built-in
// outputs are not used by default.
func NewTextCommandProcessor() *TextCommandProcessor {
	return &TextCommandProcessor{
		helpCommandMatcher:           regexp.MustCompile(`^help$`),
		sendCommandMatcher:           regexp.MustCompile(`^(\S+?): send (\S+) to (\S+) *(.*)?$`),
		sendCommandParametersMatcher: regexp.MustCompile(`^ *short_message="(.+?)" *$`),
		emptyParameterMatcher:        regexp.MustCompile(`^(\S+)=\s+`),
		emptyLastParameterMatcher:    regexp.MustCompile(`^(\S+)=$`),
		doubleQuotedParameterMatcher: regexp.MustCompile(`^(\S+)="(.+?)"\s*`),
		singleQuotedParameterMatcher: regexp.MustCompile(`^(\S+)='(.+?)'\s*`),
		unquotedParameterMatcher:     regexp.MustCompile(`^(\S+)=(\S+)\s*`),
		lastSetOfMatchGroupValues:    []string{},
	}
}

func (processor *TextCommandProcessor) ConvertCommandLineStringToUserCommand(commandLine string) (*UserCommand, error) {
	processor.lastSetOfMatchGroupValues = []string{}

	if processor.thisIsTheHelpCommand(commandLine) {
		return &UserCommand{
			Type: Help,
		}, nil
	}

	if processor.thisIsASendCommand(commandLine) {
		smppCommandName := processor.lastSetOfMatchGroupValues[2]

		smppCommandID, aValidCommandName := smpp.CommandIDFromString(smppCommandName)

		if !aValidCommandName {
			return nil, fmt.Errorf("Invalid smpp PDU type name (%s)", smppCommandName)
		}

		return &UserCommand{
			Type: SendPDU,
			Details: &SendPduDetails{
				NameOfAgentThatWillSendPdu:     processor.lastSetOfMatchGroupValues[1],
				NameOfPeerThatShouldReceivePdu: processor.lastSetOfMatchGroupValues[3],
				TypeOfSmppPDU:                  smppCommandID,
				StringParametersMap:            processor.breakParametersIntoMap(processor.lastSetOfMatchGroupValues[4]),
			},
		}, nil
	}

	return nil, fmt.Errorf("Command not understood")
}

func (processor *TextCommandProcessor) CommandTextHelp() string {
	return `
$esme_name: send submit-sm to $smsc_name short_message="$message" dest_addr=$addr
$esme_name: send enquire-link to $smsc_name
`
}

func (processor *TextCommandProcessor) thisIsTheHelpCommand(commandLine string) bool {
	return processor.helpCommandMatcher.Match([]byte(commandLine))
}

func (processor *TextCommandProcessor) thisIsASendCommand(commandLine string) bool {
	submatches := processor.sendCommandMatcher.FindStringSubmatch(commandLine)

	if len(submatches) > 0 {
		processor.lastSetOfMatchGroupValues = submatches
		return true
	}

	return false
}

func (processor *TextCommandProcessor) breakParametersIntoMap(parameterString string) map[string]string {
	parameterMap := make(map[string]string)

	for len(parameterString) > 0 {
		foundSomeTypeOfMatch := false

		for _, compiledMatcher := range []*regexp.Regexp{processor.emptyLastParameterMatcher, processor.emptyParameterMatcher, processor.doubleQuotedParameterMatcher, processor.singleQuotedParameterMatcher, processor.unquotedParameterMatcher} {
			itDoesMatch, parameterName, parameterValue, parameterStringLength := processor.extractMappableValueAndMatchingLengthFromMatcher(compiledMatcher, parameterString)

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

func (processor *TextCommandProcessor) extractMappableValueAndMatchingLengthFromMatcher(compiledRegexp *regexp.Regexp, parseString string) (doesMatch bool, name string, value string, matchLen int) {
	groups := compiledRegexp.FindStringSubmatch(parseString)

	if groups == nil {
		return false, "", "", 0
	}

	if len(groups) == 2 {
		return true, groups[1], "", len(groups[0])
	}

	return true, groups[1], groups[2], len(groups[0])
}

// func (broker *TextCommandProcessor) CreatePduFromCommand(commandDetails *TextCommandProcessorSendCommand) (*smpp.PDU, error) {
// 	commandID, commandIDIsValid := smpp.CommandIDFromString(commandDetails.smppCommandTypeName)

// 	if !commandIDIsValid {
// 		return nil, fmt.Errorf("PDU command (%s) is not valid", commandDetails.smppCommandTypeName)
// 	}

// 	switch commandID {
// 	case smpp.CommandEnquireLink:
// 		return broker.attemptToMakeEnquireLinkPdu(commandDetails.commandParametersMap)

// 	case smpp.CommandSubmitSm:
// 		return broker.attemptToMakeSubmitSmPdu(commandDetails.commandParametersMap)
// 	default:
// 		return nil, fmt.Errorf("While (%s) is a valid command, I don't know how to generate a PDU of that type", commandDetails.smppCommandTypeName)
// 	}
// }

// func (broker *TextCommandProcessor) attemptToMakeEnquireLinkPdu(commandParameterMap map[string]string) (*smpp.PDU, error) {
// 	if len(commandParameterMap) > 0 {
// 		return nil, fmt.Errorf("When sending an enquire-link PDU, no additional parameters are allowed")
// 	}

// 	return smpp.NewPDU(smpp.CommandEnquireLink, 0, 1, []*smpp.Parameter{}, []*smpp.Parameter{}), nil
// }

// func (broker *TextCommandProcessor) attemptToMakeSubmitSmPdu(commandParameterMap map[string]string) (*smpp.PDU, error) {
// 	shortMessage, customShortMessageWasProvided := commandParameterMap["short_message"]
// 	destAddr, customDestAddrWasProvided := commandParameterMap["dest_addr"]
// 	destAddrNpi := uint8(0)

// 	if !customShortMessageWasProvided {
// 		shortMessage = "Sample Short Message"
// 	}

// 	if !customDestAddrWasProvided {
// 		destAddr = ""
// 	} else {
// 		destAddrNpi = uint8(9)
// 	}

// 	if len(shortMessage) > 255 {
// 		shortMessage = shortMessage[:255]
// 	}

// 	return smpp.NewPDU(smpp.CommandSubmitSm, 0, 1, []*smpp.Parameter{
// 		smpp.NewFLParameter(uint8(0)),           // service_type
// 		smpp.NewFLParameter(uint8(0)),           // source_addr_ton
// 		smpp.NewFLParameter(uint8(0)),           // source_addr_npi
// 		smpp.NewCOctetStringParameter(""),       // source_addr
// 		smpp.NewFLParameter(uint8(0)),           // dest_addr_ton
// 		smpp.NewFLParameter(uint8(destAddrNpi)), // dest_addr_npi
// 		smpp.NewCOctetStringParameter(destAddr), // destination_addr
// 		smpp.NewFLParameter(uint8(0)),           // esm_class
// 		smpp.NewFLParameter(uint8(0)),           // protocol_id
// 		smpp.NewFLParameter(uint8(0)),           // priority_flag
// 		smpp.NewFLParameter(uint8(0)),           // scheduled_delivery_time
// 		smpp.NewFLParameter(uint8(0)),           // validity_period
// 		smpp.NewFLParameter(uint8(0)),           // registered_delivery
// 		smpp.NewFLParameter(uint8(0)),           // replace_if_present_flag
// 		smpp.NewFLParameter(uint8(0)),           // data_coding
// 		smpp.NewFLParameter(uint8(0)),           // sm_defalt_msg_id
// 		smpp.NewFLParameter(uint8(len(shortMessage))),
// 		smpp.NewOctetStringFromString(shortMessage),
// 	}, []*smpp.Parameter{}), nil
// }

// NotifyThatSmppPduWasReceived instructs the broker to write an output event message indicating that an SMPP PDU was received
// from a peer.  The message type, sequence number, and -- depending on the message type -- other attributes are ouput
// func (broker *TextCommandProcessor) NotifyThatSmppPduWasReceived(pdu *smpp.PDU, nameOfReceivingEsme string, nameOfRemoteSender string) {
// 	switch pdu.CommandID {
// 	case smpp.CommandSubmitSm:
// 		shortMessageValueInterface := pdu.MandatoryParameters[17].Value
// 		shortMessageValue := shortMessageValueInterface.([]byte)
// 		broker.eventOutputTextChannel <- fmt.Sprintf("(%s) received submit-sm from %s, SeqNum=%d, short_message=\"%s\", dest_addr=(%s)\n", nameOfReceivingEsme, nameOfRemoteSender, pdu.SequenceNumber, shortMessageValue, pdu.MandatoryParameters[6].Value.(string))
// 	case smpp.CommandSubmitSmResp:
// 		broker.eventOutputTextChannel <- fmt.Sprintf("(%s) received submit-sm-resp from %s, SeqNum=%d, message_id=(%s)\n", nameOfReceivingEsme, nameOfRemoteSender, pdu.SequenceNumber, pdu.MandatoryParameters[0].Value.(string))
// 	default:
// 		broker.eventOutputTextChannel <- fmt.Sprintf("(%s) received %s from %s, SeqNum=%d\n", nameOfReceivingEsme, pdu.CommandName(), nameOfRemoteSender, pdu.SequenceNumber)
// 	}
// }

// // NotifyThatBindWasCompletedWithPeer instructs the broker to write an output event message indicating that a bind was completed
// // with a remote peer.  The name of the peer with which the bind was completed is included in the output.
// func (broker *TextCommandProcessor) NotifyThatBindWasCompletedWithPeer(nameOfBindingEsme string, nameOfBoundPeer string) {
// 	broker.eventOutputTextChannel <- fmt.Sprintf("(%s) completed transceiver-bind with %s\n", nameOfBindingEsme, nameOfBoundPeer)
// }

// // NotifyThatSmppPduWasSentToPeer instructs the broker to write an output event message indicating that a local agent successfully sent a message to
// // a remote peer
// func (broker *TextCommandProcessor) NotifyThatSmppPduWasSentToPeer(pduSentToPeer *smpp.PDU, nameOfSendingPeer string, nameOfReceivingPeer string) {
// 	broker.eventOutputTextChannel <- fmt.Sprintf("(%s) sent %s to %s, seqNum=(%d)\n", nameOfSendingPeer, pduSentToPeer.CommandName(), nameOfReceivingPeer, pduSentToPeer.SequenceNumber)
// }

// // NotifyThatErrorOccurredWhileTryingToSendMessage instructs the broker to write an output event message indicating an attempt to send a message
// // from an agent to a remote peer failed.
// func (broker *TextCommandProcessor) NotifyThatErrorOccurredWhileTryingToSendMessage(err error, pduThatAgentAttemptedToSend *smpp.PDU, nameOfSendingPeer string, nameOfReceivingPeer string) {
// 	broker.eventOutputTextChannel <- fmt.Sprintf("[ERROR] failed to send %s from %s to %s: %s\n", pduThatAgentAttemptedToSend.CommandName(), nameOfSendingPeer, nameOfReceivingPeer, err)
// }

// // WriteOutHelp instructs the broker to write an output event message listing the various possible input commands and their
// // parameters.
// func (broker *TextCommandProcessor) WriteOutHelp() {
// 	helpText := `
// $esme_name: send submit-sm to $smsc_name short_message="$message" dest_addr=$addr
// $esme_name: send enquire-link to $smsc_name
// `
// 	broker.outputWriter.Write([]byte(helpText))
// }

// func (broker *TextCommandProcessor) notifyThatUserProvidedCommandedIsInvalid(reason string) {
// 	broker.eventOutputTextChannel <- fmt.Sprintf("[ERROR] command not understand: %s", reason)
// }

// func (broker *TextCommandProcessor) panicIfError(err error) {
// 	if err != nil {
// 		panic(err)
// 	}
// }
