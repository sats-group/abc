package web

import (
	"log"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/codegangsta/negroni"
	"github.com/olav/abc/internal/files"
	"github.com/unrolled/secure"
	"github.com/zenazn/goji/web/middleware"
)

func (w *Web) newRecover() Middleware {
	rec := negroni.NewRecovery()
	rec.PrintStack = !w.config.Prod
	rec.Logger = log.New(os.Stderr, "", log.LstdFlags)
	return rec
}

func (w *Web) newReverse() Middleware {
	if !w.config.Proxy {
		return nil
	}

	fn := func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		middleware.RealIP(next).ServeHTTP(rw, r)
	}

	return negroni.HandlerFunc(fn)
}

func (w *Web) newPrefix() Middleware {
	pre := w.config.frontendPath()

	if pre == "" {
		return nil
	}

	fn := func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		p := strings.TrimPrefix(r.URL.Path, pre)

		if len(p) == len(r.URL.Path) {
			http404(rw, r)
			return
		}

		if path.Ext(p) == "" && !strings.HasSuffix(p, "/") {
			p = p + "/"
		}

		r.URL.Path = p
		next(rw, r)
	}

	return negroni.HandlerFunc(fn)
}

func (w *Web) newSecure() Middleware {
	sec := secure.New(secure.Options{
		IsDevelopment:      !w.config.Prod,
		ContentTypeNosniff: true,
		BrowserXssFilter:   true,
		FrameDeny:          false,
	})

	fn := sec.HandlerFuncWithNext
	return negroni.HandlerFunc(fn)
}

func (w *Web) newIgnore() Middleware {
	end := w.config.backend()
	ext := w.config.backendExt()

	fn := func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		if end != "" && strings.HasSuffix(r.URL.Path, ext) {
			http404(rw, r)
		} else if files.Ignore(r.URL.Path) {
			http404(rw, r)
		} else {
			next(rw, r)
		}
	}

	return negroni.HandlerFunc(fn)
}

func (w *Web) newStatic() Middleware {
	return negroni.NewStatic(http.Dir(w.config.dir()))
}

func (w *Web) newNocache() Middleware {
	fn := func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		middleware.NoCache(next).ServeHTTP(rw, r)
	}

	return negroni.HandlerFunc(fn)
}

func (w *Web) newNotfound() Middleware {
	fn := func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		http404(rw, r)
	}

	return negroni.HandlerFunc(fn)
}

func newMiddleware(handler http.HandlerFunc) Middleware {
	fn := func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		handler(rw, r)
		next(rw, r)
	}

	return negroni.HandlerFunc(fn)
}
