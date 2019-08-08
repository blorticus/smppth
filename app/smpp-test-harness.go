package main

import (
	"log"
	"os"
	"path"
	"smppth"
)

var errLogger *log.Logger

func main() {
	errLogger = initializeErrorLogger()

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

	ui := BuildUserInterface()
	commandInputTextChannel := ui.UserInputStringCommandChannel()

	// agentGroup.StartAllAgents()

	go func() {
		for {
			<-commandInputTextChannel
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
