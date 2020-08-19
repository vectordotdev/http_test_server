package main

import (
	"math/rand"
	"net/http"
	"time"
)

type LatencyMiddlewareNormal struct {
	mean   time.Duration
	stddev time.Duration
}

func NewLatencyMiddlewareNormal(mean time.Duration, stddev time.Duration) *LatencyMiddlewareNormal {
	return &LatencyMiddlewareNormal{
		mean:   mean,
		stddev: stddev,
	}
}

func (lm *LatencyMiddlewareNormal) WrapHTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		d := time.Duration(rand.NormFloat64())*lm.stddev + lm.mean
		time.Sleep(d)
		next.ServeHTTP(rw, r)
	})
}

type LatencyDistribution string

const (
	LatencyDistributionNormal = LatencyDistribution("NORMAL")
)
