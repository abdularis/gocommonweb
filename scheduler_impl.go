package gocommonweb

import (
	"context"
	"encoding/base64"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
	gredis "github.com/go-redsync/redsync/v4/redis"
	"github.com/go-redsync/redsync/v4/redis/goredis/v8"
	"github.com/robfig/cron"
	"github.com/sirupsen/logrus"
)

type byShortestNextExecution []*scheduleEntry

func (s byShortestNextExecution) Len() int      { return len(s) }
func (s byShortestNextExecution) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s byShortestNextExecution) Less(i, j int) bool {
	return s[i].nextExecution.Before(s[j].nextExecution)
}

type scheduleEntry struct {
	jobName       string
	cronSpec      string
	nextExecution time.Time
	cronSchedule  cron.Schedule
}

type ScheduleSafeImpl struct {
	running         bool
	handlers        map[string]ScheduleHandler
	scheduleEntries []*scheduleEntry
	stopChan        chan bool
	rSync           *redsync.Redsync
	redisClient     *redis.Client
}

// NewScheduler create scheduler instance
// if you provide more than one redis client then the first
// client in a list will be used to store schedule session
// and the rest as pools for redis mutex
//
// note: if you run many instances of the scheduler it may occur
// thundering problem, but for general and most cases of scheduling
// work it should work just fine
func NewScheduler(redisClient ...*redis.Client) Scheduler {
	if len(redisClient) <= 0 {
		panic(fmt.Errorf("please provide at least one redis client instance"))
	}

	// Provide more redis client for different redis instance
	// to gives reliability in locking process as discussed in
	// https://redis.io/topics/distlock
	var pools []gredis.Pool
	if len(redisClient) == 1 {
		pools = append(pools, goredis.NewPool(redisClient[0]))
	} else {
		for _, client := range redisClient[1:] {
			pools = append(pools, goredis.NewPool(client))
		}
	}
	redSync := redsync.New(pools...)

	return &ScheduleSafeImpl{
		running:     false,
		handlers:    make(map[string]ScheduleHandler),
		stopChan:    make(chan bool),
		rSync:       redSync,
		redisClient: redisClient[0],
	}
}

func (s *ScheduleSafeImpl) ScheduleJobToQueue(jobName string, cronSpec string, queue Queue) error {
	return s.ScheduleJob(jobName, cronSpec, func(execTime time.Time, jobName string, cronSpec string) {
		_ = queue.AddJob(jobName, "")
	})
}

func (s *ScheduleSafeImpl) ScheduleJob(jobName string, cronSpec string, handler ScheduleHandler) error {
	if s.running {
		return fmt.Errorf("cannot add new job while scheduler already running")
	}

	schedule, err := cron.ParseStandard(cronSpec)
	if err != nil {
		return err
	}

	entry := scheduleEntry{
		jobName:       jobName,
		cronSpec:      cronSpec,
		nextExecution: schedule.Next(now()),
		cronSchedule:  schedule,
	}

	s.scheduleEntries = append(s.scheduleEntries, &entry)
	s.handlers[jobName] = handler
	if nextExec, err := s.retrieveNextExecutionTime(jobName, cronSpec); err == nil {
		entry.nextExecution = time.Unix(nextExec, 0)
	} else {
		return s.updateNextExecutionTime(&entry)
	}
	return nil
}

func (s *ScheduleSafeImpl) Stop() {
	if s.running {
		s.running = false
		s.stopChan <- true
	}
}

func (s *ScheduleSafeImpl) Start() {
	if s.running {
		return
	}
	s.running = true
	go s.run()
}

func (s *ScheduleSafeImpl) run() {
	for {
		if len(s.scheduleEntries) <= 0 {
			s.running = false
			return
		}

		s.refreshScheduleEntries()
		sort.Sort(byShortestNextExecution(s.scheduleEntries))
		duration := s.scheduleEntries[0].nextExecution.Sub(now())
		timer := time.NewTimer(duration)

		select {
		case <-timer.C:
			now := now()
			for _, entry := range s.scheduleEntries {
				if entry.nextExecution.Unix() <= now.Unix() {
					if handler, ok := s.handlers[entry.jobName]; ok {
						if err := s.lock(entry); err == nil {
							go handler(now, entry.jobName, entry.cronSpec)
							entry.nextExecution = entry.cronSchedule.Next(now)
							_ = s.updateNextExecutionTime(entry)
						}
					}
				} else {
					break
				}
			}
		case <-s.stopChan:
			logrus.Debug("scheduler stopped intentionally")
			return
		}
	}
}

func (s *ScheduleSafeImpl) retrieveNextExecutionTime(jobName string, cronSpec string) (int64, error) {
	key := getSchedulerKey(jobName, cronSpec)
	if res, err := s.redisClient.Get(context.Background(), key).Result(); err == nil {
		t := stringToInt64(res)
		if t == 0 {
			return 0, fmt.Errorf("next duration time is zero")
		}
		return t, nil
	} else {
		return 0, err
	}
}

func (s *ScheduleSafeImpl) refreshScheduleEntries() {
	for _, entry := range s.scheduleEntries {
		key := getSchedulerKey(entry.jobName, entry.cronSpec)
		// TODO optimize using redis hash map
		if res, err := s.redisClient.Get(context.Background(), key).Result(); err == nil {
			entry.nextExecution = time.Unix(stringToInt64(res), 0)
		}
	}
}

func (s *ScheduleSafeImpl) updateNextExecutionTime(entry *scheduleEntry) error {
	key := getSchedulerKey(entry.jobName, entry.cronSpec)
	duration := entry.nextExecution.Sub(now())
	return s.redisClient.Set(context.Background(), key, entry.nextExecution.Unix(), duration).Err()
}

func (s *ScheduleSafeImpl) lock(entry *scheduleEntry) error {
	now := time.Unix(time.Now().Unix(), 0)
	duration := entry.cronSchedule.Next(now).Sub(now)
	duration = duration - time.Duration(float64(duration)*lockDurationOffsetFactor)

	key := getMutexKey(entry.jobName, entry.cronSpec)
	m := s.rSync.NewMutex(key, redsync.WithExpiry(duration), redsync.WithTries(2))
	return m.Lock()
}

func getSchedulerKey(jobName string, cronSpec string) string {
	cronSpecEnc := base64.URLEncoding.EncodeToString([]byte(cronSpec))
	return fmt.Sprintf("scheduler:%s-%s", jobName, cronSpecEnc)
}

func getMutexKey(jobName string, cronSpec string) string {
	cronSpecEnc := base64.URLEncoding.EncodeToString([]byte(cronSpec))
	return fmt.Sprintf("handlerMutex:%s-%s", jobName, cronSpecEnc)
}

func now() time.Time {
	return time.Unix(time.Now().Unix(), 0)
}

const lockDurationOffsetFactor = 0.1

func stringToInt64(str string) int64 {
	if parsed, err := strconv.ParseInt(str, 10, 64); err == nil {
		return parsed
	}
	return 0
}
