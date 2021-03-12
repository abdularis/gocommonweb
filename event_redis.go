package gocommonweb

import (
	"context"
	"sync"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
)

type eventSubscription struct {
	eventName   string
	pubsub      *redis.PubSub
	handler     EventHandler
	stopChannel chan bool
}

type eventRedis struct {
	rds           *redis.Client
	subscriptions map[string]*eventSubscription
	mu            sync.Mutex
}

// NewEventRedis create new event bus using redis
func NewEventRedis(redisClient *redis.Client) Event {
	return &eventRedis{
		rds:           redisClient,
		subscriptions: make(map[string]*eventSubscription),
		mu:            sync.Mutex{},
	}
}

func (e *eventRedis) Publish(eventName string, payload string) error {
	return e.rds.Publish(context.Background(), eventName, payload).Err()
}

func (e *eventRedis) Subscribe(eventName string, handler EventHandler) error {
	pubsub := e.rds.Subscribe(context.Background(), eventName)
	receiveChannel := pubsub.Channel()

	subscription := eventSubscription{
		eventName:   eventName,
		pubsub:      pubsub,
		handler:     handler,
		stopChannel: make(chan bool),
	}

	e.mu.Lock()
	e.subscriptions[eventName] = &subscription
	e.mu.Unlock()

	go func() {
		for {
			select {
			case msg, ok := <-receiveChannel:
				if !ok {
					logrus.Debug("[Event] receiving channel no ok, stops receive event loop")
					return
				}
				subscription.handler.Handle(msg.Channel, msg.Payload)
			case <-subscription.stopChannel:
				logrus.Debug("[Event] intentionally stops receive loop")
				return
			}
		}
	}()

	return nil
}

func (e *eventRedis) Unsubscribe(eventName string) {
	if subscription, ok := e.subscriptions[eventName]; ok {
		if subscription != nil {
			e.mu.Lock()
			delete(e.subscriptions, eventName)
			e.mu.Unlock()

			subscription.stopChannel <- true
			_ = subscription.pubsub.Close()
		}
	}
}
