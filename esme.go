package smppth

import (
	"fmt"
	"io"
	"net"
	"syscall"

	"github.com/blorticus/smpp"
	"golang.org/x/sys/unix"
)

// ESME represents an SMPP 3.4 client, which initiates one or more transport connections and sends binds
// on those connections
type ESME struct {
	name                                        string
	ip                                          net.IP
	port                                        uint16
	peerBinds                                   []smppBindInfo
	mapOfConnectorForRemotePeerByRemotePeerName map[string]*esmePeerMessageListener
	agentEventChannel                           chan<- *AgentEvent
}

// NewEsme creates an SMPP 3.4 client with the given name, and using the given IP and port for outgoing
// transport connections
func NewEsme(esmeName string, esmeIP net.IP, esmePort uint16) *ESME {
	return &ESME{
		name:      esmeName,
		ip:        esmeIP,
		port:      esmePort,
		peerBinds: make([]smppBindInfo, 0, 10),
		mapOfConnectorForRemotePeerByRemotePeerName: make(map[string]*esmePeerMessageListener),
		agentEventChannel: nil,
	}
}

// SetAgentEventChannel sets a channel to which this ESME instance will write events
func (esme *ESME) SetAgentEventChannel(agentEventChannel chan<- *AgentEvent) {
	esme.agentEventChannel = agentEventChannel
}

// Name returns the name of this ESME agent
func (esme *ESME) Name() string {
	return esme.name
}

// SendMessageToPeer instructs this ESME agent to send a message to the peer identified in the
// MessageDescriptor.  No effort is made to validate that the MessageDescriptor SourceAgentName
// matches this agent's name.
func (esme *ESME) SendMessageToPeer(message *MessageDescriptor) error {
	connector := esme.mapOfConnectorForRemotePeerByRemotePeerName[message.NameOfReceivingPeer]

	if connector == nil {
		return fmt.Errorf("No such SMSC peer named (%s) is known to this ESME", message.NameOfReceivingPeer)
	}

	return connector.sendSmppPduToPeer(message.PDU)
}

// StartEventLoop instructs this ESME agent to start listening for incoming transport connections,
// to respond to binds, to emit AgentEvents to the agentEventChannel, and accept
// messages for remote delivery via SendMessagesToPeer().
func (esme *ESME) StartEventLoop() {
	for _, peerBind := range esme.peerBinds {
		conn, err := esme.connectTransportToPeer(peerBind.remoteIP, peerBind.remotePort)

		peerConnector := newEsmePeerMessageListener(peerBind.smscName, esme, conn)

		if err = peerConnector.completeTransceiverBindingTowardPeer(peerBind.systemID, peerBind.systemType, peerBind.password); err != nil {
			esme.sendApplicationErrorEvent(err, nil)
			return
		}

		esme.mapOfConnectorForRemotePeerByRemotePeerName[peerBind.smscName] = peerConnector

		go peerConnector.startListeningForIncomingMessagesFromPeer()
	}
}

// UnbindAll instructs this ESME agent to both unbind all outstanding peer
// connections, and close their corresponding transports.
func (esme *ESME) UnbindAll() {
	for _, peerHandler := range esme.mapOfConnectorForRemotePeerByRemotePeerName {
		peerHandler.stop()
	}
}

func (esme *ESME) sendApplicationErrorEvent(err error, pduRelatedToErrorOrNilIfNone *smpp.PDU) {
	esme.sendEventIfChannelDefined(&AgentEvent{
		Type:           ApplicationError,
		SourceAgent:    esme,
		RemotePeerName: "",
		SmppPDU:        pduRelatedToErrorOrNilIfNone,
		Error:          err,
	})
}

func (esme *ESME) sendApplicationErrorEventWhenErrorDefined(err error, pduRelatedToErrorOrNilIfNone *smpp.PDU) bool {
	if err != nil {
		esme.sendApplicationErrorEvent(err, pduRelatedToErrorOrNilIfNone)
		return true
	}

	return false
}

