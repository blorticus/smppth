package smppth

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

// ApplicationConfigYamlReader reads a testharness application YAML config file
type ApplicationConfigYamlReader struct {
}

// NewApplicationConfigYamlReader creates a new, empty ApplicationConfigYamlReader
func NewApplicationConfigYamlReader() *ApplicationConfigYamlReader {
	return &ApplicationConfigYamlReader{}
}

// ParseFile opens a file and treats its contents as a validly formatted testharness config YAML file
func (reader *ApplicationConfigYamlReader) ParseFile(fileName string) ([]*ESME, []*SMSC, error) {
	yamlFileHandle, err := os.Open(fileName)
	defer yamlFileHandle.Close()

	if err != nil {
		return nil, nil, err
	}

	return reader.ParseReader(yamlFileHandle)
}

// ParseReader reads from an io.Reader stream, treating the contents provided as a validly formatted
// testharness config YAML file
func (reader *ApplicationConfigYamlReader) ParseReader(ioReader io.Reader) ([]*ESME, []*SMSC, error) {
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

	esmeObjectByName := make(map[string]*ESME)
	smscObjectList := make([]*SMSC, len(config.SMSCs))
	esmeObjectList := make([]*ESME, len(config.ESMEs))

	for i, esmeDefinition := range config.ESMEs {
		bindIP := net.ParseIP(esmeDefinition.IP)

		if bindIP == nil {
			return nil, nil, fmt.Errorf("Invalid IP address [%s] in source yaml for ESME [%s]", esmeDefinition.IP, esmeDefinition.Name)
		}

		esmeDefinitionByName[esmeDefinition.Name] = esmeDefinition
		esme := NewEsme(esmeDefinition.Name, bindIP, esmeDefinition.Port)
		esmeObjectByName[esmeDefinition.Name] = esme
		esmeObjectList[i] = esme
	}

	for i, smscDefinition := range config.SMSCs {
		bindIP := net.ParseIP(smscDefinition.IP)

		if bindIP == nil {
			return nil, nil, fmt.Errorf("Invalid IP address [%s] in source yaml for SMSC [%s]", smscDefinition.IP, smscDefinition.Name)
		}

		smscDefinitionByName[smscDefinition.Name] = smscDefinition
		smscObjectList[i] = &SMSC{name: smscDefinition.Name, ip: bindIP, port: smscDefinition.Port}
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
