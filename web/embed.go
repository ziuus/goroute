package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:dist
var content embed.FS

func Handler() http.Handler {
	distFS, err := fs.Sub(content, "dist")
	if err != nil {
		panic(err)
	}
	return http.FileServer(http.FS(distFS))
}
