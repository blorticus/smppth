package main

type applicationControlBroker struct {
}

func newApplicationControlBroker() *applicationControlBroker {
	return &applicationControlBroker{}
}

func (broker *applicationControlBroker) retrieveSendMessageChannel() <-chan *messageDescriptor {
	return nil
}

func (broker *applicationControlBroker) startListening() {
}
