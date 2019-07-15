package main

import (
	"fmt"
	"net"
	"smpp"
)

type messageDescriptor struct {
	sendFromEsmeNamed string
	sendToSmscNamed   string
	pdu               *smpp.PDU
}

type esmeEventType int

const (
	receivedMessage esmeEventType = iota
	completedBind
)

type esmeListenerEvent struct {
	Type                esmeEventType
	sourceEsme          *esme
	boundPeerName       string
	smppPDU             *smpp.PDU
	nameOfMessageSender string
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
	parentESME                                    *esme
}

func newEsmePeerMessageListener(nameOfPeer string, parentESME *esme, connectionToRemotePeer net.Conn) *esmePeerMessageListener {
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

func (connector *esmePeerMessageListener) startListeningForIncomingMessagesFromPeer(eventChannel chan<- *esmeListenerEvent) {
	for _, pdu := range connector.extraPDUsCollectedWhileWaitingForBindResponse {
		eventChannel <- &esmeListenerEvent{Type: receivedMessage, smppPDU: pdu, nameOfMessageSender: connector.nameOfRemotePeer, sourceEsme: connector.parentESME}
	}

	for {
		pdus, err := connector.streamReader.ExtractNextPDUs()
		connector.parentESME.panicIfError(err)

		for _, pdu := range pdus {
			eventChannel <- &esmeListenerEvent{Type: receivedMessage, smppPDU: pdu, nameOfMessageSender: connector.nameOfRemotePeer, sourceEsme: connector.parentESME}
		}
	}
}

type smsc struct {
	name string
	ip   net.IP
	port uint16
}

type esme struct {
	name                                        string
	ip                                          net.IP
	port                                        uint16
	peerBinds                                   []smppBindInfo
	connectionToPeerForPeerNamed                map[string]net.Conn
	channelForMessagesThisEsmeShouldSendToPeers chan *messageDescriptor
}

func newEsme(esmeName string, esmeIP net.IP, esmePort uint16) *esme {
	return &esme{name: esmeName, ip: esmeIP, port: esmePort, peerBinds: make([]smppBindInfo, 0, 10), connectionToPeerForPeerNamed: make(map[string]net.Conn)}
}

func (esme *esme) startListening(eventChannel chan<- *esmeListenerEvent) {
	for _, peerBind := range esme.peerBinds {
		conn, err := esme.connectTransportToPeer(peerBind.remoteIP, peerBind.remotePort)

		peerConnector := newEsmePeerMessageListener(peerBind.smscName, esme, conn)

		err = peerConnector.completeTransceiverBindingTowardPeer(peerBind.systemID, peerBind.systemType, peerBind.password)
		esme.panicIfError(err)

		eventChannel <- &esmeListenerEvent{Type: completedBind, sourceEsme: esme, boundPeerName: peerBind.smscName}

		esme.connectionToPeerForPeerNamed[peerBind.smscName] = conn

		go peerConnector.startListeningForIncomingMessagesFromPeer(eventChannel)
	}
}

func (esme *esme) panicIfError(err error) {
	if err != nil {
		panic(err)
	}
}

func (esme *esme) connectTransportToPeer(remoteIP net.IP, remotePort uint16) (net.Conn, error) {
	return net.Dial("tcp", fmt.Sprintf("%s:%d", remoteIP.String(), remotePort))
}

func (esme *esme) sendMessageToPeer(message *messageDescriptor) error {
	connectionToPeer := esme.connectionToPeerForPeerNamed[message.sendToSmscNamed]

	if connectionToPeer == nil {
		return fmt.Errorf("No such SMSC peer named (%s) is known to this ESME", message.sendToSmscNamed)
	}

	encodedPDU, err := message.pdu.Encode()

	if err != nil {
		return err
	}

	_, err = connectionToPeer.Write(encodedPDU)

	if err != nil {
		return fmt.Errorf("Error writing PDU to peer named (%s): %s", message.sendToSmscNamed, err)
	}

	return nil
}

func (esme *esme) outgoingMessageChannel() chan *messageDescriptor {
	if esme.channelForMessagesThisEsmeShouldSendToPeers == nil {
		esme.channelForMessagesThisEsmeShouldSendToPeers = make(chan *messageDescriptor)
	}

	return esme.channelForMessagesThisEsmeShouldSendToPeers
}
