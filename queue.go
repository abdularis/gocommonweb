package gocommonweb

// JobHandler callback for handling actual job
type JobHandler interface {
	Handle(jobName string, payload string) error
}

// Queue async job execution
type Queue interface {
	AddJob(jobName string, payload string) error
	AddDelayedJob(jobName string, payload string, delaySecs uint) error
	AddJobHandler(jobName string, handler JobHandler)
	Start()
	Close()
}
