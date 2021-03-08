package smppth

import (
	"fmt"

	"github.com/blorticus/smpp"
)

// OutputGenerator is an inteface describing methods to generate standard responses to commands or
// events.  It can be implemented to change the responses used by standard test harness application
// components
type OutputGenerator interface {
	SayThatAPduWasReceivedByAnAgent(sendingAgentName string, receivingPeerName string, receivedPDU *smpp.PDU) string
	SayThatAPduWasSentByAnAgent(sendingAgentName string, receivingPeerName string, sentPDU *smpp.PDU) string
	SayThatATransceiverBindWasCompletedByAnAgent(localAgentName string, remotePeerName string) string
	SayThatTheTransportForAPeerClosed(localAgentName string, remotePeerName string) string
	SayThatATransportErrorWasThrown(localAgentName string, remotePeerName string, err error) string
	SayThatAnApplicationErrorWasThrown(reportingAgentName string, err error) string
}

// StandardOutputGenerator implements OutputGenerator, providing generic text responses for commands and events
type StandardOutputGenerator struct {
}

// NewStandardOutputGenerator creates a new StandardOutputGenerator
func NewStandardOutputGenerator() *StandardOutputGenerator {
	return &StandardOutputGenerator{}
}

// SayThatAPduWasReceivedByAnAgent produces output "$peer_name received $message_type from $agent_name"
func (generator *StandardOutputGenerator) SayThatAPduWasReceivedByAnAgent(sendingAgentName string, receivingPeerName string, receivedPDU *smpp.PDU) string {
	switch receivedPDU.CommandID {
	case smpp.CommandSubmitSm:
		returnString := fmt.Sprintf("%s received submit-sm from %s", receivingPeerName, sendingAgentName)

		if receivedPDU.MandatoryParameters[6].Value.(string) != "" {
			returnString = fmt.Sprintf("%s, dest_addr=(%s)", returnString, receivedPDU.MandatoryParameters[6].Value.(string))
		}

		return fmt.Sprintf("%s, short_message=(%s)", returnString, string(receivedPDU.MandatoryParameters[17].Value.([]byte)))

	case smpp.CommandSubmitSmResp:
		return fmt.Sprintf("%s received submit-sm-resp from %s, message_id=(%s)",
			receivingPeerName,
			sendingAgentName,
			receivedPDU.MandatoryParameters[0].Value.(string),
		)

	default:
		return fmt.Sprintf("%s received %s from %s", receivingPeerName, receivedPDU.CommandName(), sendingAgentName)

	}
}

// SayThatAPduWasSentByAnAgent produces output "$sendingAgentName sent $message_type to $receivingPeerName"
func (generator *StandardOutputGenerator) SayThatAPduWasSentByAnAgent(sendingAgentName string, receivingPeerName string, sentPDU *smpp.PDU) string {
	return fmt.Sprintf("%s sent %s to %s", sendingAgentName, sentPDU.CommandName(), receivingPeerName)
}

// SayThatATransceiverBindWasCompletedByAnAgent produces output "$localAgentName completed a transceiver bind with $remotePeerName"
func (generator *StandardOutputGenerator) SayThatATransceiverBindWasCompletedByAnAgent(localAgentName string, remotePeerName string) string {
	return fmt.Sprintf("%s completed a transceiver bind with %s", localAgentName, remotePeerName)
}

// SayThatTheTransportForAPeerClosed produces output "$localAgentName peer connection closed from $remotePeerName"
func (generator *StandardOutputGenerator) SayThatTheTransportForAPeerClosed(localAgentName string, remotePeerName string) string {
	return fmt.Sprintf("%s peer connection closed from %s", localAgentName, remotePeerName)
}

// SayThatATransportErrorWasThrown produces output "$localAgentName received error on transport with $remotePeerName: $errString"
func (generator *StandardOutputGenerator) SayThatATransportErrorWasThrown(localAgentName string, remotePeerName string, err error) string {
	return fmt.Sprintf("%s received error on transport with %s: %s", localAgentName, remotePeerName, err)
}

// SayThatAnApplicationErrorWasThrown produces output "$reportingAgentName reports an application error: $errString"
func (generator *StandardOutputGenerator) SayThatAnApplicationErrorWasThrown(reportingAgentName string, err error) string {
	return fmt.Sprintf("%s reports an application error: %s", reportingAgentName, err)
}
