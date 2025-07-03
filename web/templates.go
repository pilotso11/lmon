package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed templates/*
var templatesFS embed.FS

// GetTemplatesFS returns a filesystem containing the embedded templates
func GetTemplatesFS() embed.FS {
	return templatesFS
}

// GetTemplatesFSWithRoot returns a filesystem with the templates directory as the root
func GetTemplatesFSWithRoot() (fs.FS, error) {
	return fs.Sub(templatesFS, "templates")
}

// GetHTTPFileSystem returns an http.FileSystem for the templates
func GetHTTPFileSystem() http.FileSystem {
	fsys, err := GetTemplatesFSWithRoot()
	if err != nil {
		panic(err)
	}
	return http.FS(fsys)
}