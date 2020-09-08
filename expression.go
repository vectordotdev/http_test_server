package main

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/Knetic/govaluate"
)

var expressionFunctions = map[string]govaluate.ExpressionFunction{
	"rand": func(_ ...interface{}) (interface{}, error) {
		return rand.Float64(), nil
	},
	"sin": func(args ...interface{}) (interface{}, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("sin() expects one argument")
		}
		switch v := args[0].(type) {
		case float64:
			return math.Sin(v), nil
		default:
			return nil, fmt.Errorf("sin() expects a numeric argument")
		}
	},
}

type expressionParameters struct {
	t              time.Duration
	activeRequests uint32
}

func (p *expressionParameters) Get(name string) (interface{}, error) {
	switch name {
	case "active_requests":
		return p.activeRequests, nil
	case "pi":
		return math.Pi, nil
	case "t":
		return int64(p.t / time.Second), nil
	default:
		return nil, fmt.Errorf("unknown variable name: %s", name)
	}
}