func (esme *ESME) sendTransportErrorEvent(err error, remotePeerName string) {
	if err == io.EOF {
		esme.sendEventIfChannelDefined(&AgentEvent{
			Type:           PeerTransportClosed,
			SourceAgent:    esme,
			RemotePeerName: remotePeerName,
			SmppPDU:        nil,
			Error:          err,
		})
	} else {
		esme.sendEventIfChannelDefined(&AgentEvent{
			Type:           TransportError,
			SourceAgent:    esme,
			RemotePeerName: remotePeerName,
			SmppPDU:        nil,
			Error:          err,
		})
	}
}

func (esme *ESME) sendTransportErrorEventAndStopAllWhenErrorDefined(err error, remotePeerName string) bool {
	if err != nil {
		esme.sendTransportErrorEvent(err, remotePeerName)
		esme.UnbindAll()
		return true
	}

	return false
}

func (esme *ESME) sendEventIfChannelDefined(event *AgentEvent) {
	if esme.agentEventChannel != nil {
		esme.agentEventChannel <- event
	}
}

func dialControlFunctionToSetReuse(network, address string, c syscall.RawConn) error {
	var err error
	c.Control(func(fd uintptr) {
		err = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEADDR, 1)
		if err != nil {
			return
		}

		err = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)
		if err != nil {
			return
		}
	})
	return err
}

func (esme *ESME) connectTransportToPeer(remoteIP net.IP, remotePort uint16) (net.Conn, error) {
	laddr, _ := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", esme.ip.String(), esme.port))

	d := net.Dialer{
		Control:   dialControlFunctionToSetReuse,
		LocalAddr: laddr,
	}

	return d.Dial("tcp", fmt.Sprintf("%s:%d", remoteIP.String(), remotePort))
}

type smppBindInfo struct {
	smscName   string
	remoteIP   net.IP
	remotePort uint16
	systemID   string
	password   string
	systemType string
}

type esmePeerMessageListener struct {
	streamReader                                  *smpp.NetworkStreamReader
	peerConnection                                net.Conn
	extraPDUsCollectedWhileWaitingForBindResponse []*smpp.PDU
	nameOfRemotePeer                              string
	parentESME                                    *ESME
	nextGeneratedSmppRequestPduSeqNumber          uint32
	stopChannel                                   chan bool
}

func newEsmePeerMessageListener(nameOfPeer string, parentESME *ESME, connectionToRemotePeer net.Conn) *esmePeerMessageListener {
	return &esmePeerMessageListener{
		nameOfRemotePeer:                     nameOfPeer,
		parentESME:                           parentESME,
		peerConnection:                       connectionToRemotePeer,
		streamReader:                         smpp.NewNetworkStreamReader(connectionToRemotePeer),
		nextGeneratedSmppRequestPduSeqNumber: 1,
		stopChannel:                          make(chan bool),
	}
}

