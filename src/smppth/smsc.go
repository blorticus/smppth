package smppth

import (
	"fmt"
	"net"
	"smpp"
	"sync"
)

// SMSC represents an SMPP 3.4 server, which accepts one or more transport connections and responds
// to bind requests
type SMSC struct {
	name                                      string
	ip                                        net.IP
	port                                      uint16
	mapOfHandlerForRemotePeerByRemotePeerName sync.Map
	assertedSystemID                          string
	agentEventChannel                         chan<- *AgentEvent
}

// NewSMSC creates a new SMSC agent.
func NewSMSC(smscName string, smscBindSystemID string, listeningIP net.IP, listeningPort uint16) *SMSC {
	if smscBindSystemID == "" {
		smscBindSystemID = smscName
	}

	return &SMSC{
		name:              smscName,
		ip:                listeningIP,
		port:              listeningPort,
		assertedSystemID:  smscBindSystemID,
		agentEventChannel: nil,
	}
}

// Name returns the name of this SMSC agent instance
func (smsc *SMSC) Name() string {
	return smsc.name
}

// SetAgentEventChannel sets a channel to which this SMSC instance will write events
func (smsc *SMSC) SetAgentEventChannel(agentEventChannel chan<- *AgentEvent) {
	smsc.agentEventChannel = agentEventChannel
}

// SendMessageToPeer instructs this SMSC agent to send a message to the peer identified in the
// MessageDescriptor.  No effort is made to validate that the MessageDescriptor SourceAgentName
// matches this agent's name.
func (smsc *SMSC) SendMessageToPeer(message *MessageDescriptor) error {
	peerHandler, peerHandlerIsInMap := smsc.mapOfHandlerForRemotePeerByRemotePeerName.Load(message.NameOfReceivingPeer)

	if !peerHandlerIsInMap {
		return fmt.Errorf("This Agent is not bound to a peer named (%s)", message.NameOfReceivingPeer)
	}

	return peerHandler.(*smscPeerMessageHandler).sendSmppPduToPeer(message.PDU)
}

// StartEventLoop instructs this SMSC agent to start listening for incoming transport connections,
// to respond to binds, to emit AgentEvents to the agentEventChannel, and accept
// messages for remote delivery via SendMessageToPeer().
func (smsc *SMSC) StartEventLoop() {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", smsc.ip.String(), smsc.port))
	smsc.panicIfError(err)

	for {
		incomingTransport, err := listener.Accept()
		smsc.panicIfError(err)

		peerMessageHandler := newSmscPeerMessageHandler(smsc, incomingTransport)
		go peerMessageHandler.startHandlingPeerConnection()
	}
}

func (smsc *SMSC) sendEventIfChannelDefined(event *AgentEvent) {
	if smsc.agentEventChannel != nil {
		smsc.agentEventChannel <- event
	}
}

func (smsc *SMSC) notifySmscOfThisHandlersPeerName(peerNameAssertedInBindRequest string, handler *smscPeerMessageHandler) {
	smsc.mapOfHandlerForRemotePeerByRemotePeerName.Store(peerNameAssertedInBindRequest, handler)
}

func (smsc *SMSC) panicIfError(err error) {
	if err != nil {
		panic(err)
	}
}

type smscPeerMessageHandler struct {
	connectionToPeer                     net.Conn
	streamReader                         *smpp.NetworkStreamReader
	parentSMSC                           *SMSC
	nameOfRemotePeer                     string
	nextGeneratedSmppRequestPduSeqNumber uint32
}

func newSmscPeerMessageHandler(parentSmsc *SMSC, transportConnectionToPeer net.Conn) *smscPeerMessageHandler {
	return &smscPeerMessageHandler{
		connectionToPeer:                     transportConnectionToPeer,
		streamReader:                         smpp.NewNetworkStreamReader(transportConnectionToPeer),
		parentSMSC:                           parentSmsc,
		nameOfRemotePeer:                     "",
		nextGeneratedSmppRequestPduSeqNumber: 1,
	}
}

