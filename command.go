package smppth

import (
	smpp "github.com/blorticus/smpp-go"
)

// UserCommandType is the type of user command provided in a UserCommand struct
type UserCommandType int

// Constants describing the type of user command instruction
const (
	SendPDU = iota
	Help
	Quit
)

// UserCommand represents a user instruction provided to an Agent in an AgentGroup.
// When Type is SendPDU, Details must be of type SendPduDetails.  When Type is Help,
// Details must by nil.
type UserCommand struct {
	Type    UserCommandType
	Details interface{}
}

// SendPduDetails provides a structured set of details for an Agent, instructing it
// to send an SMPP PDU to a particular destination.  StringParameterMap depends on
// the TypeOfSmppPDU.
type SendPduDetails struct {
	NameOfAgentThatWillSendPdu     string
	NameOfPeerThatShouldReceivePdu string
	TypeOfSmppPDU                  smpp.CommandIDType
	StringParametersMap            map[string]string
}
