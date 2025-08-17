package handlers

import (
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/grumpyguvner/gomail/cmd/webadmin/config"
	"github.com/grumpyguvner/gomail/cmd/webadmin/logging"
)

type StaticHandler struct {
	config     *config.Config
	logger     *logging.Logger
	fileSystem http.FileSystem
}

func NewStaticHandler(cfg *config.Config, logger *logging.Logger, fs http.FileSystem) *StaticHandler {
	return &StaticHandler{
		config:     cfg,
		logger:     logger,
		fileSystem: fs,
	}
}

func (h *StaticHandler) ServeStatic() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Clean the path
		urlPath := path.Clean(r.URL.Path)

		// Remove leading slash
		urlPath = strings.TrimPrefix(urlPath, "/")

		// If path is empty, serve index.html
		if urlPath == "" {
			urlPath = "index.html"
		}

		// Try to open the file from embedded filesystem
		file, err := h.fileSystem.Open(urlPath)
		if err != nil {
			// For SPA routing, serve index.html for non-existent files
			// that don't have file extensions (likely routes)
			if !strings.Contains(urlPath, ".") {
				indexFile, indexErr := h.fileSystem.Open("index.html")
				if indexErr != nil {
					http.NotFound(w, r)
					return
				}
				defer indexFile.Close()
				h.setContentType(w, "index.html")
				// Read and serve the file content
				_, _ = io.Copy(w, indexFile)
				return
			}
			// File with extension doesn't exist
			http.NotFound(w, r)
			return
		}
		defer file.Close()

		// Set appropriate content type based on file extension
		h.setContentType(w, urlPath)

		// Read and serve the file content
		_, _ = io.Copy(w, file)
	})
}

func (h *StaticHandler) setContentType(w http.ResponseWriter, filepath string) {
	ext := strings.ToLower(filepath[strings.LastIndex(filepath, "."):])

	switch ext {
	case ".html":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case ".css":
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	case ".js":
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	case ".json":
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".gif":
		w.Header().Set("Content-Type", "image/gif")
	case ".svg":
		w.Header().Set("Content-Type", "image/svg+xml")
	case ".ico":
		w.Header().Set("Content-Type", "image/x-icon")
	case ".woff":
		w.Header().Set("Content-Type", "font/woff")
	case ".woff2":
		w.Header().Set("Content-Type", "font/woff2")
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}
}
