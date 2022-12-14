// Package web provides a decorating reverse proxy server.
package web

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/codegangsta/negroni"
)

// An Env contains data for a template.
type Env map[string]interface{}

// A Handler responds to requests, optionally with params.
type Handler interface {
	ServeHTTP(rw http.ResponseWriter, r *http.Request, p Params)
}

// The HandlerFunc type lets functions be Handlers.
type HandlerFunc func(rw http.ResponseWriter, r *http.Request, p Params)

// Middleware are bidirectonal wrappers around requests.
type Middleware interface {
	ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc)
}

// The MiddlewareFunc lets functions be Middleware.
type MiddlewareFunc func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc)

// A Web server is a stack of middleware and a router.
type Web struct {
	config *Config
	router *router
	engine *engine
	assets *assets
	before []Middleware
	after  []Middleware
}

// New creates a server instance.
func New(c *Config) *Web {
	if c == nil {
		c = &Config{}
	}

	w := &Web{config: c}

	w.router = w.newRouter()
	w.engine = w.newEngine()
	w.assets = w.newAssets()
	w.before = w.newBefore()
	w.after = w.newAfter()

	return w
}

// Serve starts the server at the frontend URL.
func (w *Web) Serve() error {
	port, err := w.config.port()

	if err != nil {
		return err
	}

	return http.ListenAndServe(port, w.newStack())
}

// ServeHTTP handles a given req/res (used for testing).
func (w *Web) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	w.newStack().ServeHTTP(rw, r)
}

// Handler adds a handler object for the given method and path.
func (w *Web) Handler(method string, path string, handler Handler) {
	w.router.handler(method, path, handler)
}

// HandlerFunc adds a handler func for the given method and path.
func (w *Web) HandlerFunc(method string, path string, handler HandlerFunc) {
	w.router.handlerFunc(method, path, handler)
}

// Middleware adds a ware object to the stack, pre router.
func (w *Web) Middleware(mw ...Middleware) {
	w.before = append(w.before, mw...)
}

// MiddlewareFunc adds a ware func to the stack, pre router.
func (w *Web) MiddlewareFunc(mw ...MiddlewareFunc) {
	for _, m := range mw {
		w.before = append(w.before, negroni.HandlerFunc(m))
	}
}

// Execute renders a template file with data.
func (w *Web) Execute(file string, input Env) (*bytes.Buffer, error) {
	return w.engine.execute(file, input)
}

// Respond renders a template file with data as a response.
func (w *Web) Respond(rw http.ResponseWriter, r *http.Request, status int, file string, input Env) {
	w.engine.respond(rw, r, status, file, input)
}

// Redirect adds a GET route that redirects to another route.
func (w *Web) Redirect(from, to string, code int) {
	w.HandlerFunc("get", from, func(rw http.ResponseWriter, r *http.Request, _ Params) {
		http.Redirect(rw, r, to, code)
	})
}

// NotFound adds a fallback handler for unmatched requests.
func (w *Web) NotFound(handler http.HandlerFunc) {
	w.after = w.after[:len(w.after)-1]
	w.after = append(w.after, newMiddleware(handler))
}

// FuncMap adds to the map of template functions.
func (w *Web) FuncMap(funcs template.FuncMap) {
	w.engine.funcMap(funcs)
}

func (w *Web) newBefore() []Middleware {
	return []Middleware{
		w.newRecover(),
		w.newReverse(),
		w.newPrefix(),
		w.newSecure(),
		w.newIgnore(),
		w.newAuth(),
	}
}

func (w *Web) newAfter() []Middleware {
	if w.config.prod() {
		return []Middleware{
			w.engine,
			w.newStatic(),
			w.newNocache(),
			w.newProxy(),
			w.newNotfound(),
		}
	}

	return []Middleware{
		w.newNocache(),
		w.engine,
		w.newStatic(),
		w.newProxy(),
		w.newNotfound(),
	}
}

func (w *Web) newStack() http.Handler {
	stack := negroni.New()

	w.append(stack, w.before...)
	w.append(stack, w.router)
	w.append(stack, w.after...)

	return stack
}

func (w *Web) append(stack *negroni.Negroni, mw ...Middleware) {
	for _, handler := range mw {
		if handler != nil {
			stack.Use(handler)
		}
	}
}
