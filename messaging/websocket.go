package messaging

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

var wsupgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// WebSocketUpgrade handle http to websocket upgrade
func WebSocketUpgrade(messenger Messenger, clientID string, w http.ResponseWriter, r *http.Request) {
	conn, err := wsupgrader.Upgrade(w, r, nil)

	if err != nil {
		logrus.Debug("Failed to set websocket upgrade: %+v", err)
		return
	}

	if impl, ok := messenger.(*messengerImpl); ok {
		if err := impl.AddClient(clientID, conn); err != nil {
			logrus.Debugf("err adding new socket client: %s", err)
		}
	} else {
		logrus.Debug("messenger implementation is not *messengerImpl")
		_ = conn.Close()
	}
}
