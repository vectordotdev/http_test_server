package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/juju/ratelimit"
)

type RateLimiter interface {
	Middleware
}

type RateLimiterNone struct{}

func (rl *RateLimiterNone) WrapHTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(rw, r)
	})
}

type RateLimiterHard struct {
	bucket     *ratelimit.Bucket
	statusCode int
}

func NewRateLimiterHard(fillInterval time.Duration, capacity, quantum int64, statusCode int) *RateLimiterHard {
	return &RateLimiterHard{
		bucket:     ratelimit.NewBucketWithQuantum(fillInterval, capacity, quantum),
		statusCode: statusCode,
	}
}

func (rl *RateLimiterHard) WrapHTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if rl.bucket.TakeAvailable(1) == 0 {
			http.Error(rw, http.StatusText(rl.statusCode), rl.statusCode)
			return
		}
		next.ServeHTTP(rw, r)
	})
}

type RateLimiterQueue struct {
	bucket *ratelimit.Bucket
}

func NewRateLimiterQueue(fillInterval time.Duration, capacity, quantum int64) *RateLimiterQueue {
	return &RateLimiterQueue{
		bucket: ratelimit.NewBucketWithQuantum(fillInterval, capacity, quantum),
	}
}

func (rl *RateLimiterQueue) WrapHTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rl.bucket.Wait(1)
		next.ServeHTTP(rw, r)
	})
}

type RateLimiterClose struct {
	bucket *ratelimit.Bucket
}

func NewRateLimiterClose(fillInterval time.Duration, capacity, quantum int64) *RateLimiterClose {
	return &RateLimiterClose{
		bucket: ratelimit.NewBucketWithQuantum(fillInterval, capacity, quantum),
	}
}

func (rl *RateLimiterClose) WrapHTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if rl.bucket.TakeAvailable(1) == 0 {
			hj, ok := rw.(http.Hijacker)
			if !ok {
				panic("connection not hijackable") // should never happen
			}
			conn, _, err := hj.Hijack()
			if err != nil {
				http.Error(rw, fmt.Sprintf("could not hijack connection: %s", err.Error()), http.StatusInternalServerError)
				return
			}
			conn.Close() // drop connection
			return
		}

		next.ServeHTTP(rw, r)
	})
}

type RateLimitBehavior string

const (
	// no rate limit
	RateLimitBehaviorNone RateLimitBehavior = "NONE"

	// returns a 429 when rate is exceeded
	RateLimitBehaviorHard RateLimitBehavior = "HARD"

	// closes the connection if the rate is exceeded
	RateLimitBehaviorClose RateLimitBehavior = "CLOSE"

	// queues request until there is available capacity
	RateLimitBehaviorQueue RateLimitBehavior = "QUEUE"
)