func (handler *smscPeerMessageHandler) startHandlingPeerConnection() {
	pdus, err := handler.streamReader.ExtractNextPDUs()
	handler.parentSMSC.panicIfError(err)

	if pdus[0].CommandID != smpp.CommandBindTransceiver {
		handler.parentSMSC.panicIfError(fmt.Errorf("First PDU from peer (%s) should be bind-transceiver, but was (%s)", handler.connectionToPeer.RemoteAddr().String(), pdus[0].CommandName()))
	}

	handler.nameOfRemotePeer = handler.extractPeerNameFromTransceiverBind(pdus[0])
	handler.parentSMSC.sendEventIfChannelDefined(&AgentEvent{RemotePeerName: handler.nameOfRemotePeer, SourceAgent: handler.parentSMSC, Type: ReceivedPDU, SmppPDU: pdus[0]})

	bindResponsePDU := handler.sendTransceiverResponseToPeerBasedOnRequestBind(pdus[0])
	handler.parentSMSC.notifySmscOfThisHandlersPeerName(handler.nameOfRemotePeer, handler)

	handler.parentSMSC.sendEventIfChannelDefined(&AgentEvent{RemotePeerName: handler.nameOfRemotePeer, SourceAgent: handler.parentSMSC, Type: CompletedBind, SmppPDU: bindResponsePDU})

	for i := 1; i < len(pdus); i++ {
		handler.parentSMSC.sendEventIfChannelDefined(&AgentEvent{Type: ReceivedPDU, SmppPDU: pdus[i], RemotePeerName: handler.nameOfRemotePeer, SourceAgent: handler.parentSMSC})
	}

	for {
		pdus, err := handler.streamReader.ExtractNextPDUs()
		handler.parentSMSC.panicIfError(err)

		for _, pdu := range pdus {
			handler.parentSMSC.sendEventIfChannelDefined(&AgentEvent{Type: ReceivedPDU, SmppPDU: pdu, RemotePeerName: handler.nameOfRemotePeer, SourceAgent: handler.parentSMSC})
		}
	}
}

func (handler *smscPeerMessageHandler) extractPeerNameFromTransceiverBind(pdu *smpp.PDU) string {
	return pdu.MandatoryParameters[0].Value.(string)
}

func (handler *smscPeerMessageHandler) sendTransceiverResponseToPeerBasedOnRequestBind(bindTransceiverPdu *smpp.PDU) (bindResponsePDU *smpp.PDU) {
	smscName := handler.makeNameShortEnoughForSmppSystemIDField(handler.parentSMSC.Name())

	bindResponsePDU = smpp.NewPDU(smpp.CommandBindTransceiverResp, 0, bindTransceiverPdu.SequenceNumber, []*smpp.Parameter{
		smpp.NewCOctetStringParameter(smscName),
	}, []*smpp.Parameter{})

	encodedBindResponse, _ := bindResponsePDU.Encode()

	_, err := handler.connectionToPeer.Write(encodedBindResponse)
	handler.parentSMSC.panicIfError(err)

	return bindResponsePDU
}

func (handler *smscPeerMessageHandler) makeNameShortEnoughForSmppSystemIDField(name string) string {
	if len(name) > 16 {
		return name[:16]
	}

	return name
}

func (handler *smscPeerMessageHandler) sendSmppPduToPeer(pdu *smpp.PDU) error {
	if pdu.IsRequest() {
		handler.resetSmppRequestPduSequenceNumberToLocalSequence(pdu)
	}

	encodedPDU, err := pdu.Encode()
	if err != nil {
		return err
	}

	_, err = handler.connectionToPeer.Write(encodedPDU)
	if err != nil {
		return err
	}

	return nil
}

func (handler *smscPeerMessageHandler) resetSmppRequestPduSequenceNumberToLocalSequence(requestPdu *smpp.PDU) {
	requestPdu.SequenceNumber = handler.nextGeneratedSmppRequestPduSeqNumber
	handler.nextGeneratedSmppRequestPduSeqNumber++
}
