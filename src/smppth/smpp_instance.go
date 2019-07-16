package smppth

import (
	"fmt"
	"net"
	"smpp"
)

// A MessageDescriptor is provided to smpp agents, indicating what PDU to send, the name of
// the source from which to send, and the name of the destination to which the PDU should be
// sent
type MessageDescriptor struct {
	SendFromEsmeNamed string
	SendToSmscNamed   string
	PDU               *smpp.PDU
}

type esmeEventType int

const (
	// ReceivedMessage is the EsmeListenerEvent type when a message is received from a peer
	ReceivedMessage esmeEventType = iota
	// CompletedBind is the EsmeListenerEvent type when an agent completes a bind sequence with a peer
	CompletedBind
)

// EsmeListenerEvent is an event from an smpp agent.  If Type is 'completedBind', then 'sourceEsme' and
// 'boundPeerName' will be set.  If Type is 'receivedMessage', then 'sourceEsme', 'smppPDU' and
// 'nameOfMessageSender' will be set
type EsmeListenerEvent struct {
	Type                esmeEventType
	SourceEsme          *ESME
	BoundPeerName       string
	SmppPDU             *smpp.PDU
	NameOfMessageSender string
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

func (connector *esmePeerMessageListener) startListeningForIncomingMessagesFromPeer(eventChannel chan<- *EsmeListenerEvent) {
	for _, pdu := range connector.extraPDUsCollectedWhileWaitingForBindResponse {
		eventChannel <- &EsmeListenerEvent{Type: ReceivedMessage, SmppPDU: pdu, NameOfMessageSender: connector.nameOfRemotePeer, SourceEsme: connector.parentESME}
	}

	for {
		pdus, err := connector.streamReader.ExtractNextPDUs()
		connector.parentESME.panicIfError(err)

		for _, pdu := range pdus {
			eventChannel <- &EsmeListenerEvent{Type: ReceivedMessage, SmppPDU: pdu, NameOfMessageSender: connector.nameOfRemotePeer, SourceEsme: connector.parentESME}
		}
	}
}

// SMSC represents an SMPP 3.4 server, which accepts one or more transport connections and responds
// to bind requests
type SMSC struct {
	name string
	ip   net.IP
	port uint16
}

// ESME represents an SMPP 3.4 client, which initiates one or more transport connections and sends binds
// on those connections
type ESME struct {
	Name                                        string
	ip                                          net.IP
	port                                        uint16
	peerBinds                                   []smppBindInfo
	connectionToPeerForPeerNamed                map[string]net.Conn
	channelForMessagesThisEsmeShouldSendToPeers chan *MessageDescriptor
}

// NewEsme creates an SMPP 3.4 client with the given name, and using the given IP and port for outgoing
// transport connections
func NewEsme(esmeName string, esmeIP net.IP, esmePort uint16) *ESME {
	return &ESME{Name: esmeName, ip: esmeIP, port: esmePort, peerBinds: make([]smppBindInfo, 0, 10), connectionToPeerForPeerNamed: make(map[string]net.Conn)}
}

// StartListening begins the ESME activity loop.  The ESME emits outcome events on the provided
// channel.
func (esme *ESME) StartListening(eventChannel chan<- *EsmeListenerEvent) {
	for _, peerBind := range esme.peerBinds {
		conn, err := esme.connectTransportToPeer(peerBind.remoteIP, peerBind.remotePort)

		peerConnector := newEsmePeerMessageListener(peerBind.smscName, esme, conn)

		err = peerConnector.completeTransceiverBindingTowardPeer(peerBind.systemID, peerBind.systemType, peerBind.password)
		esme.panicIfError(err)

		eventChannel <- &EsmeListenerEvent{Type: CompletedBind, SourceEsme: esme, BoundPeerName: peerBind.smscName}

		esme.connectionToPeerForPeerNamed[peerBind.smscName] = conn

		go peerConnector.startListeningForIncomingMessagesFromPeer(eventChannel)
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

// SendMessageToPeer instructs the corresponding ESME to send a message to the remote peer identified
// in the MessageDescriptor
func (esme *ESME) SendMessageToPeer(message *MessageDescriptor) error {
	connectionToPeer := esme.connectionToPeerForPeerNamed[message.SendToSmscNamed]

	if connectionToPeer == nil {
		return fmt.Errorf("No such SMSC peer named (%s) is known to this ESME", message.SendToSmscNamed)
	}

	encodedPDU, err := message.PDU.Encode()

	if err != nil {
		return err
	}

	_, err = connectionToPeer.Write(encodedPDU)

	if err != nil {
		return fmt.Errorf("Error writing PDU to peer named (%s): %s", message.SendToSmscNamed, err)
	}

	return nil
}

// OutgoingMessageChannel retrieves the MessageDescriptor channel that is used by the ESME.
// MessageDescriptors that are written to this channel instruct the ESME to attempt to send
// a PDU to the remote peer named in the MessageDescriptor.
func (esme *ESME) OutgoingMessageChannel() chan *MessageDescriptor {
	if esme.channelForMessagesThisEsmeShouldSendToPeers == nil {
		esme.channelForMessagesThisEsmeShouldSendToPeers = make(chan *MessageDescriptor)
	}

	return esme.channelForMessagesThisEsmeShouldSendToPeers
}
