package web

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/url"
	"path"
	"strings"

	"github.com/sats-group/abc/internal/files"
)

// Config configures a server instance.
type Config struct {
	Frontend string
	Backend  string

	Dir    string
	JSON   []string
	Layout string
	Auth   []string
	Proxy  bool
	Prod   bool
	Debug  bool

	cache map[string]interface{}
}

func (c *Config) frontend() string {
	return c.addr(c.Frontend, "http://localhost:8000/")
}

func (c *Config) backend() string {
	return c.addr(c.Backend, "")
}

func (c *Config) frontendExt() string {
	return ".html"
}

func (c *Config) backendExt() string {
	return ".tmpl"
}

func (c *Config) frontendPath() string {
	return c.path(c.frontend())
}

func (c *Config) backendPath() string {
	return c.path(c.backend())
}

func (c *Config) layout() string {
	if c.Layout == "" {
		return ""
	}

	d := strings.TrimPrefix(path.Dir(c.Layout), "/")
	n := files.Name(c.Layout)

	return path.Join(d, n)
}

func (c *Config) path(raw string) string {
	u, err := url.Parse(raw)

	if err != nil || u.Path == "/" {
		return ""
	}

	return strings.TrimSuffix(u.Path, "/")
}

func (c *Config) port() (string, error) {
	port, err := c.addrPort(c.frontend())

	if err != nil {
		return "", err
	}

	return ":" + port, nil
}

func (c *Config) prod() bool {
	return c.Prod
}

func (c *Config) proxy() bool {
	return c.Proxy
}

func (c *Config) debug() bool {
	return c.Debug
}

func (c *Config) dir() string {
	if c.Dir == "" {
		return "."
	}

	return c.Dir
}

func (c *Config) addr(addr string, fallback string) string {
	if addr == "" {
		return fallback
	}

	if strings.HasPrefix(addr, ":") {
		addr = "http://localhost" + addr
	}

	if err := c.addrValidate(addr); err != nil {
		return fallback
	}

	return strings.TrimSuffix(addr, "/") + "/"
}

func (c *Config) addrValidate(addr string) error {
	if !strings.Contains(addr, "://") {
		return fmt.Errorf("address must include protocol: %s\n", addr)
	}

	if _, err := url.Parse(addr); err != nil {
		return fmt.Errorf("address is invalid: %s (%s)", addr, err)
	}

	return nil
}

func (c *Config) addrPort(addr string) (string, error) {
	u, err := url.Parse(addr)

	if err != nil {
		return "", err
	}

	_, port, err := net.SplitHostPort(u.Host)

	if err != nil {
		return "", err
	}

	return port, nil
}

func (c *Config) json() map[string]interface{} {
	if len(c.JSON) == 0 {
		return nil
	}

	if c.prod() && c.cache != nil {
		return c.cache
	}

	c.cache = Env{}

	for _, rel := range c.JSON {
		c.cache[files.Name(rel)] = c.load(rel)
	}

	return c.cache
}

func (c *Config) load(rel string) map[string]interface{} {
	cfg := Env{}

	if !files.HasFile(rel) {
		log.Fatalf("unknown file: %s\n", rel)
		return nil
	}

	if err := json.Unmarshal(files.Read(rel), &cfg); err != nil {
		log.Fatalf("parse error: %s\n", err)
		return nil
	}

	return cfg
}
