package web

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"html/template"
	"io"
	"log"
	"mime"
	"net/http"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/microcosm-cc/bluemonday"
	"github.com/olav/abc/internal/files"
	"github.com/russross/blackfriday"
)

type assets struct {
	prod bool
	dir  string
	root string

	cache map[string]*assetCache
}

type assetType struct {
	name string
	ext  string
	html string
	proc func([]byte) []byte
}

type assetCache struct {
	name  string
	mime  string
	time  time.Time
	paths []string
	bytes []byte
}

type assetFunc func(sources ...interface{}) template.HTML

var (
	concatFile = "file"
	concatRoot = rooted("/assets")
	htmlPolicy = bluemonday.UGCPolicy()
)

var paste = &assetType{
	name: "paste",
	html: "<!-- %s -->\n%s",
}

var tpl = &assetType{
	name: "tpl",
	html: "<script type=\"text/template\" id=\"%s\">%s</script>",
}

var md = &assetType{
	name: "md",
	ext:  ".md",
	html: "<div class=\"md\" id=\"%s\">%s</div>",
	proc: markdown,
}

var css = &assetType{
	name: "css",
	ext:  ".css",
	html: "<link rel=\"stylesheet\" href=\"%s\">%s",
}

var js = &assetType{
	name: "js",
	ext:  ".js",
	html: "<script src=\"%s\">%s</script>",
}

func (w *Web) newAssets() *assets {
	a := &assets{
		prod:  w.config.prod(),
		dir:   w.config.dir(),
		root:  w.config.frontendPath(),
		cache: map[string]*assetCache{},
	}

	f := template.FuncMap{
		paste.name: a.inlined(paste),
		tpl.name:   a.inlined(tpl),
		md.name:    a.inlined(md),
		css.name:   a.file(css),
		js.name:    a.file(js),
	}

	if a.prod {
		f[css.name] = a.combined(css)
		f[js.name] = a.combined(js)
	}

	w.Handler("get", concatRoot+"*"+concatFile, a)
	w.FuncMap(f)

	return a
}

func (a *assets) ServeHTTP(rw http.ResponseWriter, r *http.Request, p Params) {
	filename := p.Wildcard(concatFile)
	file, ok := a.cache[filename]

	if !ok {
		http404(rw, r)
		return
	}

	reader := bytes.NewReader(file.bytes)
	rw.Header().Set("content-type", file.mime)
	http.ServeContent(rw, r, filename, file.time, reader)
}

func (a *assets) combined(t *assetType) assetFunc {
	return func(sources ...interface{}) template.HTML {
		pack := a.unpackPaths(sources)
		file := a.combosFromPaths(t, pack)
		href := concatRoot + file.name

		if strings.HasPrefix(pack[0], "/") {
			href = path.Join(a.root, href)
		} else {
			href = strings.TrimPrefix(href, "/")
		}

		return template.HTML(fmt.Sprintf(t.html, href, ""))
	}
}

func (a *assets) file(t *assetType) func(...interface{}) template.HTML {
	return func(sources ...interface{}) template.HTML {
		files := a.resolvePaths(a.unpackPaths(sources))

		for i, rel := range files {
			files[i] = fmt.Sprintf(t.html, path.Join(a.root, rel), "")
		}

		return template.HTML(strings.Join(files, "\n"))
	}
}

func (a *assets) inlined(t *assetType) assetFunc {
	return func(sources ...interface{}) template.HTML {
		tags := []string{}

		files := a.resolvePaths(a.unpackPaths(sources))
		items := a.inlinedFromPaths(t, files)

		for _, res := range items {
			src := path.Join(a.root, res.name)
			tags = append(tags, fmt.Sprintf(t.html, src, string(res.bytes[:])))
		}

		return template.HTML(strings.Join(tags, "\n"))
	}
}

func (a *assets) combosFromPaths(t *assetType, paths []string) *assetCache {
	name := hash(strings.Join(paths, "")) + t.ext

	if cached, ok := a.cache[name]; ok && a.prod {
		return cached
	}

	b := a.bytesFromPaths(paths)

	if t.proc != nil {
		b = t.proc(b)
	}

	a.cache[name] = &assetCache{
		name:  name,
		mime:  mime.TypeByExtension(t.ext),
		time:  time.Now(),
		paths: paths,
		bytes: b,
	}

	return a.cache[name]
}

func (a *assets) inlinedFromPaths(t *assetType, paths []string) []*assetCache {
	contents := []*assetCache{}

	for _, name := range paths {
		contents = append(contents, a.inlinedFromPath(t, name))
	}

	return contents
}

func (a *assets) inlinedFromPath(t *assetType, name string) *assetCache {
	if cached, ok := a.cache[name]; ok && a.prod {
		return cached
	}

	b := a.bytesFromPaths([]string{name})

	if t.proc != nil {
		b = t.proc(b)
	}

	a.cache[name] = &assetCache{name: name, bytes: b}
	return a.cache[name]
}

func (a *assets) bytesFromPaths(paths []string) []byte {
	hfs := http.Dir(a.dir)
	buf := bytes.NewBuffer(nil)

	for _, name := range a.resolvePaths(paths) {
		a.bufferFromPath(hfs, name, buf)
	}

	return buf.Bytes()
}

func (a *assets) bufferFromPath(dir http.FileSystem, rel string, w io.Writer) {
	if f, err := dir.Open(rel); err == nil {
		if _, err = io.Copy(w, f); err != nil {
			log.Fatalln(err)
		}
		if err := f.Close(); err != nil {
			log.Fatalln(err)
		}
	}
}

func (a *assets) unpackPaths(sources []interface{}) []string {
	files := []string{}

	for _, t := range sources {
		switch t := t.(type) {
		case string:
			files = append(files, t)
		case []interface{}:
			files = append(files, a.unpackPaths(t)...)
		default:
			files = append(files, fmt.Sprintf("%v", t))
		}
	}

	return files
}

func (a *assets) resolvePaths(sources []string) []string {
	files := []string{}

	for _, source := range sources {
		files = append(files, a.resolvePath(source)...)
	}

	return files
}

func (a *assets) resolvePath(source string) []string {
	files := files.List(filepath.Join(a.dir, source))

	for i, abs := range files {
		if rel, err := filepath.Rel(a.dir, abs); err == nil {
			files[i] = a.prefixPath(source, rel)
		}
	}

	return files
}

func (a *assets) prefixPath(source string, rel string) string {
	if strings.HasPrefix(source, "/") && !strings.HasPrefix(rel, "/") {
		rel = "/" + rel
	}

	if runtime.GOOS == "windows" {
		return strings.Replace(rel, "\\", "/", -1)
	}

	return rel
}

func markdown(b []byte) []byte {
	return htmlPolicy.SanitizeBytes(blackfriday.MarkdownCommon(b))
}

func hash(seed string) string {
	return fmt.Sprintf("%x", sha1.Sum([]byte(seed)))[:12]
}

func rooted(root string) string {
	return fmt.Sprintf("%s/%s/", root, hash(timestamp()))
}

func timestamp() string {
	return strconv.FormatInt(time.Now().Unix(), 10)
}
