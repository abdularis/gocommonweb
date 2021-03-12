package messaging

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
	"sync"
)

type (
	connectionInfo struct {
		clientID string
		serverID string
	}

	globalConnectionInfo interface {
		Get(sessionID string) (connectionInfo, error)
		Put(info connectionInfo) error
		Remove(userSerial string) error
		RemoveAll(connections *localConnections) error
	}

	connection struct {
		socket   *websocket.Conn
		clientID string
		writeMtx sync.Mutex
	}

	localConnections struct {
		mutex       sync.RWMutex
		connections map[string]*connection
	}
)

func newLocalConnectionStore() localConnections {
	return localConnections{
		connections: make(map[string]*connection),
	}
}

func (l *localConnections) Put(key string, client *connection) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.connections[key] = client
}

func (l *localConnections) Get(key string) *connection {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	return l.connections[key]
}

func (l *localConnections) Remove(key string) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	delete(l.connections, key)
}

func (l *localConnections) IterateAndPop(callback func(key string, value *connection)) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	for clientID, conn := range l.connections {
		delete(l.connections, clientID)
		callback(clientID, conn)
	}
}

func (l *localConnections) Iterate(callback func(key string, value *connection)) {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	for k, v := range l.connections {
		callback(k, v)
	}
}

type globalConnectionInfoImpl struct {
	redisClient *redis.Client
	serverID    string
}

// NewGlobalSessionRedis instantiate global session backed by redis
func newGlobalConnectionInfo(client *redis.Client, serverID string) globalConnectionInfo {
	return &globalConnectionInfoImpl{redisClient: client, serverID: serverID}
}

func createUserInfoKey(userSerial string) string {
	return fmt.Sprintf("messaging:sessions:%s", userSerial)
}

const fieldUserSerial = "cid"
const fieldServerID = "sid"

func (r *globalConnectionInfoImpl) Get(userSerial string) (connectionInfo, error) {
	key := createUserInfoKey(userSerial)
	val, err := r.redisClient.HMGet(context.Background(), key, fieldUserSerial, fieldServerID).Result()
	if err != nil {
		return connectionInfo{}, err
	}

	if val[0] == nil || val[1] == nil {
		return connectionInfo{}, fmt.Errorf("client info %s not found", userSerial)
	}

	return connectionInfo{
		clientID: val[0].(string),
		serverID: val[1].(string),
	}, nil
}

func (r *globalConnectionInfoImpl) Put(info connectionInfo) error {
	key := createUserInfoKey(info.clientID)
	return r.redisClient.HMSet(context.Background(), key, map[string]interface{}{
		fieldUserSerial: info.clientID,
		fieldServerID:   info.serverID,
	}).Err()
}

func (r *globalConnectionInfoImpl) Remove(userSerial string) error {
	return r.redisClient.Del(context.Background(), createUserInfoKey(userSerial)).Err()
}

func (r *globalConnectionInfoImpl) RemoveAll(connections *localConnections) error {
	pipe := r.redisClient.Pipeline()
	connections.Iterate(func(key string, value *connection) {
		pipe.Del(context.Background(), createUserInfoKey(value.clientID))
	})
	_, err := pipe.Exec(context.Background())
	return err
}
