package web

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"strings"
)

type proxy struct {
	config *Config
	engine *engine
	client *http.Client
}

func (w *Web) newProxy() Middleware {
	if w.config.backend() == "" {
		return nil
	}

	return &proxy{
		config: w.config,
		engine: w.engine,
		client: &http.Client{},
	}
}

func (p *proxy) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	req, err := p.newRequest(rw, r)

	fail := func(err error) {
		log.Fatalln(err)
		next(rw, r)
	}

	if err != nil {
		fail(err)
		return
	}

	res, err := p.proxyPass(req)

	if err != nil {
		fail(err)
		return
	}

	if err := p.decorate(rw, r, res); err != nil {
		fail(err)
	}
}

func (p *proxy) newRequest(rw http.ResponseWriter, r *http.Request) (*http.Request, error) {
	url := p.config.backend() + strings.TrimPrefix(r.URL.RequestURI(), "/")
	req, err := http.NewRequest(r.Method, url, r.Body)

	if err != nil {
		return nil, err
	}

	return req, nil
}

func (p *proxy) proxyPass(r *http.Request) (*http.Response, error) {
	res, err := p.client.Do(r)

	if err != nil {
		return nil, err
	}

	return res, nil
}

func (p *proxy) decorate(rw http.ResponseWriter, r *http.Request, res *http.Response) (e error) {
	body, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return err
	}

	defer func() {
		if err := res.Body.Close(); err != nil {
			e = err
		}
	}()

	path := strings.TrimPrefix(r.URL.Path, "/")
	base := strings.TrimSuffix(path, filepath.Ext(path))
	tmpl := base + p.config.backendExt()
	data := Env{}

	if p.engine.skipFile(tmpl) {
		return p.decorateJSON(rw, res, body)
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return err
	}

	p.engine.respond(rw, r, res.StatusCode, tmpl, data)

	return nil
}

func (p *proxy) decorateJSON(rw http.ResponseWriter, res *http.Response, body []byte) error {
	nice := bytes.Buffer{}

	if err := json.Indent(&nice, body, "", "  "); err != nil {
		return err
	}

	rw.WriteHeader(res.StatusCode)
	_, err := rw.Write(nice.Bytes())

	return err
}
