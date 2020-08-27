package main

import (
	"fmt"
	"math/rand"

	"github.com/Knetic/govaluate"
)

var expressionFunctions = map[string]govaluate.ExpressionFunction{
	"rand": func(_ ...interface{}) (interface{}, error) {
		return rand.Float64(), nil
	},
}

type expressionParameters struct {
	activeRequests uint32
}

func (p *expressionParameters) Get(name string) (interface{}, error) {
	switch name {
	case "active_requests":
		return p.activeRequests, nil
	default:
		return nil, fmt.Errorf("unknown variable name: %s", name)
	}
}
