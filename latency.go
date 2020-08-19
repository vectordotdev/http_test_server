package main

import (
	"net/http"
	"time"
)

type LatencyMiddlewareStatic struct {
	latency time.Duration
}

func NewLatencyMiddlewareStatic(latency time.Duration) *LatencyMiddlewareStatic {
	return &LatencyMiddlewareStatic{
		latency: latency,
	}
}

func (lm *LatencyMiddlewareStatic) WrapHTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		time.Sleep(lm.latency)
		next.ServeHTTP(rw, r)
	})
}
