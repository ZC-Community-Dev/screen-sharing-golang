package web

import (
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
)

func TestSubBundleReturnsValidatedBrowserRoot(t *testing.T) {
	filesystem := fstest.MapFS{
		"dist/browser/index.html": &fstest.MapFile{Data: []byte("<html>ok</html>")},
		"dist/browser/main.js":    &fstest.MapFile{Data: []byte("ok")},
	}
	bundle, err := subBundle(filesystem, "dist/browser")
	if err != nil {
		t.Fatal(err)
	}
	data, err := fs.ReadFile(bundle, "main.js")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "ok" {
		t.Fatalf("main.js=%q", data)
	}
}

func TestSubBundleRejectsMissingIndex(t *testing.T) {
	filesystem := fstest.MapFS{
		"dist/browser/main.js": &fstest.MapFile{Data: []byte("ok")},
	}
	_, err := subBundle(filesystem, "dist/browser")
	if err == nil || !strings.Contains(err.Error(), "frontend bundle") {
		t.Fatalf("expected frontend bundle error, got %v", err)
	}
}
