package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static/*
var staticFS embed.FS

//go:embed templates/*
var templateFS embed.FS

// StaticFS 返回静态文件系统
func StaticFS() http.FileSystem {
	subFS, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err)
	}
	return http.FS(subFS)
}

// TemplateFS 返回模板文件系统
func TemplateFS() http.FileSystem {
	subFS, err := fs.Sub(templateFS, "templates")
	if err != nil {
		panic(err)
	}
	return http.FS(subFS)
}
