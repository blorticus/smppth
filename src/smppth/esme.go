package smppth

import (
	"fmt"
	"net"
	"smpp"
)

// ESME represents an SMPP 3.4 client, which initiates one or more transport connections and sends binds
// on those connections
type ESME struct {
	name                             string
	ip                               net.IP
	port                             uint16
	peerBinds                        []smppBindInfo
	connectionToPeerForPeerNamed     map[string]net.Conn
	channelOfEventsRaisedByThisAgent chan *AgentEvent
}

// NewEsme creates an SMPP 3.4 client with the given name, and using the given IP and port for outgoing
// transport connections
func NewEsme(esmeName string, esmeIP net.IP, esmePort uint16) *ESME {
	return &ESME{
		name:                             esmeName,
		ip:                               esmeIP,
		port:                             esmePort,
		peerBinds:                        make([]smppBindInfo, 0, 10),
		connectionToPeerForPeerNamed:     make(map[string]net.Conn),
		channelOfEventsRaisedByThisAgent: nil,
	}
}

// Name returns the name of this ESME agent
func (esme *ESME) Name() string {
	return esme.name
}

// SendMessageToPeer instructs this ESME agent to send a message to the peer identified in the
// MessageDescriptor.  No effort is made to validate that the MessageDescriptor SourceAgentName
// matches this agent's name.
func (esme *ESME) SendMessageToPeer(message *MessageDescriptor) error {
	connectionToPeer := esme.connectionToPeerForPeerNamed[message.NameOfRemotePeer]

	if connectionToPeer == nil {
		return fmt.Errorf("No such SMSC peer named (%s) is known to this ESME", message.NameOfRemotePeer)
	}

	encodedPDU, err := message.PDU.Encode()

	if err != nil {
		return err
	}

	_, err = connectionToPeer.Write(encodedPDU)

	if err != nil {
		return fmt.Errorf("Error writing PDU to peer named (%s): %s", message.NameOfRemotePeer, err)
	}

	return nil
}

// StartEventLoop instructs this ESME agent to start listening for incoming transport connections,
// to respond to binds, to emit AgentEvents to the agentEventChannel, and accept
// messages for remote delivery via SendMessagesToPeer().
func (esme *ESME) StartEventLoop(agentEventChannel chan<- *AgentEvent) {
	for _, peerBind := range esme.peerBinds {
		conn, err := esme.connectTransportToPeer(peerBind.remoteIP, peerBind.remotePort)

		peerConnector := newEsmePeerMessageListener(peerBind.smscName, esme, conn)

		err = peerConnector.completeTransceiverBindingTowardPeer(peerBind.systemID, peerBind.systemType, peerBind.password)
		esme.panicIfError(err)

		agentEventChannel <- &AgentEvent{Type: CompletedBind, SourceAgent: esme, RemotePeerName: peerBind.smscName}

		esme.connectionToPeerForPeerNamed[peerBind.smscName] = conn

		go peerConnector.startListeningForIncomingMessagesFromPeer(agentEventChannel)
	}
}

func (esme *ESME) panicIfError(err error) {
	if err != nil {
		panic(err)
	}
}

func (esme *ESME) connectTransportToPeer(remoteIP net.IP, remotePort uint16) (net.Conn, error) {
	return net.Dial("tcp", fmt.Sprintf("%s:%d", remoteIP.String(), remotePort))
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
}

func newEsmePeerMessageListener(nameOfPeer string, parentESME *ESME, connectionToRemotePeer net.Conn) *esmePeerMessageListener {
	return &esmePeerMessageListener{nameOfRemotePeer: nameOfPeer, parentESME: parentESME, peerConnection: connectionToRemotePeer, streamReader: smpp.NewNetworkStreamReader(connectionToRemotePeer)}
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

	pdus, err := connector.streamReader.ExtractNextPDUs()

	if err != nil {
		return err
	}

	if pdus[0].CommandID != smpp.CommandBindTransceiverResp {
		return fmt.Errorf("Expected transceiver_bind_resp but received %s", pdus[0].CommandName())
	}

	if len(pdus) > 1 {
		connector.extraPDUsCollectedWhileWaitingForBindResponse = pdus[1:]
	}

	return nil
}

func (connector *esmePeerMessageListener) startListeningForIncomingMessagesFromPeer(eventChannel chan<- *AgentEvent) {
	for _, pdu := range connector.extraPDUsCollectedWhileWaitingForBindResponse {
		eventChannel <- &AgentEvent{Type: ReceivedMessage, SmppPDU: pdu, RemotePeerName: connector.nameOfRemotePeer, SourceAgent: connector.parentESME}
	}

	for {
		pdus, err := connector.streamReader.ExtractNextPDUs()
		connector.parentESME.panicIfError(err)

		for _, pdu := range pdus {
			eventChannel <- &AgentEvent{Type: ReceivedMessage, SmppPDU: pdu, RemotePeerName: connector.nameOfRemotePeer, SourceAgent: connector.parentESME}
		}
	}
}
