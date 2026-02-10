// internal/webui/serve.go
package webui

import (
	"io/fs"
	"net/http"
)

func Handler() http.Handler {
	sub, err := fs.Sub(FS, "dist")
	if err != nil {
		panic(err)
	}

	return http.FileServer(http.FS(sub))
}
