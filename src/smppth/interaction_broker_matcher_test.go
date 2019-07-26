package smppth

import (
	"fmt"
	"testing"
)

func TestCommandMatcherEmptyDigestCommand(t *testing.T) {
	matcher := newInteractionBrokerCommandMatcher()

	matcher.digestCommand("")

	if matcher.saysThisIsAValidSendCommand() {
		t.Errorf("On empty digestCommand() matcher saysThisIsAValidSendCommand but should not")
	}
}

func TestCommandMatcherNoMatchDigestCommand(t *testing.T) {
	for _, command := range []string{"send", "send ", "send submit-sm to ", "send submit-sm to bar", "foo: send submit-sm to ", "some esme: send submit-sm to bar", "esme: send submit-sm  to bar"} {
		if cleanMatcherReportsThisIsAValidSetCommand(command) {
			t.Errorf("For (%s) matcher saysThisIsAValidSendCommand but should not", command)
		}
	}
}

func TestValidSendCommands(t *testing.T) {
	for _, command := range []string{"foo: send submit-sm to bar", "some-esme-01: send submit-sm to some-bar-01"} {
		if !cleanMatcherReportsThisIsAValidSetCommand(command) {
			t.Errorf("For (%s) matcher says it is not a valid command (saysThisIsAValidSendCommand is false) but should not", command)
		}
	}
}

func TestBreakSendCommandIntoStructWithoutParams(t *testing.T) {
	matches, err := compareSendCommandStructsCreatedFrom("esme01: send submit-sm to smsc01", &interactionBrokerSendCommand{
		commandParametersMap: map[string]string{},
		commandTypeName:      "submit-sm",
		pduReceiverName:      "smsc01",
		pduSenderName:        "esme01",
	})

	if !matches {
		t.Errorf(err.Error())
	}
}

func TestBreakSendCommandIntoStructWithSingleParam(t *testing.T) {
	matches, err := compareSendCommandStructsCreatedFrom("esme01: send submit-sm to smsc01 short_message=\"this is a short message\"", &interactionBrokerSendCommand{
		commandParametersMap: map[string]string{"short_message": "this is a short message"},
		commandTypeName:      "submit-sm",
		pduReceiverName:      "smsc01",
		pduSenderName:        "esme01",
	})

	if !matches {
		t.Errorf(err.Error())
	}
}

func TestBreakSendCommandIntoStructWithThreeParams(t *testing.T) {
	matches, err := compareSendCommandStructsCreatedFrom("esme01: send submit-sm to smsc01 short_message='this is a short message' dest_addr_ton=Private dest_addr=001100", &interactionBrokerSendCommand{
		commandParametersMap: map[string]string{"short_message": "this is a short message", "dest_addr_ton": "Private", "dest_addr": "001100"},
		commandTypeName:      "submit-sm",
		pduReceiverName:      "smsc01",
		pduSenderName:        "esme01",
	})

	if !matches {
		t.Errorf(err.Error())
	}
}

func compareSendCommandStructsCreatedFrom(command string, expected *interactionBrokerSendCommand) (bool, error) {
	matcher := newInteractionBrokerCommandMatcher()
	matcher.digestCommand(command)

	if !matcher.saysThisIsAValidSendCommand() {
		return false, fmt.Errorf("For command (%s), matcher says this is not a valid send command", command)
	}

	sendCommandStruct := matcher.breakSendCommandIntoStruct()

	if sendCommandStruct == nil {
		return false, fmt.Errorf("For command (%s), breakSendCommandIntoStruct() returns nil", command)
	}

	if sendCommandStruct.commandTypeName != expected.commandTypeName {
		return false, fmt.Errorf("For command (%s) expected commandTypeName = (%s), got = (%s)", command, expected.commandTypeName, sendCommandStruct.commandTypeName)
	}

	if sendCommandStruct.pduReceiverName != expected.pduReceiverName {
		return false, fmt.Errorf("For command (%s) expected pduReceiverName = (%s), got = (%s)", command, expected.pduReceiverName, sendCommandStruct.pduReceiverName)
	}

	if sendCommandStruct.pduSenderName != expected.pduSenderName {
		return false, fmt.Errorf("For command (%s) expected pduSenderName = (%s), got = (%s)", command, expected.pduSenderName, sendCommandStruct.pduSenderName)
	}

	for key, expectedValue := range expected.commandParametersMap {
		gotValue, keyIsInGotMap := sendCommandStruct.commandParametersMap[key]

		if !keyIsInGotMap {
			return false, fmt.Errorf("For command (%s) expected command parameter (%s) but did not get that parameter", command, key)
		}

		if expectedValue != gotValue {
			return false, fmt.Errorf("For command (%s) expected command parameter (%s) = (%s), got (%s) = (%s)", command, key, expectedValue, key, gotValue)
		}
	}

	for key := range sendCommandStruct.commandParametersMap {
		_, keyIsInExpectedMap := expected.commandParametersMap[key]

		if !keyIsInExpectedMap {
			return false, fmt.Errorf("For command (%s) got parameter (%s) but did not expect that parameter", command, key)
		}
	}

	return true, nil
}

func cleanMatcherReportsThisIsAValidSetCommand(command string) bool {
	matcher := newInteractionBrokerCommandMatcher()
	matcher.digestCommand(command)
	return matcher.saysThisIsAValidSendCommand()
}
