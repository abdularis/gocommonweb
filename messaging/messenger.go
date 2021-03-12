package messaging

type MessageHandler func(packet *Packet)

// Messenger interface for real time data connection between users and servers
type Messenger interface {
	Send(message *OutPacket)
	Listen(eventName string, handler MessageHandler)
	TearDown()
}

// Packet message coming from client to server
type Packet struct {
	ID         string      `json:"id"`
	ReceiverID string      `json:"receiver"`
	EventName  string      `json:"event_name"`
	Data       interface{} `json:"data"`
}

// OutPacket message out from server to client
type OutPacket struct {
	ID         string      `json:"id"`
	SenderID   string      `json:"sender"`
	ReceiverID string      `json:"receiver"`
	EventName  string      `json:"event_name"`
	Data       interface{} `json:"data"`
}
