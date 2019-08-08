package main

import tui "github.com/marcusolsson/tui-go"

// TestHarnessTuiInterface provides a TUI based text interface.  It has a command
// entry box with basic EMACS-like control keys (^k, ^a, ^e, ^b) used with a bash
// shell.  Another box shows the last 10 commands entered.  A third box is used for
// general output from the application logic.
type TestHarnessTuiInterface struct {
	commandHistoryInsideBox *tui.Box
	commandHistoryBox       *tui.Box
	inputEntryWidget        *tui.Entry
	commandInputBox         *tui.Box
	eventOutputBox          *tui.Box
	uiRootBox               *tui.Box
	uiObject                tui.UI
	userInputStringChannel  chan string
	commandTextEntryQueue   *simpleStringCircularQueue
	commandIterator         *simpleCircularQueueIterator
}

// BuildUserInterface constructs the TUI interface elements
func BuildUserInterface() *TestHarnessTuiInterface {
	ui := &TestHarnessTuiInterface{
		userInputStringChannel: make(chan string),
		commandTextEntryQueue:  NewSimpleStringCircularBuffer(10),
		commandIterator:        nil,
	}

	ui.addCommandHistoryBox().andSetBoxHeightRowsCountTo(10)
	ui.addUserCommandInputBox()
	ui.addEventOutputBox()
	ui.createRootUIElement()
	ui.prepareToHandleUserInput()

	return ui
}

// UserInputStringCommandChannel retrieves a string channel that will contain user input
// provided in the command input box
func (ui *TestHarnessTuiInterface) UserInputStringCommandChannel() <-chan string {
	return ui.userInputStringChannel
}

// StartRunning launches the UI after its construction
func (ui *TestHarnessTuiInterface) StartRunning() {
	if err := ui.uiObject.Run(); err != nil {
		panicOnErr(err)
	}
}

func (ui *TestHarnessTuiInterface) addCommandHistoryBox() *TestHarnessTuiInterface {
	ui.commandHistoryInsideBox = tui.NewVBox()

	historyScroll := tui.NewScrollArea(ui.commandHistoryInsideBox)
	historyScroll.SetAutoscrollToBottom(true)

	ui.commandHistoryBox = tui.NewHBox(historyScroll)
	ui.commandHistoryBox.SetBorder(true)
	ui.commandHistoryBox.SetTitle("Command History")
	ui.commandHistoryBox.SetSizePolicy(tui.Maximum, tui.Maximum)

	return ui
}

func (ui *TestHarnessTuiInterface) andSetBoxHeightRowsCountTo(rowCount uint) *TestHarnessTuiInterface {
	for i := uint(0); i < rowCount; i++ {
		ui.commandHistoryInsideBox.Append(tui.NewHBox(tui.NewLabel("")))
	}

	return ui
}

func (ui *TestHarnessTuiInterface) addUserCommandInputBox() *TestHarnessTuiInterface {
	ui.inputEntryWidget = tui.NewEntry()
	ui.inputEntryWidget.SetFocused(true)
	ui.inputEntryWidget.SetSizePolicy(tui.Expanding, tui.Maximum)

	ui.commandInputBox = tui.NewHBox(ui.inputEntryWidget)
	ui.commandInputBox.SetBorder(true)
	ui.commandInputBox.SetSizePolicy(tui.Expanding, tui.Maximum)
	ui.commandInputBox.SetTitle("Enter Command")

	return ui
}

func (ui *TestHarnessTuiInterface) addEventOutputBox() *TestHarnessTuiInterface {
	ui.eventOutputBox = tui.NewHBox()
	ui.eventOutputBox.SetBorder(true)
	ui.eventOutputBox.SetTitle("Agent Events")
	ui.eventOutputBox.SetSizePolicy(tui.Expanding, tui.Expanding)

	return ui
}

func (ui *TestHarnessTuiInterface) createRootUIElement() *TestHarnessTuiInterface {
	ui.uiRootBox = tui.NewVBox(ui.commandHistoryBox, ui.commandInputBox, ui.eventOutputBox)

	var err error
	ui.uiObject, err = tui.New(ui.uiRootBox)
	panicOnErr(err)

	return ui
}

