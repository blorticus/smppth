package smppth

import (
	"smpp"
)

// Agent is either a testharness agent, either an ESME or an SMSC
type Agent interface {
	Name() string
	StartEventLoop(chan<- *AgentEvent)
	SendMessageToPeer(message *MessageDescriptor) error
}

// AgentEventType is an enum of the types of AgentEvents that can be raised.
type AgentEventType int

const (
	// ReceivedPDU is the AgentEvent type when an SMPP PDU is received from a peer
	ReceivedPDU AgentEventType = iota
	// SentPDU is the AgentEvent type when the local Agent sent an SMPP PDU to a peer
	SentPDU
	// CompletedBind is the AgentEvent type when an agent completes a bind sequence with a peer
	CompletedBind
)

// AgentEvent is an event from an smpp agent.
type AgentEvent struct {
	Type           AgentEventType
	SourceAgent    Agent
	RemotePeerName string
	SmppPDU        *smpp.PDU
}

// A MessageDescriptor is provided to smpp agents, indicating what PDU to send, the name of
// the source from which to send, and the name of the destination to which the PDU should be
// sent
type MessageDescriptor struct {
	NameOfSourcePeer string
	NameOfRemotePeer string
	PDU              *smpp.PDU
}
