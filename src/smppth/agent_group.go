package smppth

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"smpp"
)

// AgentGroup manages a group of Agents, routing messages for delivery to them,
// and providing them with a common event channel
type AgentGroup struct {
	mapOfAgentNameToAgentObject map[string]Agent
	sharedAgentEventChannel     chan *AgentEvent
	debugLogger                 *log.Logger
}

// NewAgentGroup creates a new AgentGroup and adds to it the provided list of managed agents
func NewAgentGroup(listOfManagedAgents []Agent) *AgentGroup {
	group := &AgentGroup{
		mapOfAgentNameToAgentObject: make(map[string]Agent),
		sharedAgentEventChannel:     make(chan *AgentEvent),
		debugLogger:                 log.New(ioutil.Discard, "", 0),
	}

	for _, agent := range listOfManagedAgents {
		group.mapOfAgentNameToAgentObject[agent.Name()] = agent
	}

	return group
}

// AttachDebugLoggerWriter enables debug logging, sending messages to the provided Writer.
func (group *AgentGroup) AttachDebugLoggerWriter(writer io.Writer) {
	group.debugLogger = log.New(writer, "(AgentGroup): ", 0)
}

// RoutePduToAgentForSending accepts an SMPP PDU, and routes it to the named source peer,
// so that peer can send it to the named destination peer
func (group *AgentGroup) RoutePduToAgentForSending(nameOfSourcePeer string, nameOfDestinationPeer string, pduToSend *smpp.PDU) error {
	agentObject := group.mapOfAgentNameToAgentObject[nameOfSourcePeer]

	if agentObject == nil {
		return fmt.Errorf("This AgentGroup is not managing an agent named [%s]", nameOfSourcePeer)
	}

	err := agentObject.SendMessageToPeer(&MessageDescriptor{
		NameOfSendingPeer:   nameOfSourcePeer,
		NameOfReceivingPeer: nameOfDestinationPeer,
		PDU:                 pduToSend,
	})

	if err != nil {
		return err
	}

	return nil
}

// AddAgent adds a agent to the AgentGroup for management.  This will silently
// replace an already managed agent with the same name.
func (group *AgentGroup) AddAgent(agent Agent) {
	group.mapOfAgentNameToAgentObject[agent.Name()] = agent
}

// AddAgents adds multiple agents to the AgentGroup for management.  This will
// silently replace any managed agents that have a name matching an already managed
// agent
func (group *AgentGroup) AddAgents(agents []Agent) {
	for _, agent := range agents {
		group.AddAgent(agent)
	}
}

// RemoveAgent removes an agent under management by name.  If there is no agent
// with a matching name under management, it is silently ignored.  If the provided
// agent does not match the managed agent by name, the agent is removed anyway
func (group *AgentGroup) RemoveAgent(agent Agent) {
	delete(group.mapOfAgentNameToAgentObject, agent.Name())
}

// RemoveAgents remove multiple agents under management by their names.  For each
// agent in the list, if there is no agent with a matching name under management,
// it is silently ignored.  Also, for each agent in the list, if the provided
// agent does not match the managed agent by name, the agent is removed anyway
func (group *AgentGroup) RemoveAgents(agents []Agent) {
	for _, agent := range agents {
		group.RemoveAgent(agent)
	}
}

// SharedAgentEventChannel returns the shared event channel used by all of the managed
// agents
func (group *AgentGroup) SharedAgentEventChannel() <-chan *AgentEvent {
	return group.sharedAgentEventChannel
}

// SetOfManagedAgents returns the current list of agents that are under management
// in this group
func (group *AgentGroup) SetOfManagedAgents() []Agent {
	managedAgentList := make([]Agent, len(group.mapOfAgentNameToAgentObject))

	i := 0
	for _, agentObject := range group.mapOfAgentNameToAgentObject {
		managedAgentList[i] = agentObject
		i++
	}

	return managedAgentList
}

// StartAllAgents executes StartEventLoop() on all managed agents, passing each
// the shared event channel.  The agents are started in no particular order.
func (group *AgentGroup) StartAllAgents() {
	for _, agent := range group.mapOfAgentNameToAgentObject {
		agent.SetAgentEventChannel(group.sharedAgentEventChannel)
		go agent.StartEventLoop()
	}
}
