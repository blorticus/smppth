package smppth

import (
	smpp "github.com/blorticus/smpp-go"
)

// Agent is either a testharness agent, either an ESME or an SMSC
type Agent interface {
	Name() string
	StartEventLoop()
	SendMessageToPeer(message *MessageDescriptor) error
	SetAgentEventChannel(chan<- *AgentEvent)
}

// AgentEventType is an enum of the types of AgentEvents that can be raised.
type AgentEventType int

const (
	// ReceivedPDU is the AgentEvent type when an SMPP PDU is received from a peer.
	ReceivedPDU AgentEventType = iota
	// SentPDU is the AgentEvent type when the local Agent sent an SMPP PDU to a peer.
	SentPDU
	// CompletedBind is the AgentEvent type when an agent completes a bind sequence with a peer.
	CompletedBind
	// CompletedUnbind is the AgentEvent type after an SMSC receives an unbind and sends an unbind-resp,
	// or after an ESME sends an unbind and receives an ubind-resp
	CompletedUnbind
	// PeerTransportClosed is the AgentEvent type after the TCP transport toward a peer closes
	PeerTransportClosed
	// TransportError is the AgentEvent type when an Agent experiences some sort of error at the
	// transport layer
	TransportError
	// ApplicationError is the AgentEvent type when an Agent experiences some sort of error at the
	// SMPP layer
	ApplicationError
)

// AgentEvent is an event from an smpp agent.  SourceAgent is always the Agent that sourced
// this event.  RemotePeerName is always the name of the remote peer for the event.  For CompletedBind,
// the SmppPDU is the transceiver-bind-resp.  For CompletedUnbind, it is the unbind-resp.  For
// PeerTransportClosed, it is nil.  Error is always nil, except for the AgentEvents of type Error
// (e.g., TransportError).  For TransportError, RemotePeerName will be set to the remote
// peer name -- unless it is at connection setup, in which case, it will be "" -- and SmppPDU will be nil.
// For ApplicationError, RemotePeerName may be empty or may have a value, and SmppPDU may be nil or
// may be defined.  If SmppPDU is defined for ApplicationError, then the error relates to the PDU in some
// way.
type AgentEvent struct {
	Type           AgentEventType
	SourceAgent    Agent
	RemotePeerName string
	SmppPDU        *smpp.PDU
	Error          error
}

// A MessageDescriptor is provided to smpp agents, indicating what PDU to send, the name of
// the source from which to send, and the name of the destination to which the PDU should be
// sent
type MessageDescriptor struct {
	NameOfSendingPeer   string
	NameOfReceivingPeer string
	PDU                 *smpp.PDU
}
