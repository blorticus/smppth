package smppth

import (
	"fmt"
	"regexp"
	"smpp"
)

// TextCommandProcessor connects to a reader, which accepts structured command messages, and a writer,
// which emits events on behalf of testharness agents that have been started.
type TextCommandProcessor struct {
	helpCommandMatcher           *regexp.Regexp
	sendCommandMatcher           *regexp.Regexp
	sendCommandParametersMatcher *regexp.Regexp
	emptyParameterMatcher        *regexp.Regexp
	emptyLastParameterMatcher    *regexp.Regexp
	doubleQuotedParameterMatcher *regexp.Regexp
	singleQuotedParameterMatcher *regexp.Regexp
	unquotedParameterMatcher     *regexp.Regexp
	lastSetOfMatchGroupValues    []string
}

// NewTextCommandProcessor creates an empty broker, where the prompt output stream is set to STDOUT,
// the command input stream is set to STDIN, and the event output writer is set to STDOUT.  Built-in
// outputs are not used by default.
func NewTextCommandProcessor() *TextCommandProcessor {
	return &TextCommandProcessor{
		helpCommandMatcher:           regexp.MustCompile(`^help$`),
		sendCommandMatcher:           regexp.MustCompile(`^(\S+?): send (\S+) to (\S+) *(.*)?$`),
		sendCommandParametersMatcher: regexp.MustCompile(`^ *short_message="(.+?)" *$`),
		emptyParameterMatcher:        regexp.MustCompile(`^(\S+)=\s+`),
		emptyLastParameterMatcher:    regexp.MustCompile(`^(\S+)=$`),
		doubleQuotedParameterMatcher: regexp.MustCompile(`^(\S+)="(.+?)"\s*`),
		singleQuotedParameterMatcher: regexp.MustCompile(`^(\S+)='(.+?)'\s*`),
		unquotedParameterMatcher:     regexp.MustCompile(`^(\S+)=(\S+)\s*`),
		lastSetOfMatchGroupValues:    []string{},
	}
}

func (processor *TextCommandProcessor) ConvertCommandLineStringToUserCommand(commandLine string) (*UserCommand, error) {
	processor.lastSetOfMatchGroupValues = []string{}

	if processor.thisIsTheHelpCommand(commandLine) {
		return &UserCommand{
			Type: Help,
		}, nil
	}

	if processor.thisIsASendCommand(commandLine) {
		smppCommandName := processor.lastSetOfMatchGroupValues[2]

		smppCommandID, aValidCommandName := smpp.CommandIDFromString(smppCommandName)

		if !aValidCommandName {
			return nil, fmt.Errorf("Invalid smpp PDU type name (%s)", smppCommandName)
		}

		return &UserCommand{
			Type: SendPDU,
			Details: &SendPduDetails{
				NameOfAgentThatWillSendPdu:     processor.lastSetOfMatchGroupValues[1],
				NameOfPeerThatShouldReceivePdu: processor.lastSetOfMatchGroupValues[3],
				TypeOfSmppPDU:                  smppCommandID,
				StringParametersMap:            processor.breakParametersIntoMap(processor.lastSetOfMatchGroupValues[4]),
			},
		}, nil
	}

	return nil, fmt.Errorf("Command not understood")
}

func (processor *TextCommandProcessor) CommandTextHelp() string {
	return `
$esme_name: send submit-sm to $smsc_name short_message="$message" dest_addr=$addr
$esme_name: send enquire-link to $smsc_name
`
}

func (processor *TextCommandProcessor) thisIsTheHelpCommand(commandLine string) bool {
	return processor.helpCommandMatcher.Match([]byte(commandLine))
}

func (processor *TextCommandProcessor) thisIsASendCommand(commandLine string) bool {
	submatches := processor.sendCommandMatcher.FindStringSubmatch(commandLine)

	if len(submatches) > 0 {
		processor.lastSetOfMatchGroupValues = submatches
		return true
	}

	return false
}

func (processor *TextCommandProcessor) breakParametersIntoMap(parameterString string) map[string]string {
	parameterMap := make(map[string]string)

	for len(parameterString) > 0 {
		foundSomeTypeOfMatch := false

		for _, compiledMatcher := range []*regexp.Regexp{processor.emptyLastParameterMatcher, processor.emptyParameterMatcher, processor.doubleQuotedParameterMatcher, processor.singleQuotedParameterMatcher, processor.unquotedParameterMatcher} {
			itDoesMatch, parameterName, parameterValue, parameterStringLength := processor.extractMappableValueAndMatchingLengthFromMatcher(compiledMatcher, parameterString)

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

func (processor *TextCommandProcessor) extractMappableValueAndMatchingLengthFromMatcher(compiledRegexp *regexp.Regexp, parseString string) (doesMatch bool, name string, value string, matchLen int) {
	groups := compiledRegexp.FindStringSubmatch(parseString)

	if groups == nil {
		return false, "", "", 0
	}

	if len(groups) == 2 {
		return true, groups[1], "", len(groups[0])
	}

	return true, groups[1], groups[2], len(groups[0])
}
