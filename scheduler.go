package gocommonweb

import "time"

type ScheduleHandler func(execTime time.Time, jobName string, cronSpec string)

type Scheduler interface {
	ScheduleJob(jobName string, cronSpec string, handler ScheduleHandler) error
	ScheduleJobToQueue(jobName string, cronSpec string, queue Queue) error
	Start()
	Stop()
}
