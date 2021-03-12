package messaging

import (
	"encoding/json"
	"gocommonweb"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"

	"github.com/gorilla/websocket"
)

type messengerImpl struct {
	serverID             string
	localConnections     localConnections
	globalConnectionInfo globalConnectionInfo
	event                gocommonweb.Event
	handlers             map[string]MessageHandler
	handlerMutex         sync.Mutex
}

// NewMessenger create new instance for Messenger implementation
func NewMessenger(serverID string, redisClient *redis.Client, event gocommonweb.Event) Messenger {
	messenger := &messengerImpl{
		serverID:             serverID,
		localConnections:     newLocalConnectionStore(),
		globalConnectionInfo: newGlobalConnectionInfo(redisClient, serverID),
		event:                event,
		handlers:             make(map[string]MessageHandler),
		handlerMutex:         sync.Mutex{},
	}
	go messenger.subscribeToEvent()
	return messenger
}

const maxRetrySubscribe = 5

func (m *messengerImpl) subscribeToEvent() {
	for i := 0; i < maxRetrySubscribe; i++ {
		if err := m.event.Subscribe(m.serverID, m); err != nil {
			time.Sleep(time.Second * 5)
		} else {
			return
		}
	}
}

func (m *messengerImpl) Send(message *OutPacket) {
	m.sendMessageImpl(message)
}

func (m *messengerImpl) Listen(eventName string, handler MessageHandler) {
	m.handlerMutex.Lock()
	defer m.handlerMutex.Unlock()
	m.handlers[eventName] = handler
}

func (m *messengerImpl) TearDown() {
	logrus.Infof("Tearing down messenger client connections, conn num: %d", len(m.localConnections.connections))
	_ = m.globalConnectionInfo.RemoveAll(&m.localConnections)
	// TODO re-check if this is called after servers graceful shutdown the it's not necessary
	m.localConnections.IterateAndPop(func(key string, value *connection) {
		_ = value.socket.Close()
	})
}

func (m *messengerImpl) AddClient(clientID string, socket *websocket.Conn) error {
	clientConn := connection{
		socket:   socket,
		clientID: clientID,
		writeMtx: sync.Mutex{},
	}

	clientInfo := connectionInfo{
		clientID: clientConn.clientID,
		serverID: m.serverID,
	}

	if err := m.globalConnectionInfo.Put(clientInfo); err != nil {
		return err
	} else {
		m.localConnections.Put(clientConn.clientID, &clientConn)
		go m.startClientReadLoop(&clientConn)
	}
	return nil
}

func (m *messengerImpl) Handle(_ string, payload string) {
	var pkt OutPacket
	if err := json.Unmarshal([]byte(payload), &pkt); err != nil {
		return
	}

	receiver := m.localConnections.Get(pkt.ReceiverID)
	if receiver != nil {
		// TODO maybe run on different go routine
		m.pushMessage(receiver, &pkt)
	}
}

func (m *messengerImpl) startClientReadLoop(client *connection) {
	for {
		_, msg, err := client.socket.ReadMessage()
		if err != nil {
			m.removeClient(client.clientID)
			logrus.Debugf("error while reading websocket pkt: %s\n", err.Error())
			break
		}

		var pkt Packet
		err = json.Unmarshal(msg, &pkt)
		if err != nil {
			logrus.Debugf("packet parsing error from %s", client.clientID)
			continue
		}

		if validateIncomingPacket(&pkt) {
			if handler, ok := m.handlers[pkt.EventName]; ok {
				go handler(&pkt)
			}
		}
	}
}

func (m *messengerImpl) sendMessageImpl(pkt *OutPacket) {
	if !validateOutgoingPacket(pkt) {
		return
	}

	receiver := m.localConnections.Get(pkt.ReceiverID)
	if receiver != nil {
		// Receiver is online on this server
		m.pushMessage(receiver, pkt)
	} else {
		clientInfo, err := m.globalConnectionInfo.Get(pkt.ReceiverID)
		if err == nil {
			if messageData, err := json.Marshal(pkt); err == nil {
				// Receiver is online on another server
				_ = m.event.Publish(clientInfo.serverID, string(messageData))
			}
		}
		// If goes here then receiver is offline
	}
}

func (m *messengerImpl) removeClient(userSerial string) {
	_ = m.globalConnectionInfo.Remove(userSerial)
	client := m.localConnections.Get(userSerial)
	if client != nil {
		m.localConnections.Remove(userSerial)
		_ = client.socket.Close()
	}
}

func (m *messengerImpl) pushMessage(receiver *connection, pkt *OutPacket) {
	data, err := json.Marshal(pkt)
	if err != nil {
		return
	}

	receiver.writeMtx.Lock()
	_ = receiver.socket.WriteMessage(websocket.TextMessage, data)
	receiver.writeMtx.Unlock()
}

func validateOutgoingPacket(pkt *OutPacket) bool {
	return pkt.ID != "" &&
		pkt.SenderID != "" &&
		pkt.ReceiverID != "" &&
		pkt.EventName != ""
}

func validateIncomingPacket(message *Packet) bool {
	return message.ID != "" &&
		message.EventName != ""
}
