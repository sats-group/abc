// Package tmpl provides HTML template helper funcs.
package tmpl

import (
	"html/template"
	"strings"

	"github.com/gosimple/slug"
)

// Funcs holds all template helpers.
var Funcs = template.FuncMap{
	"join":     Join,
	"noescape": Noescape,
	"slug":     Slug,
	"title":    Title,
	"when":     When,
	"yield":    Yield,
}

// Join concatenates elements with a separator.
func Join(s []string, sep string) string {
	return strings.Join(s, sep)
}

// Noescape allows HTML from a plain string.
func Noescape(s string) template.HTML {
	return template.HTML(s)
}

// Slug formats a string to a URL-friendly format.
func Slug(s string) string {
	return slug.Make(s)
}

// Title capitalizes the first letter of each word.
func Title(s string) string {
	return strings.Title(s)
}

// When returns one alternative based on a condition.
func When(condition bool, t, f interface{}) interface{} {
	if condition {
		return t
	}
	return f
}

// Yield returns an empty HTML string.
func Yield() template.HTML {
	return template.HTML("")
}
