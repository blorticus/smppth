package smppth

import (
	"smpp"
	"testing"
	"time"
)

func TestAgentGroupSetOfManagedAgents(t *testing.T) {
	group := NewAgentGroup([]Agent{newMockAgent("foo"), newMockAgent("bar")})

	set := group.SetOfManagedAgents()

	if len(set) != 2 {
		t.Errorf("Expected 2 agents in new group, got = %d", len(set))
	} else {
		receivedNameSet := make(map[string]bool)
		for _, got := range set {
			receivedNameSet[got.Name()] = true
		}

		for _, expectedName := range []string{"foo", "bar"} {
			if _, valueInMap := receivedNameSet[expectedName]; !valueInMap {
				t.Errorf("Expected agent named (%s) in set; not in set", expectedName)
			}
		}
	}

	group.AddAgent(newMockAgent("baz"))

	set = group.SetOfManagedAgents()

	if len(set) != 3 {
		t.Errorf("After AddAgent, expected 3 agents in new group, got = %d", len(set))
	} else {
		receivedNameSet := make(map[string]bool)
		for _, got := range set {
			receivedNameSet[got.Name()] = true
		}

		for _, expectedName := range []string{"foo", "bar", "baz"} {
			if _, valueInMap := receivedNameSet[expectedName]; !valueInMap {
				t.Errorf("After AddAgent, expected agent named (%s) in set; not in set", expectedName)
			}
		}
	}

}

func TestRouteToPduAgentForSendingWithValidAgent(t *testing.T) {
	group := NewAgentGroup([]Agent{newMockAgent("foo"), newMockAgent("bar")})

	err := group.RoutePduToAgentForSending("foo", "baz", smpp.NewPDU(smpp.CommandEnquireLink, 0, 1, []*smpp.Parameter{}, []*smpp.Parameter{}))

	if err != nil {
		t.Errorf("Unexpected error on RoutePduToAgentForSending, enquire-link from 'foo' to 'bar', err = (%s)", err)
	}

	err = group.RoutePduToAgentForSending("bar", "foo", smpp.NewPDU(smpp.CommandEnquireLink, 0, 1, []*smpp.Parameter{}, []*smpp.Parameter{}))

	if err != nil {
		t.Errorf("Unexpected error on RoutePduToAgentForSending, enquire-link from 'bar' to 'foo', err = (%s)", err)
	}
}

func TestRouteToPduAgentForSendingWithInvalidAgent(t *testing.T) {
	group := NewAgentGroup([]Agent{newMockAgent("foo"), newMockAgent("bar")})

	err := group.RoutePduToAgentForSending("baz", "foo", smpp.NewPDU(smpp.CommandEnquireLink, 0, 1, []*smpp.Parameter{}, []*smpp.Parameter{}))

	if err == nil {
		t.Errorf("Expected error on RoutePduToAgentForSending, enquire-link from (non-existent) 'baz' to 'foo'")
	}
}

func TestStartAllAgents(t *testing.T) {
	mockAgentFoo := newMockAgent("foo")
	mockAgentBar := newMockAgent("bar")
	agents := []Agent{mockAgentFoo, mockAgentBar}

	group := NewAgentGroup(agents)

	group.StartAllAgents()

	if mockAgentFoo.wasEventLoopStarted() == false {
		t.Errorf("For agent foo, expected that eventLoopStarted == true, is false")
	}

	if mockAgentBar.wasEventLoopStarted() == false {
		t.Errorf("For agent bar, expected that eventLoopStarted == true, is false")
	}
}

type mockAgent struct {
	name                string
	eventLoopBlock      chan bool
	lastReceivedMessage *MessageDescriptor
}

func newMockAgent(name string) *mockAgent {
	return &mockAgent{name: name, eventLoopBlock: make(chan bool), lastReceivedMessage: nil}
}

func (agent *mockAgent) SetAgentEventChannel(chan<- *AgentEvent) {

}

func (agent *mockAgent) Name() string {
	return agent.name
}

func (agent *mockAgent) StartEventLoop() {
	agent.eventLoopBlock <- true
}

func (agent *mockAgent) SendMessageToPeer(message *MessageDescriptor) error {
	return nil
}

func (agent *mockAgent) wasEventLoopStarted() bool {
	wasStarted := false

	select {
	case <-agent.eventLoopBlock:
		wasStarted = true
	case <-time.After(time.Second * 3):
	}

	return wasStarted
}
