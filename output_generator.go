package smppth

import (
	"fmt"

	smpp "github.com/blorticus/smpp-go"
)

// OutputGenerator is an inteface describing methods to generate standard responses to commands or
// events.  It can be implemented to change the responses used by standard test harness application
// components
type OutputGenerator interface {
	SayThatAPduWasReceivedByAnAgent(sendingAgentName string, receivingPeerName string, receivedPDU *smpp.PDU) string
	SayTheAPduWasSentByAnAgent(sendingAgentName string, receivingPeerName string, sentPDU *smpp.PDU) string
	SayTheATransceiverBindWasCompletedByAnAgent(esmePeerName string, smscPeerName string) string
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
	return fmt.Sprintf("%s received %s from %s", receivingPeerName, receivedPDU.CommandName(), sendingAgentName)
}

// SayTheAPduWasSentByAnAgent produces output "$agent_name sent $message_type to $peer_name"
func (generator *StandardOutputGenerator) SayTheAPduWasSentByAnAgent(sendingAgentName string, receivingPeerName string, sentPDU *smpp.PDU) string {
	return fmt.Sprintf("%s sent %s to %s", sendingAgentName, sentPDU.CommandName(), receivingPeerName)
}

// SayTheATransceiverBindWasCompletedByAnAgent produces output "transceiver bind completed between $esme_name and $smsc_name"
func (generator *StandardOutputGenerator) SayTheATransceiverBindWasCompletedByAnAgent(localAgentName string, remotePeerName string) string {
	return fmt.Sprintf("%s completed a transceiver bind with %s", localAgentName, remotePeerName)
}
