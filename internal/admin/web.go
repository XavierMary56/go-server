package admin

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static
var staticFiles embed.FS

func (ah *AdminHandler) registerWebUI(mux *http.ServeMux) {
	subFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		ah.log.Error("Web UI 静态文件加载失败: " + err.Error())
		return
	}

	fileServer := http.FileServer(http.FS(subFS))

	mux.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin/", http.StatusMovedPermanently)
	})
	mux.Handle("/admin/", http.StripPrefix("/admin", fileServer))
}