func (connector *esmePeerMessageListener) completeTransceiverBindingTowardPeer(esmeSystemID string, esmeSystemType string, bindPassword string) error {
	bindPDU := smpp.NewPDU(smpp.CommandBindTransceiver, 0, 1, []*smpp.Parameter{
		smpp.NewCOctetStringParameter(esmeSystemID),
		smpp.NewCOctetStringParameter(bindPassword),
		smpp.NewCOctetStringParameter(esmeSystemType),
		smpp.NewFLParameter(uint8(0x34)),
		smpp.NewFLParameter(uint8(0)),
		smpp.NewFLParameter(uint8(0)),
		smpp.NewOctetStringFromString(""),
	}, []*smpp.Parameter{})

	encodedPDU, _ := bindPDU.Encode()
	_, err := connector.peerConnection.Write(encodedPDU)

	if err != nil {
		return err
	}

	connector.parentESME.sendEventIfChannelDefined(&AgentEvent{
		Type:           SentPDU,
		SourceAgent:    connector.parentESME,
		RemotePeerName: connector.nameOfRemotePeer,
		SmppPDU:        bindPDU,
	})

	pdus, err := connector.streamReader.ExtractNextPDUs()

	if err != nil {
		return err
	}

	connector.parentESME.sendEventIfChannelDefined(&AgentEvent{
		Type:           ReceivedPDU,
		SourceAgent:    connector.parentESME,
		RemotePeerName: connector.nameOfRemotePeer,
		SmppPDU:        pdus[0],
	})

	if pdus[0].CommandID != smpp.CommandBindTransceiverResp {
		return fmt.Errorf("Expected transceiver_bind_resp but received %s", pdus[0].CommandName())
	}

	if len(pdus) > 1 {
		connector.extraPDUsCollectedWhileWaitingForBindResponse = pdus[1:]
	}

	connector.parentESME.sendEventIfChannelDefined(&AgentEvent{
		Type:           CompletedBind,
		SourceAgent:    connector.parentESME,
		RemotePeerName: connector.nameOfRemotePeer,
		SmppPDU:        pdus[0],
	})

	return nil
}

type peerMessageListenerStreamReaderOutput struct {
	pdus []*smpp.PDU
	err  error
}

func (connector *esmePeerMessageListener) startListeningForIncomingMessagesFromPeer() {
	for _, pdu := range connector.extraPDUsCollectedWhileWaitingForBindResponse {
		connector.parentESME.sendEventIfChannelDefined(&AgentEvent{
			Type:           ReceivedPDU,
			SmppPDU:        pdu,
			RemotePeerName: connector.nameOfRemotePeer,
			SourceAgent:    connector.parentESME,
		})
	}

	streamReaderReceiptChannel := make(chan *peerMessageListenerStreamReaderOutput)

	go func() {
		for {
			pdus, err := connector.streamReader.ExtractNextPDUs()
			streamReaderReceiptChannel <- &peerMessageListenerStreamReaderOutput{pdus, err}
		}
	}()

	for {
		select {
		case incomingStreamReaderResults := <-streamReaderReceiptChannel:
			if connector.parentESME.sendTransportErrorEventAndStopAllWhenErrorDefined(incomingStreamReaderResults.err, connector.nameOfRemotePeer) {
				return
			}

			for _, pdu := range incomingStreamReaderResults.pdus {
				connector.parentESME.sendEventIfChannelDefined(&AgentEvent{
					Type:           ReceivedPDU,
					SmppPDU:        pdu,
					RemotePeerName: connector.nameOfRemotePeer,
					SourceAgent:    connector.parentESME,
				})
			}

		case <-connector.stopChannel:
			if err := connector.peerConnection.Close(); err != nil {
				connector.parentESME.sendTransportErrorEvent(fmt.Errorf("On local connection close: %s", err), connector.nameOfRemotePeer)
			}

			return
		}
	}
}

func (connector *esmePeerMessageListener) sendSmppPduToPeer(pdu *smpp.PDU) error {
	if pdu.IsRequest() {
		connector.resetSmppRequestPduSequenceNumberToLocalSequence(pdu)
	}

	encodedPDU, err := pdu.Encode()
	if err != nil {
		return err
	}

	_, err = connector.peerConnection.Write(encodedPDU)
	if err != nil {
		return err
	}

	connector.parentESME.sendEventIfChannelDefined(&AgentEvent{
		Type:           SentPDU,
		SourceAgent:    connector.parentESME,
		RemotePeerName: connector.nameOfRemotePeer,
		SmppPDU:        pdu,
	})

	return nil
}

func (connector *esmePeerMessageListener) resetSmppRequestPduSequenceNumberToLocalSequence(requestPdu *smpp.PDU) {
	requestPdu.SequenceNumber = connector.nextGeneratedSmppRequestPduSeqNumber
	connector.nextGeneratedSmppRequestPduSeqNumber++
}

func (connector *esmePeerMessageListener) stop() {
	connector.stopChannel <- true
}
