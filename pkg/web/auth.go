package web

import (
	"log"
	"net/http"
	"strings"

	"github.com/goji/httpauth"
)

type auth struct {
	patterns []string
	matchers []*matcher
}

type matcher struct {
	handler func(http.Handler) http.Handler
	pattern string
}

func (w *Web) newAuth() Middleware {
	if len(w.config.Auth) == 0 {
		return nil
	}

	a := &auth{patterns: w.config.Auth}
	a.matchers = a.parsePatterns(a.patterns)

	return a
}

func (a *auth) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	if len(a.matchers) == 0 {
		next(rw, r)
		return
	}

	if handler := a.matchingHandler(r.URL.RequestURI()); handler != nil {
		handler(next).ServeHTTP(rw, r)
		return
	}

	next(rw, r)
}

func (a *auth) matchingHandler(uri string) func(http.Handler) http.Handler {
	source := strings.TrimPrefix(uri, "/")

	for _, m := range a.matchers {
		if strings.HasPrefix(source, m.pattern) {
			return m.handler
		}
	}

	return nil
}

func (a *auth) parsePatterns(patterns []string) []*matcher {
	matchers := []*matcher{}

	for _, pattern := range a.patterns {
		if pattern != "" {
			if matcher := a.parsePattern(pattern); matcher != nil {
				matchers = append(matchers, matcher)
			}
		}
	}

	return matchers
}

func (a *auth) parsePattern(pattern string) *matcher {
	colon := strings.LastIndex(pattern, ":")
	alpha := strings.LastIndex(pattern, "@")

	if alpha == -1 {
		alpha = len(pattern) - 1
	}

	if colon == -1 || len(pattern) < 3 {
		log.Fatalf("invalid auth pattern: %s\n", pattern)
	}

	user := pattern[:colon]
	pass := pattern[colon+1 : alpha]
	path := pattern[alpha+1:]

	return &matcher{
		handler: httpauth.SimpleBasicAuth(user, pass),
		pattern: strings.TrimPrefix(path, "/"),
	}
}
