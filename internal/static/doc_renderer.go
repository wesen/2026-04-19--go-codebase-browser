// Package static performs build-time pre-computation for the static WASM build.
package static

import (
	"fmt"
	"io/fs"

	"github.com/wesen/codebase-browser/internal/browser"
	"github.com/wesen/codebase-browser/internal/docs"
)

// DocRenderer pre-renders all documentation pages to static HTML.
type DocRenderer struct {
	Loaded   *browser.Loaded
	SourceFS fs.FS
	PagesFS  fs.FS
}

// RenderAll renders every doc page and returns the manifest + HTML map.
func (r *DocRenderer) RenderAll() (manifest []docs.PageMeta, html map[string]string, errors []string, err error) {
	pages, err := docs.ListPages(r.PagesFS)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("list pages: %w", err)
	}

	html = make(map[string]string)

	for _, page := range pages {
		data, err := fs.ReadFile(r.PagesFS, page.Path)
		if err != nil {
			errors = append(errors, fmt.Sprintf("read %s: %v", page.Path, err))
			continue
		}

		rendered, err := docs.Render(page.Slug, data, r.Loaded, r.SourceFS)
		if err != nil {
			errors = append(errors, fmt.Sprintf("render %s: %v", page.Slug, err))
			continue
		}

		manifest = append(manifest, docs.PageMeta{
			Slug:  page.Slug,
			Title: rendered.Title,
			Path:  page.Path,
		})
		html[page.Slug] = rendered.HTML
	}

	return manifest, html, errors, nil
}