func (ui *TestHarnessTuiInterface) prepareToHandleUserInput() {
	ui.uiObject.SetKeybinding("Esc", func() { ui.uiObject.Quit() })

	ui.inputEntryWidget.OnSubmit(func(entry *tui.Entry) {
		textOfEnteredCommand := entry.Text()

		ui.addEntryToCommandHistory(textOfEnteredCommand)
		ui.commandTextEntryQueue.PutItemAtEnd(textOfEnteredCommand)
		ui.inputEntryWidget.SetText("")

		ui.commandIterator = nil

		ui.userInputStringChannel <- textOfEnteredCommand
	})

	ui.uiObject.SetKeybinding("Up", func() {
		if ui.commandIterator == nil {
			ui.commandIterator = ui.commandTextEntryQueue.GenerateNewIterator().PreviousStartsAtEnd()
		}
		iteratedCommandString := ui.commandIterator.Previous()
		if iteratedCommandString != "" {
			ui.inputEntryWidget.SetText(iteratedCommandString)
		}
	})

	ui.uiObject.SetKeybinding("Down", func() {
		if ui.commandIterator == nil {
			ui.commandIterator = ui.commandTextEntryQueue.GenerateNewIterator()
		}
		iteratedCommandString := ui.commandIterator.Next()
		if iteratedCommandString != "" {
			ui.inputEntryWidget.SetText(iteratedCommandString)
		}
	})
}

func (ui *TestHarnessTuiInterface) addEntryToCommandHistory(commandString string) {
	ui.commandHistoryInsideBox.Append(
		tui.NewHBox(
			tui.NewLabel(commandString),
		),
	)
}

type simpleStringCircularQueue struct {
	stringSlice             []string
	capacity                uint
	headIndex               uint
	indexOfNextInsert       uint
	indexOfLastSliceElement uint
	countOfItemsInQueue     uint
}

func NewSimpleStringCircularBuffer(capacity uint) *simpleStringCircularQueue {
	return &simpleStringCircularQueue{
		stringSlice:             make([]string, capacity),
		headIndex:               0,
		indexOfNextInsert:       0,
		indexOfLastSliceElement: capacity - 1,
		countOfItemsInQueue:     0,
	}
}

func (queue *simpleStringCircularQueue) PutItemAtEnd(item string) {
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

func (queue *simpleStringCircularQueue) HasNoItems() bool {
	return queue.countOfItemsInQueue == 0
}

func (queue *simpleStringCircularQueue) NumberOfItemsInTheQueue() uint {
	return queue.countOfItemsInQueue
}

func (queue *simpleStringCircularQueue) GetItemAtIndex(index uint) (item string, thereIsAnItemAtThatIndex bool) {
	if queue.countOfItemsInQueue == 0 || index > queue.countOfItemsInQueue-1 {
		return "", false
	}

	sliceIndexOfItem := queue.headIndex + index
	if sliceIndexOfItem > queue.indexOfLastSliceElement {
		sliceIndexOfItem -= (queue.indexOfLastSliceElement + 1)
	}

	return queue.stringSlice[sliceIndexOfItem], true
}

func (queue *simpleStringCircularQueue) GenerateNewIterator() *simpleCircularQueueIterator {
	return newSimpleStringCircularIterator(queue)
}

type simpleCircularQueueIterator struct {
	attachedQueue       *simpleStringCircularQueue
	currentIndexPtr     uint
	iterationHasStarted bool
	previousStartsAtEnd bool
}

func newSimpleStringCircularIterator(forQueue *simpleStringCircularQueue) *simpleCircularQueueIterator {
	return &simpleCircularQueueIterator{
		attachedQueue:       forQueue,
		currentIndexPtr:     0,
		iterationHasStarted: false,
		previousStartsAtEnd: false,
	}
}

func (iter *simpleCircularQueueIterator) PreviousStartsAtEnd() *simpleCircularQueueIterator {
	iter.previousStartsAtEnd = true
	return iter
}

func (iter *simpleCircularQueueIterator) Next() string {
	if iter.iterationHasStarted {
		if iter.attachedQueue.NumberOfItemsInTheQueue() > iter.currentIndexPtr+1 {
			iter.currentIndexPtr++
		}
	} else {
		if iter.attachedQueue.NumberOfItemsInTheQueue() > 0 {
			iter.currentIndexPtr = 0
			iter.iterationHasStarted = true
		}
	}

	if iter.iterationHasStarted {
		v, _ := iter.attachedQueue.GetItemAtIndex(iter.currentIndexPtr)
		return v
	}

	return ""
}

func (iter *simpleCircularQueueIterator) Previous() string {
	if iter.iterationHasStarted {
		if iter.currentIndexPtr > 0 {
			iter.currentIndexPtr--
		}
	} else {
		if iter.attachedQueue.NumberOfItemsInTheQueue() > 0 {
			if iter.previousStartsAtEnd {
				iter.currentIndexPtr = iter.attachedQueue.NumberOfItemsInTheQueue() - 1
			} else {
				iter.currentIndexPtr = 0
			}
			iter.iterationHasStarted = true
		}
	}

	if iter.iterationHasStarted {
		v, _ := iter.attachedQueue.GetItemAtIndex(iter.currentIndexPtr)
		return v
	}

	return ""
}
