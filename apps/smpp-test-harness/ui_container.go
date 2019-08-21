package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

// TestHarnessTextUI provides a TUI based text interface.  It has a command
// entry box with basic EMACS-like control keys (^k, ^a, ^e, ^b) used with a bash
// shell.  Another box shows the last 10 commands entered.  A third box is used for
// general output from the application logic.
type TestHarnessTextUI struct {
	tviewApplication           *tview.Application
	userCommandHistoryTextView *tview.TextView
	userCommandInputField      *tview.InputField
	eventOutputTextView        *tview.TextView
	userInputStringChannel     chan string
	commandReadlineHistory     *readlineHistory
	debugLogger                *log.Logger
}

// BuildUserInterface constructs the TUI interface elements
func BuildUserInterface() *TestHarnessTextUI {
	ui := &TestHarnessTextUI{
		userInputStringChannel: make(chan string),
		commandReadlineHistory: newReadlineHistory(200),
		debugLogger:            nil,
	}

	ui.createTviewApplication().
		createCommandHistoryTextView().
		createCommandInputField().
		createEventOutputTextView().
		composeIntoUIGrid().
		addGlobalKeybindings()

	return ui
}

func (ui *TestHarnessTextUI) createTviewApplication() *TestHarnessTextUI {
	ui.tviewApplication = tview.NewApplication()
	return ui
}

func (ui *TestHarnessTextUI) createCommandHistoryTextView() *TestHarnessTextUI {
	ui.userCommandHistoryTextView = tview.NewTextView()
	ui.userCommandHistoryTextView.
		SetBorder(true).
		SetTitle("Command History").
		SetTitleAlign(tview.AlignLeft)

	return ui
}

func (ui *TestHarnessTextUI) createCommandInputField() *TestHarnessTextUI {
	ui.userCommandInputField = tview.NewInputField().
		SetLabel("Enter Command> ").
		SetFieldBackgroundColor(tcell.ColorBlack).
		SetFieldWidth(100).
		SetDoneFunc(func(key tcell.Key) {
			if key == tcell.KeyEnter {
				userProvidedCommandText := ui.userCommandInputField.GetText()

				ui.commandReadlineHistory.AddItem(userProvidedCommandText)
				ui.commandReadlineHistory.ResetIteration()
				if ui.userCommandHistoryTextView.GetText(false) == "" {
					fmt.Fprintf(ui.userCommandHistoryTextView, userProvidedCommandText)
				} else {
					fmt.Fprintf(ui.userCommandHistoryTextView, "\n%s", userProvidedCommandText)
				}
				ui.sendNextInputCommandToChannelWithoutBlocking(userProvidedCommandText)
				ui.userCommandInputField.SetText("")
				ui.tviewApplication.Draw()
			}
		})

	ui.userCommandInputField.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyUp {
			if commandFromHistory, thereWereMoreCommandsInHistory := ui.commandReadlineHistory.Up(); thereWereMoreCommandsInHistory {
				ui.userCommandInputField.SetText(commandFromHistory)
			}
			return nil
		} else if event.Key() == tcell.KeyDown {
			if commandFromHistory, wasNotYetAtFirstCommand := ui.commandReadlineHistory.Down(); wasNotYetAtFirstCommand {
				ui.userCommandInputField.SetText(commandFromHistory)
			}
			return nil
		} else {
			return event
		}
	})

	return ui
}

func (ui *TestHarnessTextUI) sendNextInputCommandToChannelWithoutBlocking(commandText string) {
	go func() { ui.userInputStringChannel <- commandText }()
}

func (ui *TestHarnessTextUI) createEventOutputTextView() *TestHarnessTextUI {
	ui.eventOutputTextView = tview.NewTextView()

	ui.eventOutputTextView.
		SetBorder(true).
		SetTitle("Events").
		SetTitleAlign(tview.AlignLeft)

	return ui
}

