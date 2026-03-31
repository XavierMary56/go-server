package admin

import (
	"bytes"
	"embed"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strings"
)

//go:embed static
var staticFiles embed.FS

func (ah *AdminHandler) registerWebUI(mux *http.ServeMux) {
	var staticFS http.FileSystem

	// 优先从文件系统加载（开发时挂载 volume 可热更新）
	const overrideDir = "/admin-static"
	if info, err := os.Stat(overrideDir); err == nil && info.IsDir() {
		staticFS = http.Dir(overrideDir)
	} else {
		subFS, err := fs.Sub(staticFiles, "static")
		if err != nil {
			ah.log.Error("Web UI 静态文件加载失败: " + err.Error())
			return
		}
		staticFS = http.FS(subFS)
	}

	fileServer := http.FileServer(staticFS)

	mux.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin/", http.StatusMovedPermanently)
	})

	mux.HandleFunc("/admin/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/admin")
		if path == "/" || path == "" || path == "/index.html" {
			ah.serveIndex(w, r, staticFS)
			return
		}
		http.StripPrefix("/admin", fileServer).ServeHTTP(w, r)
	})
}

func (ah *AdminHandler) serveIndex(w http.ResponseWriter, r *http.Request, staticFS http.FileSystem) {
	f, err := staticFS.Open("/index.html")
	if err != nil {
		http.Error(w, "index.html not found", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		http.Error(w, "read error", http.StatusInternalServerError)
		return
	}

	// 优先从数据库读取版本号，回退到固定默认值
	version := "20260401"
	if ah.db != nil {
		if setting, err := ah.db.GetAdminSetting(staticVersionSettingKey); err == nil && setting != nil && strings.TrimSpace(setting.Value) != "" {
			version = strings.TrimSpace(setting.Value)
		}
	}

	data = bytes.ReplaceAll(data, []byte("{{STATIC_VERSION}}"), []byte(version))

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}
