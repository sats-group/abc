package web

import (
	"net/http"
)

func http403(rw http.ResponseWriter, r *http.Request) {
	http.Error(rw, "403 Forbidden", http.StatusForbidden)
}

func http404(rw http.ResponseWriter, r *http.Request) {
	http.Error(rw, "404 Not Found", http.StatusNotFound)
}

func http500(rw http.ResponseWriter, r *http.Request) {
	http.Error(rw, "500 Internal Server Error", http.StatusInternalServerError)
}