func (ui *TestHarnessTextUI) composeIntoUIGrid() *TestHarnessTextUI {
	grid := tview.NewGrid().
		SetRows(10, 1, 0).
		SetColumns(0)

	grid.AddItem(ui.userCommandHistoryTextView, 0, 0, 1, 1, 0, 0, false).
		AddItem(ui.userCommandInputField, 1, 0, 1, 1, 0, 0, true).
		AddItem(ui.eventOutputTextView, 2, 0, 1, 1, 0, 0, false)

	ui.tviewApplication.SetRoot(grid, true)

	return ui
}

func (ui *TestHarnessTextUI) addGlobalKeybindings() *TestHarnessTextUI {
	ui.tviewApplication.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTab:
			switch ui.tviewApplication.GetFocus() {
			case ui.userCommandHistoryTextView:
				ui.tviewApplication.SetFocus(ui.userCommandInputField)
			case ui.userCommandInputField:
				ui.tviewApplication.SetFocus(ui.eventOutputTextView)
			default:
				ui.tviewApplication.SetFocus(ui.userCommandHistoryTextView)
			}
			return nil
		case tcell.KeyESC:
			ui.Exit()
		case tcell.KeyCtrlQ:
			ui.Exit()
		}

		return event
	})

	return ui
}

// UserInputStringCommandChannel retrieves a string channel that will contain user input
// provided in the command input box
func (ui *TestHarnessTextUI) UserInputStringCommandChannel() <-chan string {
	return ui.userInputStringChannel
}

// WriteLineToEventBox writes a string to the UI event box, treating it as a line
// This means the next string line written will have a newline between it (the next
// string, that is) and the line written here
func (ui *TestHarnessTextUI) WriteLineToEventBox(line string) {
	if ui.eventOutputTextView.GetText(false) == "" {
		fmt.Fprintf(ui.eventOutputTextView, line)
	} else {
		fmt.Fprintf(ui.eventOutputTextView, "\n%s", line)
	}
	ui.tviewApplication.Draw()
}

func (ui *TestHarnessTextUI) Write(p []byte) (n int, err error) {
	ui.WriteLineToEventBox(string(p))
	return len(p), nil
}

// AttachDebugLogger attaches a logger to the UI object.  If a logger is
// attached, debug logging messages will be emitted
func (ui *TestHarnessTextUI) AttachDebugLogger(logger *log.Logger) {
	ui.debugLogger = logger
}

// StartRunning launches the UI after its construction
func (ui *TestHarnessTextUI) StartRunning() error {
	if err := ui.tviewApplication.Run(); err != nil {
		return err
	}
	return nil
}

// Exit stops the application and exits with a status of zero
func (ui *TestHarnessTextUI) Exit() {
	ui.tviewApplication.Stop()
	os.Exit(0)
}

func (ui *TestHarnessTextUI) debugLogPrintf(format string, v ...interface{}) {
	if ui.debugLogger != nil {
		format += "\n"
		ui.debugLogger.Printf(format, v...)
	}
}

// SimpleStringCircularQueue is a simple circular buffer of strings
type SimpleStringCircularQueue struct {
	stringSlice             []string
	capacity                uint
	headIndex               uint
	indexOfNextInsert       uint
	indexOfLastSliceElement uint
	countOfItemsInQueue     uint
}

// NewSimpleStringCircularBuffer creates a simple circular buffer of strings, with
// the queue holding up to 'capacity' number of items.
func NewSimpleStringCircularBuffer(capacity uint) *SimpleStringCircularQueue {
	return &SimpleStringCircularQueue{
		stringSlice:             make([]string, capacity),
		headIndex:               0,
		indexOfNextInsert:       0,
		indexOfLastSliceElement: capacity - 1,
		countOfItemsInQueue:     0,
	}
}

