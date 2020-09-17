package main

import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/Knetic/govaluate"
)

type ErrorExpressionMiddleware struct {
	expr *govaluate.EvaluableExpression

	activeRequests  uint32
	serverStartTime time.Time
}

func NewErrorExpressionMiddleware(expression string) (*ErrorExpressionMiddleware, error) {
	expr, err := govaluate.NewEvaluableExpressionWithFunctions(expression, expressionFunctions)
	if err != nil {
		return nil, fmt.Errorf("could not use expression: %s", err)
	}

	return &ErrorExpressionMiddleware{
		expr:            expr,
		serverStartTime: time.Now(),
	}, nil
}

func (em *ErrorExpressionMiddleware) WrapHTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		atomic.AddUint32(&em.activeRequests, 1)
		defer atomic.AddUint32(&em.activeRequests, ^uint32(0)) // decrement

		errFn := func(err error) {
			log.Println(err)
			http.Error(rw, err.Error(), http.StatusInternalServerError)
		}

		parameters := &expressionParameters{
			activeRequests: atomic.LoadUint32(&em.activeRequests),
			t:              time.Now().Sub(em.serverStartTime),
		}

		v, err := em.expr.Eval(parameters)
		if err != nil {
			errFn(fmt.Errorf("cannot evaluate mean expression: %s", err))
			return
		}

		switch v := v.(type) {
		case float64:
			code := int(v)
			rw.WriteHeader(code)
			fmt.Fprintln(rw, http.StatusText(code))
		case string:
			switch v {
			case "CLOSE":
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
			default:
				errFn(fmt.Errorf("expression returned a string, '%s', but it was not recognized", v))
			}
		case bool:
			if v {
				http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			} else {
				next.ServeHTTP(rw, r)
			}
		default:
			errFn(fmt.Errorf("expression did not return an expected type, returned: %T", v))
		}
	})
}
