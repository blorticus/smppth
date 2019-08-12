package smppth

import "smpp"

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
	return ""
}

func (generator *StandardOutputGenerator) sayTheAPduWasSentByAnAgent(sendingAgentName string, receivingPeerName string, sentPDU *smpp.PDU) string {
	return ""
}

func (generator *StandardOutputGenerator) sayTheATransceiverBindWasCompletedByAnAgent(esmePeerName string, smscPeerName string) string {
	return ""
}