// PutItemAtEnd places an item at the end of the circular queue
func (queue *SimpleStringCircularQueue) PutItemAtEnd(item string) {
	if queue.countOfItemsInQueue > 0 && queue.indexOfNextInsert == queue.headIndex {
		if queue.headIndex == queue.indexOfLastSliceElement {
			queue.headIndex = 0
		} else {
			queue.headIndex++
		}
	}

	queue.stringSlice[queue.indexOfNextInsert] = item

	if queue.indexOfNextInsert == queue.indexOfLastSliceElement {
		queue.indexOfNextInsert = 0
	} else {
		queue.indexOfNextInsert++
	}

	if queue.countOfItemsInQueue < uint(len(queue.stringSlice)) {
		queue.countOfItemsInQueue++
	}
}

// IsEmpty returns true if the queue has no items in it; false otherwise
func (queue *SimpleStringCircularQueue) IsEmpty() bool {
	return queue.countOfItemsInQueue == 0
}

// IsNotEmpty returns true if the queue has at least one item in it; false otherwise
func (queue *SimpleStringCircularQueue) IsNotEmpty() bool {
	return queue.countOfItemsInQueue != 0
}

// NumberOfItemsInTheQueue returns a count of the number of items in the queue
func (queue *SimpleStringCircularQueue) NumberOfItemsInTheQueue() uint {
	return queue.countOfItemsInQueue
}

// GetItemAtIndex retrieves the string at the specified index (0 is the first item)
func (queue *SimpleStringCircularQueue) GetItemAtIndex(index uint) (item string, thereIsAnItemAtThatIndex bool) {
	if queue.countOfItemsInQueue == 0 || index > queue.countOfItemsInQueue-1 {
		return "", false
	}

	sliceIndexOfItem := queue.headIndex + index
	if sliceIndexOfItem > queue.indexOfLastSliceElement {
		sliceIndexOfItem -= (queue.indexOfLastSliceElement + 1)
	}

	return queue.stringSlice[sliceIndexOfItem], true
}

type readlineHistory struct {
	attachedQueue           *SimpleStringCircularQueue
	indexOfLastItemReturned uint
	iterationHasStarted     bool
}

func newReadlineHistory(maximumHistoryEntries uint) *readlineHistory {
	return &readlineHistory{
		attachedQueue:           NewSimpleStringCircularBuffer(maximumHistoryEntries),
		indexOfLastItemReturned: 0,
		iterationHasStarted:     false,
	}
}

func (history *readlineHistory) Up() (historyItem string, wasNotYetAtTopOfList bool) {
	if history.attachedQueue.IsNotEmpty() {
		if history.iterationHasStarted {
			if history.iteratorIsNotAtStartOfHistoryList() {
				v, _ := history.attachedQueue.GetItemAtIndex(history.indexOfLastItemReturned - 1)
				history.indexOfLastItemReturned--
				return v, true
			}
		} else {
			history.iterationHasStarted = true
			v, _ := history.attachedQueue.GetItemAtIndex(history.attachedQueue.NumberOfItemsInTheQueue() - 1)
			history.indexOfLastItemReturned = history.attachedQueue.NumberOfItemsInTheQueue() - 1
			return v, true
		}
	}

	return "", false
}

func (history *readlineHistory) Down() (historyItem string, wasNotYetAtBottomOfList bool) {
	if history.attachedQueue.IsNotEmpty() {
		if history.iterationHasStarted {
			if history.iteratorIsNotAtEndOfHistoryList() {
				v, _ := history.attachedQueue.GetItemAtIndex(history.indexOfLastItemReturned + 1)
				history.indexOfLastItemReturned++
				return v, true
			}
		}
	}

	return "", false
}

func (history *readlineHistory) iteratorIsNotAtEndOfHistoryList() bool {
	return history.attachedQueue.NumberOfItemsInTheQueue() > history.indexOfLastItemReturned+1
}

func (history *readlineHistory) iteratorIsNotAtStartOfHistoryList() bool {
	return history.indexOfLastItemReturned != 0
}

func (history *readlineHistory) ResetIteration() {
	history.indexOfLastItemReturned = 0
	history.iterationHasStarted = false
}

func (history *readlineHistory) AddItem(item string) {
	history.attachedQueue.PutItemAtEnd(item)
}
