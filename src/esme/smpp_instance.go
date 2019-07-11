package main

import "net"

type smppBindInfo struct {
	smscName   string
	remoteIP   net.IP
	remotePort uint16
	systemID   string
	password   string
	systemType string
}

type esme struct {
	name      string
	ip        net.IP
	port      uint16
	peerBinds []smppBindInfo
}

type smsc struct {
	name string
	ip   net.IP
	port uint16
}

func (esme *esme) outgoingMessageChannel() chan<- *messageDescriptor {
	return nil
}

func (esme *esme) performTranceiverBindTo(bindInfo *smppBindInfo) error {
	return nil
}

func (esme *esme) startListening() {
}

func (esme *esme) sendMessageToPeer(message *messageDescriptor) {

}

func (esme *esme) incomingEventChannel() <-chan *esmeListenerEvent {
	return nil
}
