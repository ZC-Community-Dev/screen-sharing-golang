package web

import (
	"embed"
	"fmt"
	"io/fs"
)

//go:embed all:dist
var embedded embed.FS

func Embedded() (fs.FS, error) {
	return subBundle(embedded, "dist/browser")
}

func subBundle(files fs.FS, root string) (fs.FS, error) {
	bundle, err := fs.Sub(files, root)
	if err != nil {
		return nil, fmt.Errorf("frontend bundle is missing")
	}
	info, err := fs.Stat(bundle, "index.html")
	if err != nil || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("frontend bundle is missing index.html")
	}
	return bundle, nil
}
