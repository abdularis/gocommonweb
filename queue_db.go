package gocommonweb

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	statusWaiting    = "waiting"
	statusProcessing = "processing"
	statusComplete   = "complete"
)

type job struct {
	gorm.Model
	JobName     string
	Payload     string
	Status      string    `gorm:"index"`
	RunAt       time.Time `gorm:"index"`
	LastVisited time.Time `gorm:"index"`
}

type queueDB struct {
	db              *gorm.DB
	startMutex      sync.Mutex
	running         bool
	workerCount     int
	handlers        map[string]JobHandler
	handlerMutex    sync.Mutex
	stopRequeueChan chan bool
	stopWorkersChan []chan bool
}

// NewQueueDB create job queue backend by database
func NewQueueDB(db *gorm.DB, workerCount int) (Queue, error) {
	if workerCount <= 0 {
		panic(fmt.Errorf("queue job worker count must not be <= 0"))
	}

	err := db.AutoMigrate(&job{})
	if err != nil {
		return nil, err
	}

	return &queueDB{
		db:              db,
		handlers:        make(map[string]JobHandler),
		startMutex:      sync.Mutex{},
		handlerMutex:    sync.Mutex{},
		workerCount:     workerCount,
		running:         false,
		stopRequeueChan: make(chan bool),
	}, nil
}

func (q *queueDB) AddJob(jobName string, payload string) error {
	return q.AddDelayedJob(jobName, payload, 0)
}

func (q *queueDB) AddDelayedJob(jobName string, payload string, delaySecs uint) error {
	runAt := time.Now()
	if delaySecs > 0 {
		runAt = time.Now().Add(time.Second * time.Duration(delaySecs))
	}
	j := job{
		JobName:     jobName,
		Payload:     payload,
		Status:      statusWaiting,
		RunAt:       runAt,
		LastVisited: time.Now(),
	}
	return q.db.Create(&j).Error
}

func (q *queueDB) AddJobHandler(jobName string, handler JobHandler) {
	q.handlerMutex.Lock()
	defer q.handlerMutex.Unlock()
	q.handlers[jobName] = handler
}

func (q *queueDB) Start() {
	q.startMutex.Lock()
	defer q.startMutex.Unlock()
	if !q.running {
		q.running = true
		for i := 0; i < q.workerCount; i++ {
			stopChan := make(chan bool)
			q.stopWorkersChan = append(q.stopWorkersChan, stopChan)
			go q.startWorkerLoop(stopChan)
		}
		go q.startJobRequeueLoop()
		logrus.Info("[qdb] worker and scheduler running...")
	}
}

func (q *queueDB) Close() {
	q.stopRequeueChan <- true
	for _, stopChan := range q.stopWorkersChan {
		stopChan <- true
	}
}

func (q *queueDB) startJobRequeueLoop() {
	timer := time.NewTimer(time.Second)
	for {
		select {
		case <-timer.C:
			err := q.db.Transaction(func(tx *gorm.DB) error {
				var res job
				err := tx.
					Clauses(clause.Locking{Strength: "UPDATE"}).
					Where("status = ? AND last_visited <= ?", statusProcessing, time.Now().Add(-maxLastVisitedDuration)).
					First(&res).Error
				if err != nil {
					return err
				}

				err = tx.Model(&job{}).
					Where("id = ?", res.ID).
					Updates(job{Status: statusWaiting, LastVisited: time.Now()}).Error

				if err == nil {
					logrus.Debugf("[qdb] requeue job %s - %d\n", res.JobName, res.ID)
				}
				return err
			})

			if err == gorm.ErrRecordNotFound {
				timer = time.NewTimer(time.Minute * 10)
			} else {
				timer = time.NewTimer(time.Second * 10)
			}
			break
		case <-q.stopRequeueChan:
			logrus.Info("[qdb] requeue loop stopped")
			return
		}
	}
}

func (q *queueDB) startWorkerLoop(stop chan bool) {
	timer := time.NewTimer(time.Second)
	for {
		select {
		case <-timer.C:
			j, err := q.findJobToProcess()
			if err == nil {
				if handler, ok := q.handlers[j.JobName]; ok {
					visitor := jobVisitor{stopChannel: make(chan bool)}
					go visitor.startVisiting(func() {
						_ = q.updateLastVisited(j.ID)
					})
					err := handler.Handle(j.JobName, j.Payload)
					visitor.stop()

					if err == nil {
						_ = q.updateJobStatus(j.ID, statusComplete)
					} else {
						_ = q.updateJobStatus(j.ID, statusWaiting)
					}
				} else {
					_ = q.updateJobStatus(j.ID, statusWaiting)
				}

				timer = time.NewTimer(time.Second * time.Duration(rand.Intn(10)))
			} else {
				var deSyncDelay time.Duration
				if err == gorm.ErrRecordNotFound {
					deSyncDelay = time.Second * time.Duration(rand.Intn(30))
				}
				timer = time.NewTimer(time.Second*10 + deSyncDelay)
			}
			break
		case <-stop:
			logrus.Info("[qdb] worker loop stopped")
			return
		}
	}
}

func (q *queueDB) findJobToProcess() (*job, error) {
	var res job
	tx := q.db.Begin()
	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("status = ? AND (run_at <= ? OR run_at IS NULL)", statusWaiting, time.Now()).
		Limit(1).
		First(&res).Error
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	err = tx.Model(&job{}).
		Where("id = ?", res.ID).
		Updates(job{Status: statusProcessing, LastVisited: time.Now()}).Error
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	tx.Commit()
	return &res, nil
}

func (q *queueDB) updateJobStatus(jobID uint, status string) error {
	return q.db.Transaction(func(tx *gorm.DB) error {
		return tx.Model(&job{}).
			Where("id = ?", jobID).
			Update("status", status).Error
	})
}

func (q *queueDB) updateLastVisited(jobID uint) error {
	return q.db.Transaction(func(tx *gorm.DB) error {
		return tx.Model(&job{}).
			Where("id = ?", jobID).
			Update("last_visited", time.Now()).Error
	})
}

// jobVisitor act as heartbeat to inform that this job is still processing
type jobVisitor struct {
	stopChannel chan bool
}

func (v *jobVisitor) stop() {
	v.stopChannel <- true
}

func (v *jobVisitor) startVisiting(onVisiting func()) {
	for {
		ticker := time.NewTicker(visitingInterval)
		select {
		case <-v.stopChannel:
			ticker.Stop()
			return
		case <-ticker.C:
			onVisiting()
			break
		}
	}
}

const (
	visitingInterval       = time.Second * 10
	maxLastVisitedDuration = time.Minute * 15
)
