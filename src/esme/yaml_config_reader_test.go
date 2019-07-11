package main

import (
	"fmt"
	"net"
	"strings"
	"testing"
)

// TestParseIoReader tests applicationConfigYamlReader.parseIoReader()
func TestParseIoReader(t *testing.T) {
	ioReader := strings.NewReader(`
---
SMSCs:
  - Name: smsc-cluster01-vs
    IP: 192.168.1.1
    Port: 2775
    BindPassword: passwd1
  - Name: smsc-cluster02-vs
    IP: 192.168.1.2
    Port: 2775
    BindPassword: passwd2
ESMEs:
  - Name: esme-rcs01-tp01
    IP: 10.1.1.1
    Port: 2775
    BindSystemID: rcs01-tp01
    BindSystemType: rcs
  - Name: esme-rcs01-tp02
    IP: 10.1.1.2
    Port: 2775
    BindSystemID: rcs01-tp02
    BindSystemType: rcs
TransceiverBinds:
  - ESME: esme-rcs01-tp01
    SMSC: smsc-cluster01-vs
  - ESME: esme-rcs01-tp01
    SMSC: smsc-cluster02-vs
  - ESME: esme-rcs01-tp02
    SMSC: smsc-cluster01-vs
  - ESME: esme-rcs01-tp02
    SMSC: smsc-cluster02-vs	
`)

	yamlReader := newApplicationConfigYamlReader()

	esmeList, smscList, err := yamlReader.parseReader(ioReader)

	if err != nil {
		t.Errorf("[sample-config.yaml]: on parseReader() received error: %s", err)
	}

	if len(esmeList) != 2 {
		t.Errorf("[sample-config.yaml]: on parseReader() expected esmeList length 2, actual length is %d", len(esmeList))
	}

	if len(smscList) != 2 {
		t.Errorf("[sample-config.yaml]: on parseReader() expected smscList length 2, actual length is %d", len(smscList))
	}

	if ok, err := compareEsme(esmeList[0],
		&esme{
			name: "esme-rcs01-tp01",
			ip:   net.ParseIP("10.1.1.1"),
			port: uint16(2775),
			peerBinds: []smppBindInfo{
				smppBindInfo{
					smscName:   "smsc-cluster01-vs",
					remoteIP:   net.ParseIP("192.168.1.1"),
					remotePort: uint16(2775),
					systemID:   "rcs01-tp01",
					password:   "passwd1",
					systemType: "rcs",
				},
				smppBindInfo{
					smscName:   "smsc-cluster02-vs",
					remoteIP:   net.ParseIP("192.168.1.2"),
					remotePort: uint16(2775),
					systemID:   "rcs01-tp01",
					password:   "passwd2",
					systemType: "rcs",
				},
			},
		}); !ok {
		t.Errorf("[sample-config.yaml] on parseReader(), returned esme 0: %s", err)
	}

	if ok, err := compareEsme(esmeList[1],
		&esme{
			name: "esme-rcs01-tp02",
			ip:   net.ParseIP("10.1.1.2"),
			port: uint16(2775),
			peerBinds: []smppBindInfo{
				smppBindInfo{
					smscName:   "smsc-cluster01-vs",
					remoteIP:   net.ParseIP("192.168.1.1"),
					remotePort: uint16(2775),
					systemID:   "rcs01-tp02",
					password:   "passwd1",
					systemType: "rcs",
				},
				smppBindInfo{
					smscName:   "smsc-cluster02-vs",
					remoteIP:   net.ParseIP("192.168.1.2"),
					remotePort: uint16(2775),
					systemID:   "rcs01-tp02",
					password:   "passwd2",
					systemType: "rcs",
				},
			},
		}); !ok {
		t.Errorf("[sample-config.yaml] on parseReader(), returned esme 1: %s", err)
	}

}

func compareEsme(received *esme, expected *esme) (bool, error) {
	if received.name != expected.name {
		return false, fmt.Errorf("Received name = (%s), expected = (%s)", received.name, expected.name)
	}

	if !received.ip.Equal(expected.ip) {
		return false, fmt.Errorf("Received ip = (%s), expected = (%s)", received.ip.String(), expected.ip.String())
	}

	if received.port != expected.port {
		return false, fmt.Errorf("Received port = (%d), expected = (%d)", received.port, expected.port)
	}

	if len(received.peerBinds) != len(expected.peerBinds) {
		return false, fmt.Errorf("Received peer bind count = (%d), expected = (%d)", len(received.peerBinds), len(expected.peerBinds))
	}

	for i, receivedPeerBind := range received.peerBinds {
		if ok, err := compareSmppBindInfo(receivedPeerBind, expected.peerBinds[i]); !ok {
			return false, err
		}
	}

	return true, nil
}

func compareSmppBindInfo(received smppBindInfo, expected smppBindInfo) (bool, error) {
	if received.smscName != expected.smscName {
		return false, fmt.Errorf("Received smppBindInfo.smscName = (%s), expected = (%s)", received.smscName, expected.smscName)
	}

	if received.systemID != expected.systemID {
		return false, fmt.Errorf("Received smppBindInfo.systemID = (%s), expected = (%s)", received.systemID, expected.systemID)
	}

	if received.password != expected.password {
		return false, fmt.Errorf("Received smppBindInfo.password = (%s), password = (%s)", received.password, expected.password)
	}

	if received.systemType != expected.systemType {
		return false, fmt.Errorf("Received smppBindInfo.systemType = (%s), expected = (%s)", received.systemType, expected.systemType)
	}

	if received.remotePort != expected.remotePort {
		return false, fmt.Errorf("Received smppBindInfo.remotePort = (%d), expected = (%d)", received.remotePort, expected.remotePort)
	}

	if !received.remoteIP.Equal(expected.remoteIP) {
		return false, fmt.Errorf("Received smppBindInfo.remoteIP = (%s), expected = (%s)", received.remoteIP.String(), expected.remoteIP.String())
	}

	return true, nil
}
