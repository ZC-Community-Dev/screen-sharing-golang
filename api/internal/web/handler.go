package web

import (
	"bytes"
	"fmt"
	"io/fs"
	"mime"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"
)

var hashedAsset = regexp.MustCompile(`(?:[.-])[A-Za-z0-9_-]{6,}\.[A-Za-z0-9]+$`)

type Handler struct {
	files fs.FS
	index []byte
}

func NewHandler(files fs.FS) (http.Handler, error) {
	if files == nil {
		return nil, fmt.Errorf("frontend bundle is missing")
	}
	if containsForbiddenFile(files) {
		return nil, fmt.Errorf("frontend bundle contains a forbidden file")
	}
	info, err := fs.Stat(files, "index.html")
	if err != nil || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("frontend bundle is missing index.html")
	}
	index, err := fs.ReadFile(files, "index.html")
	if err != nil {
		return nil, fmt.Errorf("frontend bundle index.html is unreadable")
	}
	return &Handler{files: files, index: index}, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.NotFound(w, r)
		return
	}
	if unsafePath(r.URL.EscapedPath()) {
		http.NotFound(w, r)
		return
	}

	name := cleanName(r.URL.Path)
	if name == "" {
		h.serve(w, r, "index.html", h.index)
		return
	}

	info, err := fs.Stat(h.files, name)
	if err != nil || !info.Mode().IsRegular() {
		if path.Ext(name) != "" {
			http.NotFound(w, r)
			return
		}
		h.serve(w, r, "index.html", h.index)
		return
	}
	data, err := fs.ReadFile(h.files, name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	h.serve(w, r, name, data)
}

func (h *Handler) serve(w http.ResponseWriter, r *http.Request, name string, data []byte) {
	contentType := mime.TypeByExtension(path.Ext(name))
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	w.Header().Set("Content-Type", contentType)
	if name == "index.html" {
		w.Header().Set("Cache-Control", "no-cache")
	} else if hashedAsset.MatchString(path.Base(name)) {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	}
	http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(data))
}

func cleanName(urlPath string) string {
	cleaned := path.Clean("/" + urlPath)
	return strings.TrimPrefix(cleaned, "/")
}

func unsafePath(escapedPath string) bool {
	decoded, err := url.PathUnescape(escapedPath)
	if err != nil || strings.Contains(decoded, `\`) {
		return true
	}
	for _, segment := range strings.Split(decoded, "/") {
		if segment == ".." {
			return true
		}
	}
	return false
}

func containsForbiddenFile(files fs.FS) bool {
	forbidden := false
	_ = fs.WalkDir(files, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		base := strings.ToLower(path.Base(name))
		if base == ".env" ||
			strings.HasPrefix(base, ".env.") ||
			path.Ext(base) == ".db" ||
			strings.Contains(base, "link_id_salt") ||
			strings.Contains(base, "presenter_token") {
			forbidden = true
			return fs.SkipAll
		}
		return nil
	})
	return forbidden
}
