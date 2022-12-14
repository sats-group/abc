// Package files contains specialized helpers for working with files.
package files

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// Matchers for ignored paths.
var patterns = []*regexp.Regexp{
	regexp.MustCompile(`^_`),
	regexp.MustCompile(`/_`),
	regexp.MustCompile(`^\.[^.]`),
	regexp.MustCompile(`/\.[^.]`),
}

// Ignore checks if a path includes leading dots or underscores.
func Ignore(path string) bool {
	for _, pattern := range patterns {
		if pattern.MatchString(path) {
			return true
		}
	}

	return false
}

// Name will extract the filename, without the extension, from a path.
func Name(source string) string {
	return path.Base(strings.TrimSuffix(source, filepath.Ext(source)))
}

// List will list all paths to files in the given source directory.
func List(source string) []string {
	files := []string{}

	if Ignore(source) {
		return files
	}

	if HasFile(source) {
		return []string{source}
	}

	if !HasDir(source) {
		return files
	}

	infos, err := ioutil.ReadDir(source)

	if err != nil {
		return files
	}

	for _, info := range infos {
		rel := path.Join(source, info.Name())

		if Ignore(rel) {
			continue
		}

		if info.IsDir() {
			files = append(files, List(rel)...)
			continue
		}

		files = append(files, rel)
	}

	return files
}

// ListType will list all paths to files with the given extension.
func ListType(source, ext string) []string {
	dext := "." + strings.TrimPrefix(ext, ".")
	matches := []string{}

	for _, file := range List(source) {
		if path.Ext(file) == dext {
			matches = append(matches, file)
		}
	}

	return matches
}

// Copy will copy a file from source to target.
func Copy(source, target string) error {
	if Ignore(source) || Ignore(target) {
		return nil
	}

	input, err := os.Open(source)

	if err != nil {
		return err
	}

	if err := MkdirAll(path.Dir(target)); err != nil {
		return err
	}

	output, err := os.Create(target)

	if err != nil {
		return err
	}

	if _, err := io.Copy(output, input); err != nil {
		return err
	}

	if err := input.Close(); err != nil {
		return err
	}

	if err := output.Close(); err != nil {
		return err
	}

	return nil
}

// Drain will empty a source reader and save it to the target path.
func Drain(target string, source io.Reader) error {
	if Ignore(target) {
		return nil
	}

	if err := MkdirAll(path.Dir(target)); err != nil {
		return err
	}

	file, err := os.Create(target)

	if err != nil {
		return err
	}

	if _, err := io.Copy(file, source); err != nil {
		return err
	}

	if err := file.Close(); err != nil {
		return err
	}

	return nil
}

// Read will read the contents of a file.
func Read(source string) []byte {
	bytes, err := ioutil.ReadFile(source)

	if err != nil {
		return []byte{}
	}

	return bytes
}

// Write will write data to a target file.
func Write(target string, data []byte) error {
	return ioutil.WriteFile(target, data, os.ModePerm)
}

// HasDir checks if a dir exists at the given path.
func HasDir(source string) bool {
	stat, err := os.Stat(source)

	if err != nil {
		return false
	}

	return stat.IsDir()
}

// HasFile checks if a file exists at the given path.
func HasFile(source string) bool {
	stat, err := os.Stat(source)

	if err != nil {
		return false
	}

	return !stat.IsDir()
}

// MkdirAll will create a directory tree if it does not exist.
func MkdirAll(target string) error {
	if !HasDir(target) {
		if err := os.MkdirAll(target, os.ModePerm); err != nil {
			return err
		}
	}

	return nil
}

// Source gets file info for an input file.
func Source(source string) (os.FileInfo, error) {
	file, err := os.Stat(source)

	if err != nil {
		return nil, err
	}

	if !file.Mode().IsRegular() {
		return nil, fmt.Errorf("irregular file: %s", source)
	}

	return file, nil
}

// Target gets file info for an output file.
func Target(target string) (os.FileInfo, error) {
	file, err := os.Stat(target)

	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return file, nil
}

// Walk traverses files and takes care of common scenarios.
func Walk(source string, fn func(string) error) error {
	return filepath.Walk(source,
		func(abs string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info == nil || info.IsDir() {
				return nil
			}

			rel, err := filepath.Rel(source, abs)

			if err != nil {
				return err
			}

			return fn(rel)
		})
}

// TempDir creates a temporary directory for a callback.
func TempDir(name string, fn func(string)) {
	dir, err := ioutil.TempDir(os.TempDir(), "temp-dir")

	if err != nil {
		return
	}

	fn(dir)

	if os.RemoveAll(dir) != nil {
		return
	}
}

// TempFile creates a temporary file for a callback.
func TempFile(name string, data string, fn func(string)) {
	dir, err := ioutil.TempDir(os.TempDir(), "temp-file")

	if err != nil {
		return
	}

	abs := path.Join(dir, name)
	bit := []byte(data)

	if err := ioutil.WriteFile(abs, bit, os.ModePerm); err != nil {
		return
	}

	fn(dir)

	if err := os.RemoveAll(dir); err != nil {
		return
	}
}
