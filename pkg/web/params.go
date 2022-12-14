package web

import (
	"strings"

	"github.com/julienschmidt/httprouter"
)

// Params holds named or wildcard URL parameters.
type Params struct {
	httprouter.Params
}

// Get returns the value of a named parameter.
func (p Params) Get(name string) string {
	return p.Params.ByName(name)
}

// Wildcard returns the value of a wildcard parameter.
func (p Params) Wildcard(name string) string {
	param := p.Get(name)

	if strings.HasPrefix(param, "/") {
		return param[1:]
	}

	return param
}
