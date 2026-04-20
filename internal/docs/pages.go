package docs

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// PageMeta describes a doc page without rendering it.
type PageMeta struct {
	Slug  string `json:"slug"`
	Title string `json:"title"`
	Path  string `json:"path"`
}

// ListPages returns the metadata for every *.md under pagesFS, sorted by slug.
// pagesFS should be rooted at the pages directory (e.g. embed/pages).
func ListPages(pagesFS fs.FS) ([]PageMeta, error) {
	var pages []PageMeta
	err := fs.WalkDir(pagesFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}
		data, err := fs.ReadFile(pagesFS, path)
		if err != nil {
			return err
		}
		slug := strings.TrimSuffix(path, ".md")
		pages = append(pages, PageMeta{
			Slug:  slug,
			Title: firstH1OrSlug(data, slug),
			Path:  path,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(pages, func(i, j int) bool { return pages[i].Slug < pages[j].Slug })
	return pages, nil
}

func firstH1OrSlug(data []byte, slug string) string {
	if t := firstH1(data); t != "" {
		return t
	}
	return slug
}
