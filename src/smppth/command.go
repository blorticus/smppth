package smppth

import "smpp"

type UserCommandType int

const (
	SendPDU = iota
	Help
)

type UserCommand struct {
	Type    UserCommandType
	Details interface{}
}

type SendPduDetails struct {
	NameOfAgentThatWillSendPdu     string
	NameOfPeerThatShouldReceivePdu string
	TypeOfSmppPDU                  smpp.CommandIDType
	StringParametersMap            map[string]string
}
