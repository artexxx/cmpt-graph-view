package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type FS interface {
	ReadFile(name string) ([]byte, error)
	WalkDir(root string, fn fs.WalkDirFunc) error
}

type OSFS struct{}

func (OSFS) ReadFile(name string) ([]byte, error) { return os.ReadFile(name) }
func (OSFS) WalkDir(root string, fn fs.WalkDirFunc) error {
	return filepath.WalkDir(root, fn)
}

func collectMarkdownFiles(fsys FS, contentRoot string) ([]string, error) {
	var files []string
	err := fsys.WalkDir(contentRoot, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk %q: %w", p, err)
		}
		if d.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(d.Name()), ".md") {
			files = append(files, p)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

func generateLinkMap(fsys FS, contentRoot string) (LinkMap, error) {
	mdFiles, err := collectMarkdownFiles(fsys, contentRoot)
	if err != nil {
		return LinkMap{}, fmt.Errorf("collect markdown files: %w", err)
	}

	pageMap, err := buildPageMap(contentRoot, mdFiles)
	if err != nil {
		return LinkMap{}, fmt.Errorf("build page map: %w", err)
	}

	content := make(map[string]Page, len(mdFiles))
	allLinks := make([]Link, 0, 256)

	for _, fp := range mdFiles {
		sourceURL, err := pageURLFromMarkdown(fp, contentRoot)
		if err != nil {
			return LinkMap{}, fmt.Errorf("page url for %q: %w", fp, err)
		}

		b, err := fsys.ReadFile(fp)
		if err != nil {
			return LinkMap{}, fmt.Errorf("read %q: %w", fp, err)
		}
		md := string(b)

		title := extractDisplayTitle(md, sourceURL)
		content[sourceURL] = Page{Title: title}

		links := extractLinks(md, sourceURL, pageMap)
		allLinks = append(allLinks, links...)
	}

	allLinks = dedupeLinks(allLinks)
	idx := buildIndex(allLinks)

	return LinkMap{
		Content: content,
		Index:   idx,
		Links:   allLinks,
	}, nil
}
