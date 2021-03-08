package smppth

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/blorticus/smpp"
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
		{
			commandToTest: "foo: send submit-sm to bar",
			expectedStruct: &UserCommand{
				Type: SendPDU,
				Details: &SendPduDetails{
					TypeOfSmppPDU:                  smpp.CommandSubmitSm,
					StringParametersMap:            make(map[string]string),
					NameOfPeerThatShouldReceivePdu: "bar",
					NameOfAgentThatWillSendPdu:     "foo",
				},
			},
		},
		{
			commandToTest: "foo: send enquire-link to bar",
			expectedStruct: &UserCommand{
				Type: SendPDU,
				Details: &SendPduDetails{
					TypeOfSmppPDU:                  smpp.CommandEnquireLink,
					StringParametersMap:            make(map[string]string),
					NameOfPeerThatShouldReceivePdu: "bar",
					NameOfAgentThatWillSendPdu:     "foo",
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
		{
			commandToTest: `foo: send submit-sm to bar short_message="This is a short message" dest_addr=001100`,
			expectedStruct: &UserCommand{
				Type: SendPDU,
				Details: &SendPduDetails{
					TypeOfSmppPDU:                  smpp.CommandSubmitSm,
					StringParametersMap:            map[string]string{"short_message": "This is a short message", "dest_addr": "001100"},
					NameOfPeerThatShouldReceivePdu: "bar",
					NameOfAgentThatWillSendPdu:     "foo",
				},
			},
		},
		{
			commandToTest: `foo: send enquire-link to bar alpha= beta=gamma`,
			expectedStruct: &UserCommand{
				Type: SendPDU,
				Details: &SendPduDetails{
					TypeOfSmppPDU:                  smpp.CommandEnquireLink,
					StringParametersMap:            map[string]string{"alpha": "", "beta": "gamma"},
					NameOfPeerThatShouldReceivePdu: "bar",
					NameOfAgentThatWillSendPdu:     "foo",
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

	if reflect.TypeOf(got.Details).Elem().Name() != "SendPduDetails" {
		return fmt.Errorf("Expected Details from received struct to be (SendPduDetails), got = (%s)", reflect.TypeOf(got.Details).Name())
	}

	expectedDetails := expected.Details.(*SendPduDetails)
	gotDetails := got.Details.(*SendPduDetails)

	if expectedDetails.TypeOfSmppPDU != gotDetails.TypeOfSmppPDU {
		return fmt.Errorf("Expected PduCommandIDType = ([%d] %s), got = ([%d] %s)", int(expectedDetails.TypeOfSmppPDU), smpp.CommandName(expectedDetails.TypeOfSmppPDU), int(gotDetails.TypeOfSmppPDU), smpp.CommandName(gotDetails.TypeOfSmppPDU))
	}

	if expectedDetails.NameOfAgentThatWillSendPdu != gotDetails.NameOfAgentThatWillSendPdu {
		return fmt.Errorf("Expected NameOfAgentThatWillSendPdu = (%s), got = (%s)", expectedDetails.NameOfAgentThatWillSendPdu, gotDetails.NameOfAgentThatWillSendPdu)
	}

	if expectedDetails.NameOfPeerThatShouldReceivePdu != gotDetails.NameOfPeerThatShouldReceivePdu {
		return fmt.Errorf("Expected NameOfPeerThatShouldReceivePdu = (%s), got = (%s)", expectedDetails.NameOfPeerThatShouldReceivePdu, gotDetails.NameOfPeerThatShouldReceivePdu)
	}

	if len(expectedDetails.StringParametersMap) != len(gotDetails.StringParametersMap) {
		return fmt.Errorf("Expected %d entries in StringParametersMap map, got = %d", len(expectedDetails.StringParametersMap), len(gotDetails.StringParametersMap))
	}

	for k, v := range expectedDetails.StringParametersMap {
		gv, hasKey := gotDetails.StringParametersMap[k]

		if !hasKey {
			return fmt.Errorf("Expected StringParametersMap key (%s), but did not get that key", k)
		}

		if v != gv {
			return fmt.Errorf("Expected value (%s) for StringParametersMap with key (%s), got value = (%s)", v, k, gv)
		}
	}

	return nil
}
