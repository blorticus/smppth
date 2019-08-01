package smppth

import "regexp"

type digestedCommand struct {
	isNotValid               bool
	reasonCommandIsNotValid  string
	compiledUserInputCommand *textInteractionBrokerValidUserInputCommand
}

type userInputCommandType int

const (
	helpCommand userInputCommandType = iota
	sendCommand
)

type textInteractionBrokerSendCommand struct {
	pduSenderName        string
	pduReceiverName      string
	smppCommandTypeName  string
	commandParametersMap map[string]string
}

type textInteractionBrokerValidUserInputCommand struct {
	commandType            userInputCommandType
	commandString          string
	sendCommandInformation *textInteractionBrokerSendCommand
}

type textInteractionBrokerCommandMatcher struct {
	//	digestingCommand                                string
	helpCommandMatcher                              *regexp.Regexp
	sendCommandMatcher                              *regexp.Regexp
	sendCommandParametersMatcher                    *regexp.Regexp
	emptyParameterMatcher                           *regexp.Regexp
	emptyLastParameterMatcher                       *regexp.Regexp
	doubleQuotedParameterMatcher                    *regexp.Regexp
	singleQuotedParameterMatcher                    *regexp.Regexp
	unquotedParameterMatcher                        *regexp.Regexp
	lastMatchStringSet                              []string
	lastSendCommandParameterSetIncludedShortMessage bool
}

func newTextInteractionBrokerCommandMatcher() *textInteractionBrokerCommandMatcher {
	return &textInteractionBrokerCommandMatcher{
		//		digestingCommand:                                "",
		helpCommandMatcher:                              regexp.MustCompile(`^help$`),
		sendCommandMatcher:                              regexp.MustCompile(`^(\S+?): send (\S+) to (\S+) *(.*)?$`),
		sendCommandParametersMatcher:                    regexp.MustCompile(`^ *short_message="(.+?)" *$`),
		emptyParameterMatcher:                           regexp.MustCompile(`^(\S+)=\s+`),
		emptyLastParameterMatcher:                       regexp.MustCompile(`^(\S+)=$`),
		doubleQuotedParameterMatcher:                    regexp.MustCompile(`^(\S+)="(.+?)"\s*`),
		singleQuotedParameterMatcher:                    regexp.MustCompile(`^(\S+)='(.+?)'\s*`),
		unquotedParameterMatcher:                        regexp.MustCompile(`^(\S+)=(\S+)\s*`),
		lastMatchStringSet:                              []string{},
		lastSendCommandParameterSetIncludedShortMessage: false,
	}
}

func (matcher *textInteractionBrokerCommandMatcher) digestCommandString(command string) *digestedCommand {
	if digestedCommand := matcher.commandIsHelp(command); digestedCommand != nil {
		return digestedCommand
	}

	if digestedCommand := matcher.commandIsSend(command); digestedCommand != nil {
		return digestedCommand
	}

	return &digestedCommand{isNotValid: true, reasonCommandIsNotValid: "command structure not understood", compiledUserInputCommand: nil}
}

func (matcher *textInteractionBrokerCommandMatcher) commandIsHelp(command string) *digestedCommand {
	if matcher.helpCommandMatcher.Match([]byte(command)) {
		return &digestedCommand{
			isNotValid:              false,
			reasonCommandIsNotValid: "",
			compiledUserInputCommand: &textInteractionBrokerValidUserInputCommand{
				commandString:          command,
				commandType:            helpCommand,
				sendCommandInformation: nil,
			},
		}
	}

	return nil
}

func (matcher *textInteractionBrokerCommandMatcher) commandIsSend(command string) *digestedCommand {
	submatches := matcher.sendCommandMatcher.FindStringSubmatch(command)

	if len(submatches) == 0 {
		return nil
	}

	return &digestedCommand{
		isNotValid:              false,
		reasonCommandIsNotValid: "",
		compiledUserInputCommand: &textInteractionBrokerValidUserInputCommand{
			commandString: command,
			commandType:   helpCommand,
			sendCommandInformation: &textInteractionBrokerSendCommand{
				pduSenderName:        submatches[1],
				pduReceiverName:      submatches[3],
				smppCommandTypeName:  submatches[2],
				commandParametersMap: matcher.breakParametersIntoMap(submatches[4]),
			},
		},
	}
}

func (matcher *textInteractionBrokerCommandMatcher) breakParametersIntoMap(parameterString string) map[string]string {
	parameterMap := make(map[string]string)

	for len(parameterString) > 0 {
		foundSomeTypeOfMatch := false

		for _, compiledMatcher := range []*regexp.Regexp{matcher.emptyLastParameterMatcher, matcher.emptyParameterMatcher, matcher.doubleQuotedParameterMatcher, matcher.singleQuotedParameterMatcher, matcher.unquotedParameterMatcher} {
			itDoesMatch, parameterName, parameterValue, parameterStringLength := matcher.extractMappableValueAndMatchingLengthFromMatcher(compiledMatcher, parameterString)

			if itDoesMatch {
				parameterMap[parameterName] = parameterValue
				parameterString = parameterString[parameterStringLength:]
				foundSomeTypeOfMatch = true
				break
			}
		}

		if !foundSomeTypeOfMatch {
			break
		}
	}

	return parameterMap
}

func (matcher *textInteractionBrokerCommandMatcher) extractMappableValueAndMatchingLengthFromMatcher(compiledRegexp *regexp.Regexp, parseString string) (doesMatch bool, name string, value string, matchLen int) {
	groups := compiledRegexp.FindStringSubmatch(parseString)

	if groups == nil {
		return false, "", "", 0
	}

	if len(groups) == 2 {
		return true, groups[1], "", len(groups[0])
	}

	return true, groups[1], groups[2], len(groups[0])
}

// func (matcher *textInteractionBrokerCommandMatcher) saysThatSendCommandParametersContainedShortMessage() bool {
// 	return matcher.lastSendCommandParameterSetIncludedShortMessage
// }
