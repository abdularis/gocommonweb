package gocommonweb

// EventHandler is a callback to handle incoming event
type EventHandler interface {
	Handle(eventName string, payload string)
}

// Event abstraction for pub/sub mechanism and
// can be used as server to server event bus
type Event interface {
	Publish(eventName string, payload string) error
	Subscribe(eventName string, handler EventHandler) error
	Unsubscribe(eventName string)
}
