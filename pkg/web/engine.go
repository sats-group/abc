package web

import (
	"bytes"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/sats-group/abc/internal/files"
	"github.com/sats-group/abc/internal/tmpl"
)

const (
	contentTypeKey = "Content-Type"
	contentTypeVal = "text/html; charset=UTF-8"
)

type engine struct {
	config    *Config
	funcs     template.FuncMap
	templates *template.Template
}

func (w *Web) newEngine() *engine {
	return &engine{
		config: w.config,
		funcs:  tmpl.Funcs,
	}
}

func (e *engine) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	path := e.expandPath(r.URL.Path)

	if r.Method != http.MethodGet || e.skipPath(path) {
		next(rw, r)
		return
	}

	e.respond(rw, r, http.StatusOK, path, nil)
}

func (e *engine) respond(rw http.ResponseWriter, r *http.Request, status int, file string, data Env) {
	env := e.createEnv(rw, r, data)
	out, err := e.execute(file, env)

	if err != nil {
		log.Println(file, err)
		http404(rw, r)
		return
	}

	rw.Header().Set(contentTypeKey, contentTypeVal)
	rw.WriteHeader(http.StatusOK)

	if _, err = out.WriteTo(rw); err != nil {
		log.Println(file, err)
		http500(rw, r)
	}
}

func (e *engine) execute(file string, env interface{}) (*bytes.Buffer, error) {
	if e.templates == nil || !e.config.prod() {
		if err := e.compileTemplates(); err != nil {
			return nil, err
		}
	}

	if e.config.layout() != "" {
		e.funcMapLayout(e.templateName(file), env)
		file = e.config.layout()
	}

	return e.template(file, env)
}

func (e *engine) template(file string, env interface{}) (*bytes.Buffer, error) {
	tpl := e.templateName(file)
	buf := new(bytes.Buffer)

	return buf, e.templates.ExecuteTemplate(buf, tpl, env)
}

func (e *engine) funcMap(funcs template.FuncMap) {
	for k, v := range funcs {
		e.funcs[k] = v
	}

	e.templates = nil
}

func (e *engine) createEnv(rw http.ResponseWriter, r *http.Request, data Env) Env {
	env := Env{
		"prod":   e.config.prod(),
		"config": e.config.json(),
	}

	for key, val := range data {
		env[key] = val
	}

	return env
}

func (e *engine) templateName(path string) string {
	return strings.TrimPrefix(
		strings.TrimSuffix(strings.Replace(path, "\\", "/", -1), filepath.Ext(path)),
		"/",
	)
}

func (e *engine) expandPath(path string) string {
	if strings.HasSuffix(path, "/") {
		return path + "index.html"
	}

	return path
}

func (e *engine) skipPath(file string) bool {
	std := e.config.frontendExt()
	ext := filepath.Ext(file)

	if len(ext) > 0 && ext != std {
		return true
	}

	if len(ext) > 0 {
		return e.skipFile(file)
	}

	return e.skipFile(file + std)
}

func (e *engine) skipFile(file string) bool {
	f, err := http.Dir(e.config.dir()).Open(file)

	if err != nil {
		return true
	}

	if _, err = f.Stat(); err != nil {
		return true
	}

	if err := f.Close(); err != nil {
		return true
	}

	return false
}

func (e *engine) compileTemplates() error {
	e.templates = template.New(e.config.dir())

	return files.Walk(e.config.dir(), func(rel string) error {
		ext := filepath.Ext(rel)

		if ext != e.config.frontendExt() && ext != e.config.backendExt() {
			return nil
		}

		return e.compileTemplate(rel)
	})
}

func (e *engine) compileTemplate(rel string) error {
	name := e.templateName(rel)
	data := files.Read(filepath.Join(e.config.dir(), rel))

	tmpl := e.templates.Funcs(e.funcs).New(name)
	_, err := tmpl.Parse(string(data[:]))

	return err
}

func (e *engine) funcMapLayout(name string, env interface{}) {
	if tmpl := e.templates.Lookup(name); tmpl != nil {
		tmpl.Funcs(template.FuncMap{
			"yield": func() template.HTML {
				buf, err := e.template(name, env)

				if err != nil {
					return template.HTML("")
				}

				return template.HTML(buf.String())
			},
		})
	}
}
