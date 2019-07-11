package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"

	yaml "gopkg.in/yaml.v2"
)

type applicationConfig struct {
	SMSCs            []smscYaml            `yaml:"SMSCs"`
	ESMEs            []esmeYaml            `yaml:"ESMEs"`
	TransceiverBinds []transceiverBindYaml `yaml:"TransceiverBinds"`
}

type smscYaml struct {
	Name         string `yaml:"Name"`
	IP           string `yaml:"IP"`
	Port         uint16 `yaml:"Port"`
	BindPassword string `yaml:"BindPassword"`
}

type esmeYaml struct {
	Name           string `yaml:"Name"`
	IP             string `yaml:"IP"`
	Port           uint16 `yaml:"Port"`
	BindSystemID   string `yaml:"BindSystemID"`
	BindSystemType string `yaml:"BindSystemType"`
}

type transceiverBindYaml struct {
	EsmeName string `yaml:"ESME"`
	SmscName string `yaml:"SMSC"`
}

type applicationConfigYamlReader struct {
}

func newApplicationConfigYamlReader() *applicationConfigYamlReader {
	return &applicationConfigYamlReader{}
}

func (reader *applicationConfigYamlReader) parseFile(fileName string) ([]*esme, []*smsc, error) {
	yamlFileHandle, err := os.Open(fileName)
	defer yamlFileHandle.Close()

	if err != nil {
		return nil, nil, err
	}

	return reader.parseReader(yamlFileHandle)
}

func (reader *applicationConfigYamlReader) parseReader(ioReader io.Reader) ([]*esme, []*smsc, error) {
	var config applicationConfig
	decoder := yaml.NewDecoder(ioReader)
	err := decoder.Decode(&config)

	if err != nil {
		return nil, nil, err
	}

	if len(config.ESMEs) == 0 {
		return nil, nil, errors.New("No ESMEs defined in source yaml")
	}

	esmeDefinitionByName := make(map[string]esmeYaml)
	smscDefinitionByName := make(map[string]smscYaml)

	esmeObjectByName := make(map[string]*esme)
	smscObjectList := make([]*smsc, len(config.SMSCs))
	esmeObjectList := make([]*esme, len(config.ESMEs))

	for i, esmeDefinition := range config.ESMEs {
		bindIP := net.ParseIP(esmeDefinition.IP)

		if bindIP == nil {
			return nil, nil, fmt.Errorf("Invalid IP address [%s] in source yaml for ESME [%s]", esmeDefinition.IP, esmeDefinition.Name)
		}

		esmeDefinitionByName[esmeDefinition.Name] = esmeDefinition
		esme := &esme{name: esmeDefinition.Name, ip: bindIP, port: esmeDefinition.Port, peerBinds: make([]smppBindInfo, 0, 10)}
		esmeObjectByName[esmeDefinition.Name] = esme
		esmeObjectList[i] = esme
	}

	for i, smscDefinition := range config.SMSCs {
		bindIP := net.ParseIP(smscDefinition.IP)

		if bindIP == nil {
			return nil, nil, fmt.Errorf("Invalid IP address [%s] in source yaml for SMSC [%s]", smscDefinition.IP, smscDefinition.Name)
		}

		smscDefinitionByName[smscDefinition.Name] = smscDefinition
		smscObjectList[i] = &smsc{name: smscDefinition.Name, ip: bindIP, port: smscDefinition.Port}
	}

	for _, bindDefinition := range config.TransceiverBinds {
		esme := esmeObjectByName[bindDefinition.EsmeName]
		esmeDefinition := esmeDefinitionByName[bindDefinition.EsmeName]

		if esme == nil {
			return nil, nil, fmt.Errorf("Invalid ESME name [%s] in TransceiverBind definition", bindDefinition.EsmeName)
		}

		smscDefinition, definitionIsInMap := smscDefinitionByName[bindDefinition.SmscName]

		if !definitionIsInMap {
			return nil, nil, fmt.Errorf("Invalid SMSC name [%s] in TransceiverBind definition", bindDefinition.SmscName)
		}

		esme.peerBinds = append(esme.peerBinds,
			smppBindInfo{
				remoteIP:   net.ParseIP(smscDefinition.IP),
				remotePort: smscDefinition.Port,
				password:   smscDefinition.BindPassword,
				smscName:   smscDefinition.Name,
				systemID:   esmeDefinition.BindSystemID,
				systemType: esmeDefinition.BindSystemType,
			})
	}

	return esmeObjectList, smscObjectList, nil
}

func (reader *applicationConfigYamlReader) extractEmseObjectsFromConfigObject(config *applicationConfig) (*esme, error) {

	return nil, nil
}
