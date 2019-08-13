package smppth

import (
	"fmt"
	"smpp"
)

type OutputGenerator interface {
	sayThatAPduWasReceivedByAnAgent(sendingAgentName string, receivingPeerName string, receivedPDU *smpp.PDU) string
	sayTheAPduWasSentByAnAgent(sendingAgentName string, receivingPeerName string, sentPDU *smpp.PDU) string
	sayTheATransceiverBindWasCompletedByAnAgent(esmePeerName string, smscPeerName string) string
}

type StandardOutputGenerator struct {
}

func NewStandardOutputGenerator() *StandardOutputGenerator {
	return &StandardOutputGenerator{}
}

func (generator *StandardOutputGenerator) sayThatAPduWasReceivedByAnAgent(sendingAgentName string, receivingPeerName string, receivedPDU *smpp.PDU) string {
	return fmt.Sprintf("%s received %s from %s", receivingPeerName, receivedPDU.CommandName(), sendingAgentName)
}

func (generator *StandardOutputGenerator) sayTheAPduWasSentByAnAgent(sendingAgentName string, receivingPeerName string, sentPDU *smpp.PDU) string {
	return fmt.Sprintf("%s sent %s to %s", sendingAgentName, sentPDU.CommandName(), receivingPeerName)
}

func (generator *StandardOutputGenerator) sayTheATransceiverBindWasCompletedByAnAgent(esmePeerName string, smscPeerName string) string {
	return fmt.Sprintf("transceiver bind completed between %s and %s", esmePeerName, smscPeerName)
}
