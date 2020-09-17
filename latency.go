package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/Knetic/govaluate"
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

type LatencyMiddlewareExpression struct {
	mean   *govaluate.EvaluableExpression
	stddev *govaluate.EvaluableExpression

	activeRequests  uint32
	serverStartTime time.Time
}

func NewLatencyMiddlewareExpression(mean string, stddev string) (*LatencyMiddlewareExpression, error) {
	meanExpression, err := govaluate.NewEvaluableExpressionWithFunctions(mean, expressionFunctions)
	if err != nil {
		return nil, fmt.Errorf("could not use mean expression: %s", err)
	}
	stddevExpression, err := govaluate.NewEvaluableExpressionWithFunctions(stddev, expressionFunctions)
	if err != nil {
		return nil, fmt.Errorf("could not use stddev expression: %s", err)
	}
	return &LatencyMiddlewareExpression{
		mean:            meanExpression,
		stddev:          stddevExpression,
		serverStartTime: time.Now(),
	}, nil
}

func (lm *LatencyMiddlewareExpression) WrapHTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		atomic.AddUint32(&lm.activeRequests, 1)
		defer atomic.AddUint32(&lm.activeRequests, ^uint32(0)) // decrement

		errFn := func(err error) {
			log.Println(err)
			http.Error(rw, err.Error(), http.StatusInternalServerError)
		}

		parameters := &expressionParameters{
			activeRequests: atomic.LoadUint32(&lm.activeRequests),
			t:              time.Now().Sub(lm.serverStartTime),
		}

		v, err := lm.mean.Eval(parameters)
		if err != nil {
			errFn(fmt.Errorf("cannot evaluate mean expression: %s", err))
			return
		}

		var mean float64
		switch v := v.(type) {
		case float64:
			mean = v
		default:
			errFn(fmt.Errorf("mean expression did not return a float64, returned: %T", v))
			return
		}

		v, err = lm.stddev.Eval(parameters)
		if err != nil {
			errFn(fmt.Errorf("cannot evaluate stddev expression: %s", err))
			return
		}

		var stddev float64
		switch v := v.(type) {
		case float64:
			stddev = v
		default:
			errFn(fmt.Errorf("mean expression did not return a float64, returned: %T", v))
			return
		}

		d := time.Duration(rand.NormFloat64()*stddev+mean) * time.Millisecond
		time.Sleep(d)
		next.ServeHTTP(rw, r)
	})
}

type LatencyDistribution string

const (
	LatencyDistributionNormal   = LatencyDistribution("NORMAL")
	LatencyDistributionFunction = LatencyDistribution("EXPRESSION")
)
