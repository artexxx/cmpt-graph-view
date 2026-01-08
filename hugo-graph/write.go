package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type FileWriter interface {
	MkdirAll(path string, perm os.FileMode) error
	WriteFile(name string, data []byte, perm os.FileMode) error
}

type OSWriter struct{}

func (OSWriter) MkdirAll(p string, perm os.FileMode) error { return os.MkdirAll(p, perm) }
func (OSWriter) WriteFile(name string, data []byte, perm os.FileMode) error {
	return os.WriteFile(name, data, perm)
}

func buildIndex(links []Link) Index {
	linkMap := make(map[string][]Link)
	backMap := make(map[string][]Link)

	for _, l := range links {
		linkMap[l.Source] = append(linkMap[l.Source], l)
		backMap[l.Target] = append(backMap[l.Target], l)
	}

	return Index{Links: linkMap, Backlinks: backMap}
}

func writeJSON(w FileWriter, outPath string, lm LinkMap, pretty bool) error {
	if strings.TrimSpace(outPath) == "" {
		return fmt.Errorf("%w: output path is empty", errInvalidInput)
	}

	dir := filepath.Dir(outPath)
	if dir != "." && dir != "" {
		if err := w.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("mkdir %q: %w", dir, err)
		}
	}

	var b []byte
	var err error
	if pretty {
		b, err = json.MarshalIndent(lm, "", "  ")
	} else {
		b, err = json.Marshal(lm)
	}
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}

	if err := w.WriteFile(outPath, b, 0o644); err != nil {
		return fmt.Errorf("write %q: %w", outPath, err)
	}

	return nil
}
