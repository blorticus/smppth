package main

type eventChannelConcentrator struct {
}

func newEventChannelConcentrator() *eventChannelConcentrator {
	return &eventChannelConcentrator{}
}

func (concentrator *eventChannelConcentrator) addEventChannelForEsmeNamed(esmeName string, esmeListenerEventChannel <-chan *esmeListenerEvent) {

}

func (concentrator *eventChannelConcentrator) retrieveConcentrationChannel() <-chan *esmeListenerEvent {
	return nil
}

func (concentrator *eventChannelConcentrator) startReceivingEvents() {

}
