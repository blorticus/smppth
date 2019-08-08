package smppth

import (
	"fmt"
	"smpp"
)

type UserCommandType int

const (
	SendMessage = iota
	Help
	InputError
)

type UserCommand struct {
	Command            UserCommandType
	SendCommandPDUType smpp.CommandIDType
	CommandParameters  map[string]string
	InputErrorMessage  string
}

// TextCommandProcessor connects to a reader, which accepts structured command messages, and a writer,
// which emits events on behalf of testharness agents that have been started.
type TextCommandProcessor struct {
	commandStringInputChannel  chan string
	commandStructOutputChannel chan *UserCommand
}

// NewTextCommandProcessor creates an empty broker, where the prompt output stream is set to STDOUT,
// the command input stream is set to STDIN, and the event output writer is set to STDOUT.  Built-in
// outputs are not used by default.
func NewTextCommandProcessor() *TextCommandProcessor {
	return &TextCommandProcessor{
		commandStringInputChannel:  make(chan string),
		commandStructOutputChannel: make(chan *UserCommand),
	}
}

// UserInteractionInputChannel returns a channel that accepts incoming user input command strings
func (processor *TextCommandProcessor) UserInteractionInputChannel() <-chan string {
	return processor.commandStringInputChannel
}

// ProcessedCommandOutputChannel returns a channel that contains structured command extracted
// from the input channel strings.
func (processor *TextCommandProcessor) ProcessedCommandOutputChannel() <-chan *UserCommand {
	return processor.commandStructOutputChannel
}

// BeginInteractiveSession instructs the broker to send the prompt to the prompt output stream,
// wait for a command on the command input stream, attempt to execute the command, and send any
// results to the event output stream.  This cycle (prompt, read, write) is repeated indefinitely.
func (processor *TextCommandProcessor) BeginInteractiveSession() {
	commandMatcher := newTextCommandMatcher()

	for {
		incomingUserCommandString := <-processor.commandStringInputChannel

		digestedCommand := commandMatcher.digestCommandString(incomingUserCommandString)

		if digestedCommand.isNotValid {
			processor.commandStructOutputChannel <- &UserCommand{
				Command:           InputError,
				InputErrorMessage: fmt.Sprintf("Invalid command: %s", digestedCommand.reasonCommandIsNotValid),
			}
		} else {
			
		}
	}
}

// func (broker *TextCommandProcessor) repeatedlyPromptTheUserForInput(userInputChannel chan<- *TextCommandProcessorValidUserInputCommand) {
// 	commandMatcher := newTextCommandProcessorCommandMatcher()

// 	for {
// 		nextCommandString := broker.promptForNextCommand()

// 		digestedCommand := commandMatcher.digestCommandString(nextCommandString)

// 		if digestedCommand.isNotValid {
// 			broker.notifyThatUserProvidedCommandedIsInvalid(digestedCommand.reasonCommandIsNotValid)
// 		} else {
// 			userInputChannel <- digestedCommand.compiledUserInputCommand
// 		}
// 	}
// }

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

// func (broker *TextCommandProcessor) promptForNextCommand() string {
// 	if broker.inputPromptStream != nil {
// 		broker.inputPromptStream.Write([]byte(broker.promptString))
// 	}

// 	input := broker.read()

// 	if broker.inputByteStream[len(input)-1] != byte('\n') {
// 		broker.writeLine("[ERROR] Command contains no newline or is too long.\n")
// 		broker.discardInputUntilNewline()

// 		return broker.promptForNextCommand()
// 	}

// 	return strings.TrimRight(string(input), "\n")
// }

// func (broker *TextCommandProcessor) read() string {
// 	bytesRead, err := broker.inputReader.Read(broker.inputByteStream)
// 	broker.panicIfError(err)

// 	return string(broker.inputByteStream[:bytesRead])
// }

// func (broker *TextCommandProcessor) write(outputString string) {
// 	_, err := broker.outputWriter.Write([]byte(outputString))
// 	broker.panicIfError(err)
// }

// func (broker *TextCommandProcessor) writeLine(outputString string) {
// 	broker.write(outputString)
// 	broker.write("\n")
// }

// func (broker *TextCommandProcessor) discardInputUntilNewline() {
// 	for input := broker.read(); input[len(input)-1] != byte('\n'); {
// 	}
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
