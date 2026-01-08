package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Link struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Text   string `json:"text"`
}

type Index struct {
	Links     map[string][]Link `json:"links"`
	Backlinks map[string][]Link `json:"backlinks"`
}

type Page struct {
	Title string `json:"title"`
}

type LinkMap struct {
	Content map[string]Page `json:"content"`
	Index   Index           `json:"index"`
	Links   []Link          `json:"links"`
}

func main() {
	var (
		in     = flag.String("input", "content", "Input Hugo content directory")
		out    = flag.String("output", filepath.FromSlash("data/linkmap.json"), "Output JSON file path")
		pretty = flag.Bool("pretty", false, "Pretty JSON (indent=2)")
	)
	flag.Parse()

	if strings.TrimSpace(*in) == "" {
		_, _ = fmt.Fprintln(os.Stderr, "input directory is required")
		os.Exit(2)
	}

	fsys := OSFS{}
	writer := OSWriter{}

	lm, err := generateLinkMap(fsys, *in)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if err := writeJSON(writer, *out, lm, *pretty); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	_, _ = fmt.Fprintf(os.Stdout, "[OK] Wrote %s (%d links, %d pages)\n", *out, len(lm.Links), len(lm.Content))
}
