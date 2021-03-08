package smppth

import (
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/blorticus/smpp"
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
	incomingPeerTransportListener             net.Listener
	isStopped                                 bool
}

// NewSMSC creates a new SMSC agent.
func NewSMSC(smscName string, smscBindSystemID string, listeningIP net.IP, listeningPort uint16) *SMSC {
	if smscBindSystemID == "" {
		smscBindSystemID = smscName
	}

	return &SMSC{
		name:                          smscName,
		ip:                            listeningIP,
		port:                          listeningPort,
		assertedSystemID:              smscBindSystemID,
		agentEventChannel:             nil,
		incomingPeerTransportListener: nil,
		isStopped:                     true,
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
	if smsc.sendTransportErrorEventAndStopAllWhenErrorDefined(err, "") {
		return
	}

	smsc.isStopped = false

	for smsc.isStopped == false {
		incomingTransport, err := listener.Accept()
		if smsc.sendTransportErrorEventAndStopAllWhenErrorDefined(err, "") {
			return
		}

		peerMessageHandler := newSmscPeerMessageHandler(smsc, incomingTransport)
		go peerMessageHandler.startHandlingPeerConnection()
	}
}

// StopAndUnbindAll instructs this SMSC agent to stop listening for incoming transport connections,
// and to both unbind all outstanding peer connections, and close their corresponding transports.
func (smsc *SMSC) StopAndUnbindAll() {
	smsc.isStopped = true

	if smsc.incomingPeerTransportListener != nil {
		if listenerCloseErr := smsc.incomingPeerTransportListener.Close(); listenerCloseErr != nil {
			smsc.sendEventIfChannelDefined(&AgentEvent{
				Type:           TransportError,
				SourceAgent:    smsc,
				RemotePeerName: "",
				SmppPDU:        nil,
				Error:          fmt.Errorf("Failed to close listener: %s", listenerCloseErr.Error()),
			})
		}
	}

	smsc.mapOfHandlerForRemotePeerByRemotePeerName.Range(func(peerName interface{}, peerHandler interface{}) bool {
		peerHandler.(*smscPeerMessageHandler).stop()
		return false
	})
}

func (smsc *SMSC) sendApplicationErrorEvent(err error, pduRelatedToErrorOrNilIfNone *smpp.PDU) {
	smsc.sendEventIfChannelDefined(&AgentEvent{
		Type:           ApplicationError,
		SourceAgent:    smsc,
		RemotePeerName: "",
		SmppPDU:        pduRelatedToErrorOrNilIfNone,
		Error:          err,
	})
}

func (smsc *SMSC) sendApplicationErrorEventWhenErrorDefined(err error, pduRelatedToErrorOrNilIfNone *smpp.PDU) bool {
	if err != nil {
		smsc.sendApplicationErrorEvent(err, pduRelatedToErrorOrNilIfNone)
		return true
	}

	return false
}

func (smsc *SMSC) sendTransportErrorEvent(err error, remotePeerName string) {
	if err == io.EOF {
		smsc.sendEventIfChannelDefined(&AgentEvent{
			Type:           PeerTransportClosed,
			SourceAgent:    smsc,
			RemotePeerName: remotePeerName,
			SmppPDU:        nil,
			Error:          err,
		})
	} else {
		smsc.sendEventIfChannelDefined(&AgentEvent{
			Type:           TransportError,
			SourceAgent:    smsc,
			RemotePeerName: remotePeerName,
			SmppPDU:        nil,
			Error:          err,
		})
	}
}

func (smsc *SMSC) sendTransportErrorEventAndStopAllWhenErrorDefined(err error, remotePeerName string) bool {
	if err != nil {
		smsc.sendTransportErrorEvent(err, remotePeerName)
		smsc.StopAndUnbindAll()
		return true
	}

	return false
}

func (smsc *SMSC) sendEventIfChannelDefined(event *AgentEvent) {
	if smsc.agentEventChannel != nil {
		smsc.agentEventChannel <- event
	}
}

func (smsc *SMSC) notifySmscOfThisHandlersPeerName(peerNameAssertedInBindRequest string, handler *smscPeerMessageHandler) {
	smsc.mapOfHandlerForRemotePeerByRemotePeerName.Store(peerNameAssertedInBindRequest, handler)
}

type smscPeerMessageHandler struct {
	connectionToPeer                     net.Conn
	streamReader                         *smpp.NetworkStreamReader
	parentSMSC                           *SMSC
	nameOfRemotePeer                     string
	nextGeneratedSmppRequestPduSeqNumber uint32
	stopChannel                          chan bool
}

func newSmscPeerMessageHandler(parentSmsc *SMSC, transportConnectionToPeer net.Conn) *smscPeerMessageHandler {
	return &smscPeerMessageHandler{
		connectionToPeer:                     transportConnectionToPeer,
		streamReader:                         smpp.NewNetworkStreamReader(transportConnectionToPeer),
		parentSMSC:                           parentSmsc,
		nameOfRemotePeer:                     "",
		nextGeneratedSmppRequestPduSeqNumber: 1,
		stopChannel:                          make(chan bool),
	}
}

type peerHandlerStreamReaderOutput struct {
	pdus []*smpp.PDU
	err  error
}

func (handler *smscPeerMessageHandler) startHandlingPeerConnection() {
	pdus, err := handler.streamReader.ExtractNextPDUs()
	if handler.parentSMSC.sendApplicationErrorEventWhenErrorDefined(err, nil) {
		return
	}

	if pdus[0].CommandID != smpp.CommandBindTransceiver {
		handler.parentSMSC.sendApplicationErrorEvent(fmt.Errorf("First PDU from peer (%s) should be bind-transceiver, but was (%s)", handler.connectionToPeer.RemoteAddr().String(), pdus[0].CommandName()), pdus[0])
		return
	}

	handler.nameOfRemotePeer = handler.extractPeerNameFromTransceiverBind(pdus[0])
	handler.parentSMSC.sendEventIfChannelDefined(&AgentEvent{
		RemotePeerName: handler.nameOfRemotePeer,
		SourceAgent:    handler.parentSMSC,
		Type:           ReceivedPDU,
		SmppPDU:        pdus[0],
	})

	bindResponsePDU, err := handler.sendTransceiverResponseToPeerBasedOnRequestBind(pdus[0])
	if handler.parentSMSC.sendTransportErrorEventAndStopAllWhenErrorDefined(err, handler.nameOfRemotePeer) {
		return
	}

	handler.parentSMSC.notifySmscOfThisHandlersPeerName(handler.nameOfRemotePeer, handler)

	handler.parentSMSC.sendEventIfChannelDefined(&AgentEvent{
		RemotePeerName: handler.nameOfRemotePeer,
		SourceAgent:    handler.parentSMSC,
		Type:           SentPDU,
		SmppPDU:        bindResponsePDU,
	})

	handler.parentSMSC.sendEventIfChannelDefined(&AgentEvent{
		RemotePeerName: handler.nameOfRemotePeer,
		SourceAgent:    handler.parentSMSC,
		Type:           CompletedBind,
		SmppPDU:        bindResponsePDU,
	})

	for i := 1; i < len(pdus); i++ {
		handler.parentSMSC.sendEventIfChannelDefined(&AgentEvent{
			Type:           ReceivedPDU,
			SmppPDU:        pdus[i],
			RemotePeerName: handler.nameOfRemotePeer,
			SourceAgent:    handler.parentSMSC,
		})
	}

	streamReaderReceiptChannel := make(chan *peerHandlerStreamReaderOutput)

	go func() {
		for {
			pdus, err := handler.streamReader.ExtractNextPDUs()
			streamReaderReceiptChannel <- &peerHandlerStreamReaderOutput{pdus, err}
		}
	}()

	for {
		select {
		case incomingStreamReaderResults := <-streamReaderReceiptChannel:
			if handler.parentSMSC.sendTransportErrorEventAndStopAllWhenErrorDefined(incomingStreamReaderResults.err, handler.nameOfRemotePeer) {
				return
			}

			for _, pdu := range incomingStreamReaderResults.pdus {
				handler.parentSMSC.sendEventIfChannelDefined(&AgentEvent{
					Type:           ReceivedPDU,
					SmppPDU:        pdu,
					RemotePeerName: handler.nameOfRemotePeer,
					SourceAgent:    handler.parentSMSC,
				})
			}

		case <-handler.stopChannel:
			if err := handler.connectionToPeer.Close(); err != nil {
				handler.parentSMSC.sendTransportErrorEvent(fmt.Errorf("On local connection close: %s", err), handler.nameOfRemotePeer)
			}

			return
		}
	}
}

func (handler *smscPeerMessageHandler) stop() {
	handler.stopChannel <- true
}

func (handler *smscPeerMessageHandler) extractPeerNameFromTransceiverBind(pdu *smpp.PDU) string {
	return pdu.MandatoryParameters[0].Value.(string)
}

func (handler *smscPeerMessageHandler) sendTransceiverResponseToPeerBasedOnRequestBind(bindTransceiverPdu *smpp.PDU) (bindResponsePDU *smpp.PDU, writeError error) {
	smscName := handler.makeNameShortEnoughForSmppSystemIDField(handler.parentSMSC.Name())

	bindResponsePDU = smpp.NewPDU(smpp.CommandBindTransceiverResp, 0, bindTransceiverPdu.SequenceNumber, []*smpp.Parameter{
		smpp.NewCOctetStringParameter(smscName),
	}, []*smpp.Parameter{})

	encodedBindResponse, _ := bindResponsePDU.Encode()

	if _, err := handler.connectionToPeer.Write(encodedBindResponse); err != nil {
		return nil, err
	}

	return bindResponsePDU, nil
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

	handler.parentSMSC.sendEventIfChannelDefined(&AgentEvent{
		Type:           SentPDU,
		SmppPDU:        pdu,
		RemotePeerName: handler.nameOfRemotePeer,
		SourceAgent:    handler.parentSMSC,
	})

	return nil
}

func (handler *smscPeerMessageHandler) resetSmppRequestPduSequenceNumberToLocalSequence(requestPdu *smpp.PDU) {
	requestPdu.SequenceNumber = handler.nextGeneratedSmppRequestPduSeqNumber
	handler.nextGeneratedSmppRequestPduSeqNumber++
}
