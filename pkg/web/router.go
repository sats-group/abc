package web

import (
	"net/http"
	"strings"

	"github.com/julienschmidt/httprouter"
)

type router struct {
	*httprouter.Router
}

func (w *Web) newRouter() *router {
	return &router{
		httprouter.New(),
	}
}

func (rt *router) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	rt.Router.HandleMethodNotAllowed = false
	rt.Router.NotFound = next
	rt.Router.ServeHTTP(rw, r)
}

func (rt *router) handler(method string, path string, handler Handler) {
	rt.handle(path, method, func(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
		handler.ServeHTTP(rw, r, Params{p})
	})
}

func (rt *router) handlerFunc(method string, path string, handler HandlerFunc) {
	rt.handle(path, method, func(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
		handler(rw, r, Params{p})
	})
}

func (rt *router) handle(path string, method string, fn httprouter.Handle) {
	path = strings.ToLower(path)
	method = strings.ToUpper(method)
	rt.Router.Handle(method, path, fn)
}
