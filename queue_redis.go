package gocommonweb

import (
	"github.com/gocraft/work"
	"github.com/gomodule/redigo/redis"
	"github.com/sirupsen/logrus"
)

type queueImpl struct {
	enqueuer *work.Enqueuer
	worker   *work.WorkerPool
}

type contextSample struct{}

// NewQueueRedis create new queue backed by redis
func NewQueueRedis(
	appName string,
	redisAddress string,
	redisPassword string) Queue {

	// Make a redis pool
	var redisPool = &redis.Pool{
		MaxActive: 5,
		MaxIdle:   5,
		Wait:      true,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", redisAddress, redis.DialPassword(redisPassword))
		},
	}

	enqueuer := work.NewEnqueuer(appName, redisPool)
	pool := work.NewWorkerPool(contextSample{}, 5, appName, redisPool)

	return &queueImpl{
		enqueuer: enqueuer,
		worker:   pool,
	}
}

func (q *queueImpl) AddJob(jobName string, payload string) error {
	_, err := q.enqueuer.Enqueue(jobName, work.Q{"payload": payload})
	return err
}

func (q *queueImpl) AddDelayedJob(jobName string, payload string, delaySecs uint) error {
	_, err := q.enqueuer.EnqueueIn(jobName, int64(delaySecs), work.Q{"payload": payload})
	return err
}

func (q *queueImpl) AddJobHandler(jobName string, handler JobHandler) {
	q.worker.Job(jobName, func(job *work.Job) error {
		payload := job.ArgString("payload")
		return handler.Handle(job.Name, payload)
	})
}

func (q *queueImpl) Start() {
	q.worker.Start()
}

func (q *queueImpl) Close() {
	q.worker.Stop()
	if err := q.enqueuer.Pool.Close(); err != nil {
		logrus.Errorf("err closing queue worker pool: %s", err)
	}
}
