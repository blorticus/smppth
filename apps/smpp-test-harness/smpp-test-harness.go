package main

import (
	"fmt"
	"log"
	"os"
	"path"

	smppth "github.com/blorticus/smppth"
)

var errLogger *log.Logger
var debugLogger *log.Logger

func main() {
	errLogger = initializeErrorLogger()

	runAsEsmesOrSmscs, yamlConfigFileName := parseCommandLineOptions()

	esmes, smscs, err := smppth.NewApplicationConfigYamlReader().ParseFile(yamlConfigFileName)
	fatalOutsideUIOnErr(err)

	debugLogger = initializeDebugLogger(runAsEsmesOrSmscs)

	var agentGroup *smppth.AgentGroup
	if runAsEsmesOrSmscs == "esmes" {
		agentGroup = generateEsmeAgentGroup(esmes)
	} else {
		agentGroup = generateSmscAgentGroup(smscs)
	}
	agentGroup.AttachDebugLoggerWriter(debugLogger.Writer())

	ui := BuildUserInterface()
	commandInputTextChannel := ui.UserInputStringCommandChannel()

	application := smppth.NewStandardApplication().
		SetPduFactory(smppth.NewDefaultPduFactory()).
		SetOutputGenerator(smppth.NewStandardOutputGenerator()).
		SetAgentGroup(agentGroup).
		OnQuit(func() { ui.Exit() }).
		SetEventOutputWriter(ui)

	application.AttachEventChannel(agentGroup.SharedAgentEventChannel())

	go application.Start()
	go startListeningForUserCommands(commandInputTextChannel, ui, application)
	agentGroup.StartAllAgents()

	ui.StartRunning()
}

func startListeningForUserCommands(commandInputTextChannel <-chan string, ui *TestHarnessTextUI, app *smppth.StandardApplication) {
	textCommandProcessor := smppth.NewTextCommandProcessor()

	for {
		nextUserCommandText := <-commandInputTextChannel
		userCommandStruct, err := textCommandProcessor.ConvertCommandLineStringToUserCommand(nextUserCommandText)

		if err != nil {
			ui.WriteLineToEventBox(fmt.Sprintf("[ERROR] Invalid command (%s)", nextUserCommandText))
		} else {
			app.ReceiveNextCommand(userCommandStruct)
		}
	}

}

func initializeErrorLogger() *log.Logger {
	return log.New(os.Stderr, "", 0)
}

func initializeDebugLogger(forEsmesOrSmscs string) *log.Logger {
	debugFileHandle, err := os.OpenFile(fmt.Sprintf("/tmp/smpp-debug-%s.log", forEsmesOrSmscs), os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0640)
	fatalOutsideUIOnErr(err)

	return log.New(debugFileHandle, "(smpp-test-harness): ", 0)
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

func generateEsmeAgentGroup(esmes []*smppth.ESME) *smppth.AgentGroup {
	agents := make([]smppth.Agent, len(esmes))
	for i, v := range esmes {
		agents[i] = v
	}
	return smppth.NewAgentGroup(agents)
}

func generateSmscAgentGroup(smscs []*smppth.SMSC) *smppth.AgentGroup {
	agents := make([]smppth.Agent, len(smscs))
	for i, v := range smscs {
		agents[i] = v
	}
	return smppth.NewAgentGroup(agents)
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