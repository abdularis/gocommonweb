## Go Common Web 📦🌏

Contains utilities and tools commonly used in web application. You can reuse this package easily for different projects since it's an independent package.

## Components
- [Cache](#cache)
- [Event](#event)
- [JWT](#jwt)
- [Queue](#queue)
- [Scheduler](#scheduler)
- [Storage](#storage)
- [WebSocket](#websocket)
- [Other Utils](#other-utilities)

### Cache

Provides general purpose simple key value pair caching mechanism implemented using redis.

Usage:
```go
// new cache implemented using redis
cache := framework.NewCacheRedis(redisClient, "starter-app")

// put in cache forever
cache.Put("key0", "you data payload here")

// this will disappear in 1 minute
cache.PutWithTTL("key1", "your data payload here", time.Minute)

// 
value, err := cache.Get("key1")
```

### Event

Event is a pub/sub mechanism, provided implementation using redis pub/sub.
> note that redis pub/sub is broadcast

Usage:
```go
type mySubscription struct {}

func (s mySubscription) Handle(eventName string, payload string) {
	// will print 'event occurred su_server_0
	fmt.Printf("event occurred %s\n", eventName)
}

func main() {
    event := framework.NewEventRedis(redisCli)
    
    // subscribe to an event called 'sub_server_0' with mySubscription as handler that
    // will be called when event published by other
    event.Subscribe("sub_server_0", &mySubscription{})
    
    go func() {
    	// publish event that will be handle by subscriber
    	event.Publish("sub_server_0", "")
    }()
    
    time.Sleep(time.Minute)
    
    event.Unsubscribe("sub_server_0")
}
```

### JWT

Provide functionality to generate JWS/JWE token and verify them with given private key

Usage:
```go
config := framework.JWTConfig{
	SignatureAlgo:      jwa.RS256,
	EnableJWE:          true,
	KeyEncryptAlgo:     jwa.RSA1_5,
	ContentEncryptAlgo: jwa.A128CBC_HS256,
	JWECompressAlgo:    jwa.NoCompress,
}

pemKey := framework.GenerateRSAPrivateKey(2048)
jwtUtil := framework.NewJWTUtilWithPEM(string(pemKey), config)


var claims map[string]interface{}
claims["email"] = "aris@gmail.com"

token, err := jwtUtil.GenerateJWT(claims)
if err != nil {
    panic(err)
}

jwtTkn, err := jwtUtil.VerifyJWT(token)
if err != nil {
    panic(err)
}
```

### Queue

Queue provide common job queuing functionality for asynchronous execution.
Implemented using database `queue_db.go` or using redis `queue_redis.go`

Usage:
```go
type sendEmailHandler struct {}

func (s sendEmailHandler) Handle(jobName string, payload string) error {
    // Send email logic here
}

func main() {
    queue := framework.NewQueueDB(gormDB, 5)

    // first add job handler that will be called if there is a job needs to be run
    queue.AddJobHandler("send_email", &sendEmailHandler{})

    // queue the job
    queue.AddJob("send_email", "aris@gmail.com")

    // execute job after 30 secs
    queue.AddDelayedJob("send_email", "aris@gmail.com", 30)

    // stop queue
    queue.Stop()
}
```

### Scheduler

Implement periodic job scheduler, you provide cron spec as it's scheduling pattern. this implementation is safe to run on multiple instances, but at the same time only one job for a particular schedule will be run.

> note that this is implemented using redis, all scheduler instance will schedule and when there is an overdue job it will acquire a lock for that job and only one instance would win

Usage:
```go
// create new scheduler, provide multiple redisClient master instance for global lock safety see NewScheduler() docs
s := framework.NewScheduler(redisClient)

// provide standard cron spec for schedule specification
s.ScheduleJob("send_email_promo", "@every 5h", func(execTime time.Time, jobName string, cronSpec string) {
	// this will be called every 5 hour
	fmt.Println("send email promotion logic here")
})

// start scheduling worker
s.Start()
defer s.Stop()

time.Sleep(time.Hour)
```

### Storage

Provide storage abstraction for working with files/persistence object.

Storage implementation:
- `AWS S3`
- `Alibaba OSS`
- `Local File Storage`

Usage:

```go
// first param is where the private file will be stored
// second param is where the public hosted file will be stored (note that this directory needs to be served by your web server to give access)
// third param is the base url for this hosted file, it is use to generate URL from object key (path) and append this base url to build full url
storage := framework.NewStorageLocalFile("./storage/private", "./storage/public", "http://localhost/files")

stream, err := storage.Read("users/images/profile.jpg")
```

### WebSocket

WebSocket provide full-duplex communication between user and server. You can run multiple websocket server, if you send
a message to a particular user connection that is not connected in the same server it will find the server where the
client is connected to and push via that server. This mechanism handled by redis to store all connection sessions
and [Event](#event) for server-to-server communication.


### Other Utilities

**Password Hash**

bcrypt password hash function

Usage:
```go
hashed, err := framework.HashPassword("your_password")
if err != nil {
    panic(err)
}

valid := framework.CheckPasswordHash("your_password", hashed)
if valid {
    fmt.Println("your password correct")
}
```
