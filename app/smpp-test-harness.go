package main

import (
	"log"
	"os"
	"path"
	"smpp"
	"smppth"
)

var errLogger *log.Logger

func main() {
	// fh, err := os.OpenFile("/tmp/output.log", os.O_CREATE|os.O_WRONLY, 0664)
	// if err != nil {
	// 	panic(fmt.Sprintf("Failed to open file for writing: %s", err))
	// }

	// debugLogger := log.New(fh, "", 0)

	// errLogger = initializeErrorLogger()

	// runAsEsmesOrSmscs, yamlConfigFileName := parseCommandLineOptions()

	// esmes, smscs, err := smppth.NewApplicationConfigYamlReader().ParseFile(yamlConfigFileName)
	// fatalOutsideUIOnErr(err)

	// var agentGroup *smppth.AgentGroup
	// if runAsEsmesOrSmscs == "esmes" {
	// 	agentGroup = generateEsmeAgentGroup(esmes, smscs)
	// } else {
	// 	agentGroup = generateSmscAgentGroup(esmes, smscs)
	// }

	// sharedAgentEventChannel := agentGroup.SharedAgentEventChannel()

	textCommandProcessor := smppth.NewTextCommandProcessor()

	ui := BuildUserInterface()
	commandInputTextChannel := ui.UserInputStringCommandChannel()
	// ui.AttachDebugLogger(debugLogger)

	// agentGroup.StartAllAgents()

	go func() {
		//pduFactory := smppth.NewPduFactory()

		for {
			select {
			case nextCommandLine := <-commandInputTextChannel:
				structuredCommand, invalidCommandError := textCommandProcessor.ConvertCommandLineStringToUserCommand(nextCommandLine)
				if invalidCommandError != nil {
					ui.WriteLineToEventBox("Invalid command")
				} else {
					switch structuredCommand.Type {
					case smppth.SendPDU:
						switch structuredCommand.PduCommandIDType {
						case smpp.CommandSubmitSm:

						case smpp.CommandEnquireLink:
						default:
							ui.WriteLineToEventBox("I don't know how to generate a message of that type")
						}
					case smppth.Help:
						ui.WriteLineToEventBox(textCommandProcessor.CommandTextHelp())
					}
				}
				//case incomingEvent := <-sharedAgentEventChannel:
			}
		}
	}()

	ui.StartRunning()

}

func initializeErrorLogger() *log.Logger {
	return log.New(os.Stderr, "", 0)
}

func fatalOutsideUI(msg string) {
	errLogger.Fatalln(msg)
}

func parseCommandLineOptions() (esmesOrSmscs string, yamlConfigFileName string) {
	if len(os.Args) != 4 || os.Args[1] != "run" || (os.Args[2] != "esmes" && os.Args[2] != "smscs") {
		errLogger.Fatalln(syntaxString())
	}

	return os.Args[2], os.Args[3]
}

func generateEsmeAgentGroup(esmes []*smppth.ESME, smscs []*smppth.SMSC) *smppth.AgentGroup {
	return nil
}

func generateSmscAgentGroup(esmes []*smppth.ESME, smscs []*smppth.SMSC) *smppth.AgentGroup {
	return nil
}

func panicOnErr(err error) {
	if err != nil {
		panic(err)
	}
}

func fatalOutsideUIOnErr(err error) {
	if err != nil {
		fatalOutsideUI(err.Error())
	}
}

func syntaxString() string {
	return path.Base(os.Args[0]) + " run esmes|smscs <config_yaml_file>"
}
