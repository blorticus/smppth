package smppth

import (
	"fmt"
	"smpp"
	"testing"
)

func TestCommandProcessorEmptyCommand(t *testing.T) {
	for _, command := range []string{"", "   ", "\t", "\n"} {
		processor := NewTextCommandProcessor()

		_, err := processor.ConvertCommandLineStringToUserCommand(command)

		if err == nil {
			t.Errorf("For (%s) expected error on ConvertCommandLineStringToUserCommand but did not get one", command)
		}
	}
}

func TestCommandProcessorInvalidCommandStrings(t *testing.T) {
	for _, command := range []string{"send", "send ", "send submit-sm to ", "send submit-sm to bar", "foo: send submit-sm to ", "some esme: send submit-sm to bar", "esme: send submit-sm  to bar"} {
		processor := NewTextCommandProcessor()

		_, err := processor.ConvertCommandLineStringToUserCommand(command)

		if err == nil {
			t.Errorf("For (%s) expected error on ConvertCommandLineStringToUserCommand but did not get one", command)
		}
	}
}

type sendCommandCompare struct {
	commandToTest  string
	expectedStruct *UserCommand
}

func TestValidSendCommandsWithoutParameters(t *testing.T) {
	testSet := []*sendCommandCompare{
		&sendCommandCompare{
			commandToTest: "foo: send submit-sm to bar",
			expectedStruct: &UserCommand{
				Type:              SendPDU,
				PduCommandIDType:  smpp.CommandSubmitSm,
				CommandParameters: make(map[string]string),
				Peers: &PeerDetails{
					NameOfReceivingPeer: "bar",
					NameOfSendingAgent:  "foo",
				},
			},
		},
		&sendCommandCompare{
			commandToTest: "foo: send enquire-link to bar",
			expectedStruct: &UserCommand{
				Type:              SendPDU,
				PduCommandIDType:  smpp.CommandEnquireLink,
				CommandParameters: make(map[string]string),
				Peers: &PeerDetails{
					NameOfReceivingPeer: "bar",
					NameOfSendingAgent:  "foo",
				},
			},
		},
	}

	for _, testCase := range testSet {
		processor := NewTextCommandProcessor()

		userCommandStruct, err := processor.ConvertCommandLineStringToUserCommand(testCase.commandToTest)

		if err != nil {
			t.Errorf("For (%s) expected error on ConvertCommandLineStringToUserCommand but did not get one", testCase.commandToTest)
		}

		if err := compareUserCommandStructsForSendCommand(testCase.expectedStruct, userCommandStruct); err != nil {
			t.Errorf("For (%s) struct is not as expected: %s", testCase.commandToTest, err)
		}
	}
}

func TestValidSendCommandsWithParameters(t *testing.T) {
	testSet := []*sendCommandCompare{
		&sendCommandCompare{
			commandToTest: `foo: send submit-sm to bar short_message="This is a short message" dest_addr=001100`,
			expectedStruct: &UserCommand{
				Type:              SendPDU,
				PduCommandIDType:  smpp.CommandSubmitSm,
				CommandParameters: map[string]string{"short_message": "This is a short message", "dest_addr": "001100"},
				Peers: &PeerDetails{
					NameOfReceivingPeer: "bar",
					NameOfSendingAgent:  "foo",
				},
			},
		},
		&sendCommandCompare{
			commandToTest: `foo: send enquire-link to bar alpha= beta=gamma`,
			expectedStruct: &UserCommand{
				Type:              SendPDU,
				PduCommandIDType:  smpp.CommandEnquireLink,
				CommandParameters: map[string]string{"alpha": "", "beta": "gamma"},
				Peers: &PeerDetails{
					NameOfReceivingPeer: "bar",
					NameOfSendingAgent:  "foo",
				},
			},
		},
	}

	for _, testCase := range testSet {
		processor := NewTextCommandProcessor()

		userCommandStruct, err := processor.ConvertCommandLineStringToUserCommand(testCase.commandToTest)

		if err != nil {
			t.Errorf("For (%s) expected error on ConvertCommandLineStringToUserCommand but did not get one", testCase.commandToTest)
		}

		if err := compareUserCommandStructsForSendCommand(testCase.expectedStruct, userCommandStruct); err != nil {
			t.Errorf("For (%s) struct is not as expected: %s", testCase.commandToTest, err)
		}
	}
}

func compareUserCommandStructsForSendCommand(expected *UserCommand, got *UserCommand) error {
	if expected == nil {
		if got == nil {
			return nil
		}

		return fmt.Errorf("expected nil struct, got non-nil struct")
	}

	if got == nil {
		return fmt.Errorf("expected non-nil struct, got nil struct")
	}

	if expected.Type != got.Type {
		return fmt.Errorf("Expected Type = (%d), got = (%d)", int(expected.Type), int(got.Type))
	}

	if expected.PduCommandIDType != got.PduCommandIDType {
		return fmt.Errorf("Expected PduCommandIDType = ([%d] %s), got = ([%d] %s)", int(expected.PduCommandIDType), smpp.CommandName(expected.PduCommandIDType), int(got.PduCommandIDType), smpp.CommandName(got.PduCommandIDType))
	}

	if got.Peers == nil {
		return fmt.Errorf("Got nil Peers value")
	}

	if expected.Peers.NameOfSendingAgent != got.Peers.NameOfSendingAgent {
		return fmt.Errorf("Expected Peers.NameOfSendingAgent = (%s), got = (%s)", expected.Peers.NameOfSendingAgent, got.Peers.NameOfSendingAgent)
	}

	if expected.Peers.NameOfReceivingPeer != got.Peers.NameOfReceivingPeer {
		return fmt.Errorf("Expected Peers.NameOfSendingAgent = (%s), got = (%s)", expected.Peers.NameOfReceivingPeer, got.Peers.NameOfReceivingPeer)
	}

	if len(expected.CommandParameters) != len(got.CommandParameters) {
		return fmt.Errorf("Expected %d entries in CommandParameters map, got = %d", len(expected.CommandParameters), len(got.CommandParameters))
	}

	for k, v := range expected.CommandParameters {
		gv, hasKey := got.CommandParameters[k]

		if !hasKey {
			return fmt.Errorf("Expected CommandParameters key (%s), but did not get that key", k)
		}

		if v != gv {
			return fmt.Errorf("Expected value (%s) for CommandParameter with key (%s), got value = (%s)", v, k, gv)
		}
	}

	return nil
}
