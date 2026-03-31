package admin

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
)

//go:embed static
var staticFiles embed.FS

func (ah *AdminHandler) registerWebUI(mux *http.ServeMux) {
	var fileServer http.Handler

	// 优先从文件系统加载（开发时挂载 volume 可热更新）
	const overrideDir = "/admin-static"
	if info, err := os.Stat(overrideDir); err == nil && info.IsDir() {
		fileServer = http.FileServer(http.Dir(overrideDir))
	} else {
		subFS, err := fs.Sub(staticFiles, "static")
		if err != nil {
			ah.log.Error("Web UI 静态文件加载失败: " + err.Error())
			return
		}
		fileServer = http.FileServer(http.FS(subFS))
	}

	mux.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin/", http.StatusMovedPermanently)
	})
	mux.Handle("/admin/", http.StripPrefix("/admin", fileServer))
}
