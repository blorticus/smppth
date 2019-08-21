package smppth

import (
	"fmt"
	"regexp"

	smpp "github.com/blorticus/smpp-go"
)

// TextCommandProcessor accepts incoming text commands and, if they match the TextCommandProcessor syntax,
// then it emits a corresponding UserCommand structs.  The syntax includes:
//    $agent_name: send enquire-link to $peer_name
//    $agent_name: send submit-sm to $peer_name [source_addr_ton=$sat] [source_address=$saddr] [dest_addr_ton=$dat] [destination_address=$daddr] [short_message=$msg]
//    help
type TextCommandProcessor struct {
	helpCommandMatcher           *regexp.Regexp
	quitCommandMatcher           *regexp.Regexp
	sendCommandMatcher           *regexp.Regexp
	sendCommandParametersMatcher *regexp.Regexp
	emptyParameterMatcher        *regexp.Regexp
	emptyLastParameterMatcher    *regexp.Regexp
	doubleQuotedParameterMatcher *regexp.Regexp
	singleQuotedParameterMatcher *regexp.Regexp
	unquotedParameterMatcher     *regexp.Regexp
	lastSetOfMatchGroupValues    []string
}

// NewTextCommandProcessor creates a TextCommandProcessor.
func NewTextCommandProcessor() *TextCommandProcessor {
	return &TextCommandProcessor{
		helpCommandMatcher:           regexp.MustCompile(`^help$`),
		quitCommandMatcher:           regexp.MustCompile(`^quit$`),
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

// ConvertCommandLineStringToUserCommand accepts a string command line, and if it matches the TextCommandProcessor
// syntax, returns the matching UserCommand struct.  It returns an error if some part of the command is not understood.
func (processor *TextCommandProcessor) ConvertCommandLineStringToUserCommand(commandLine string) (*UserCommand, error) {
	processor.lastSetOfMatchGroupValues = []string{}

	if processor.thisIsTheQuitCommand(commandLine) {
		return &UserCommand{
			Type: Quit,
		}, nil
	}

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

func (processor *TextCommandProcessor) thisIsTheQuitCommand(commandLine string) bool {
	return processor.quitCommandMatcher.Match([]byte(commandLine))
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
