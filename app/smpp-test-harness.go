package main

import (
	"smppth"

	"github.com/marcusolsson/tui-go"
)

func main() {
	// runAsEsmesOrSmscs, yamlConfigFileName := parseCommandLineOptions()

	// esmes, smscs := processYamlConfigFile(yamlConfigFileName)

	// var agentGroup *smppth.AgentGroup
	// if runAsEsmesOrSmscs == "esmes" {
	// 	agentGroup = generateEsmeAgentGroup(esmes, smscs)
	// } else {
	// 	agentGroup = generateSmscAgentGroup(esmes, smscs)
	// }

	// outputMessageGenerator := smppth.NewGeneratorOfStandardOutputMessages()
	// textCommandProcessor := smppth.NewStandardTextCommandProcessor()

	// agentGroup.StartAllAgents()

	ui := buildUserInterface()
	ui.prepareToHandleUserInput()
	ui.prepareToProduceOutput()

	ui.startRunning()
}

func parseCommandLineOptions() (esmesOrSmscs string, yamlConfigFileName string) {
	return "", ""
}

func processYamlConfigFile(filename string) (esmes []*smppth.ESME, smscs []*smppth.SMSC) {
	return nil, nil
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

type localTmuiContainer struct {
	commandHistoryInsideBox *tui.Box
	commandHistoryBox       *tui.Box
	commandInputBox         *tui.Box
	eventOutputBox          *tui.Box
	uiRootBox               *tui.Box
	uiObject                tui.UI
}

func buildUserInterface() *localTmuiContainer {
	ui := &localTmuiContainer{}

	ui.addCommandHistoryBox().andSetBoxHeightRowsCountTo(10)
	ui.addUserCommandInputBox()
	ui.addEventOutputBox()
	ui.createRootUIElement()

	return ui
}

func (ui *localTmuiContainer) addCommandHistoryBox() *localTmuiContainer {
	ui.commandHistoryInsideBox = tui.NewVBox()

	historyScroll := tui.NewScrollArea(ui.commandHistoryInsideBox)
	historyScroll.SetAutoscrollToBottom(true)

	ui.commandHistoryBox = tui.NewHBox(historyScroll)
	ui.commandHistoryBox.SetBorder(true)
	ui.commandHistoryBox.SetTitle("Command History")
	ui.commandHistoryBox.SetSizePolicy(tui.Maximum, tui.Maximum)

	return ui
}

func (ui *localTmuiContainer) andSetBoxHeightRowsCountTo(rowCount uint) *localTmuiContainer {
	for i := uint(0); i < rowCount; i++ {
		ui.commandHistoryInsideBox.Append(tui.NewHBox(tui.NewLabel("")))
	}

	return ui
}

func (ui *localTmuiContainer) addUserCommandInputBox() *localTmuiContainer {
	inputEntryWidget := tui.NewEntry()
	inputEntryWidget.SetFocused(true)
	inputEntryWidget.SetSizePolicy(tui.Expanding, tui.Maximum)

	ui.commandInputBox = tui.NewHBox(inputEntryWidget)
	ui.commandInputBox.SetBorder(true)
	ui.commandInputBox.SetSizePolicy(tui.Expanding, tui.Maximum)
	ui.commandInputBox.SetTitle("Enter Command")

	return ui
}

func (ui *localTmuiContainer) addEventOutputBox() *localTmuiContainer {
	ui.eventOutputBox = tui.NewHBox()
	ui.eventOutputBox.SetBorder(true)
	ui.eventOutputBox.SetTitle("Agent Events")
	ui.eventOutputBox.SetSizePolicy(tui.Expanding, tui.Expanding)

	return ui
}

func (ui *localTmuiContainer) createRootUIElement() *localTmuiContainer {
	ui.uiRootBox = tui.NewVBox(ui.commandHistoryBox, ui.commandInputBox, ui.eventOutputBox)

	var err error
	ui.uiObject, err = tui.New(ui.uiRootBox)
	panicOnErr(err)

	return ui
}

func (ui *localTmuiContainer) prepareToHandleUserInput() *localTmuiContainer {
	ui.uiObject.SetKeybinding("Esc", func() { ui.uiObject.Quit() })

	return ui
}

func (ui *localTmuiContainer) prepareToProduceOutput() *localTmuiContainer {
	return ui
}

func (ui *localTmuiContainer) startRunning() {
	if err := ui.uiObject.Run(); err != nil {
		panicOnErr(err)
	}
}
