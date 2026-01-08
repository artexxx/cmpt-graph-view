package main

import (
	"errors"
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	errInvalidInput = errors.New("invalid input")

	badSchemes = []string{
		"http://",
		"https://",
		"mailto:",
		"tel:",
		"javascript:",
		"data:",
	}

	reFenceBackticks = regexp.MustCompile("(?s)```.*?```")
	reFenceTildes    = regexp.MustCompile(`(?s)~~~.*?~~~`)

	// Remove base URL: https://example.com/foo -> /foo
	reStripHost = regexp.MustCompile(`(?i)^https?://[^/]+`)
)

func stripCodeFences(md string) string {
	md = reFenceBackticks.ReplaceAllString(md, "")
	md = reFenceTildes.ReplaceAllString(md, "")
	return md
}

// canonicalPath returns a canonical internal path:
// - leading '/'
// - no query/fragment
// - no trailing '/' except root
// - url-escaped (segment-safe) to match typical Hugo URL encoding
func canonicalPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return "/"
	}

	// strip host if present
	p = reStripHost.ReplaceAllString(p, "")

	// drop query/fragment
	if i := strings.IndexByte(p, '#'); i >= 0 {
		p = p[:i]
	}
	if i := strings.IndexByte(p, '?'); i >= 0 {
		p = p[:i]
	}
	p = strings.TrimSpace(p)
	if p == "" {
		return "/"
	}

	// ensure leading slash
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}

	// normalize separators + clean (posix)
	p = filepath.ToSlash(p)
	p = path.Clean(p)
	if p == "." {
		p = "/"
	}
	if p != "/" && strings.HasSuffix(p, "/") {
		p = strings.TrimSuffix(p, "/")
	}

	// unescape then escape consistently
	if u, err := url.PathUnescape(p); err == nil {
		p = u
	}
	p = unicodeSanitize(p)
	p = url.PathEscape(p)
	p = strings.ReplaceAll(p, "%2F", "/")

	// keep root
	if p == "" {
		return "/"
	}
	return p
}

func isInternalURL(u string) bool {
	u = strings.TrimSpace(u)
	if u == "" {
		return false
	}
	lu := strings.ToLower(u)
	if strings.HasPrefix(lu, "//") {
		return false
	}
	for _, s := range badSchemes {
		if strings.HasPrefix(lu, s) {
			return false
		}
	}
	return true
}

// pageURLFromMarkdown computes a canonical internal URL for a Markdown file under content root.
//
// Examples:
//
//	content/posts/hello-world.md       -> /posts/hello-world
//	content/shortcodes/x/index.md      -> /shortcodes/x
//	content/_index.md                 -> /
//	content/section/_index.md         -> /section
func pageURLFromMarkdown(mdFile string, contentRoot string) (string, error) {
	rel, err := filepath.Rel(contentRoot, mdFile)
	if err != nil {
		return "", fmt.Errorf("rel path: %w", err)
	}
	rel = filepath.ToSlash(rel)
	parts := strings.Split(rel, "/")
	name := strings.ToLower(parts[len(parts)-1])

	switch {
	case strings.HasPrefix(name, "_index"):
		// root _index.md => /
		if len(parts) == 1 {
			return "/", nil
		}
		// section _index.md => /section
		return canonicalPath("/" + strings.Join(parts[:len(parts)-1], "/")), nil

	case strings.HasPrefix(name, "index"):
		// index.md => directory
		if len(parts) == 1 {
			return "/", nil
		}
		return canonicalPath("/" + strings.Join(parts[:len(parts)-1], "/")), nil

	default:
		// normal file => use stem
		stem := strings.TrimSuffix(parts[len(parts)-1], filepath.Ext(parts[len(parts)-1]))
		parts[len(parts)-1] = stem
		return canonicalPath("/" + strings.Join(parts, "/")), nil
	}
}

// buildPageMap maps various reference forms to the canonical URL.
func buildPageMap(contentRoot string, mdFiles []string) (map[string]string, error) {
	m := make(map[string]string, len(mdFiles)*4)

	for _, fp := range mdFiles {
		canon, err := pageURLFromMarkdown(fp, contentRoot)
		if err != nil {
			return nil, fmt.Errorf("page url from markdown %q: %w", fp, err)
		}

		m[canon] = canon
		m[canon+"/"] = canon

		rel, err := filepath.Rel(contentRoot, fp)
		if err != nil {
			return nil, fmt.Errorf("rel path: %w", err)
		}
		rel = filepath.ToSlash(rel)

		m["/"+rel] = canon

		if strings.HasSuffix(strings.ToLower(rel), ".md") {
			noExt := strings.TrimSuffix(rel, ".md")
			m["/"+noExt] = canon
			m["/"+noExt+".md"] = canon
		}
	}

	return m, nil
}

func resolveTarget(rawURL string, sourceURL string, pageMap map[string]string) (string, bool) {
	u := strings.TrimSpace(rawURL)
	if u == "" {
		return "", false
	}
	if strings.HasPrefix(u, "#") {
		return "", false
	}
	if !isInternalURL(u) {
		return "", false
	}

	// unwrap angle brackets
	if strings.HasPrefix(u, "<") && strings.HasSuffix(u, ">") && len(u) >= 2 {
		u = strings.TrimSpace(u[1 : len(u)-1])
	}

	// drop query/fragment
	if i := strings.IndexByte(u, '#'); i >= 0 {
		u = u[:i]
	}
	if i := strings.IndexByte(u, '?'); i >= 0 {
		u = u[:i]
	}
	u = strings.TrimSpace(u)
	if u == "" {
		return "", false
	}

	// absolute
	if strings.HasPrefix(u, "/") {
		key := canonicalPath(u)
		if v, ok := pageMap[key]; ok {
			return v, true
		}
		return key, true
	}

	// relative: resolve against source directory
	base := sourceURL
	if base != "/" {
		if i := strings.LastIndex(base, "/"); i >= 0 {
			base = base[:i]
			if base == "" {
				base = "/"
			}
		}
	}

	joined := path.Clean(path.Join(base, u))
	if !strings.HasPrefix(joined, "/") {
		joined = "/" + joined
	}
	if strings.HasSuffix(strings.ToLower(joined), ".md") {
		joined = strings.TrimSuffix(joined, ".md")
	}
	key := canonicalPath(joined)
	if v, ok := pageMap[key]; ok {
		return v, true
	}
	return key, true
}

func dedupeLinks(in []Link) []Link {
	seen := make(map[string]struct{}, len(in))
	out := make([]Link, 0, len(in))

	for _, l := range in {
		k := l.Source + "\x00" + l.Target + "\x00" + l.Text
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, l)
	}

	// stable output order
	sort.Slice(out, func(i, j int) bool {
		if out[i].Source != out[j].Source {
			return out[i].Source < out[j].Source
		}
		if out[i].Target != out[j].Target {
			return out[i].Target < out[j].Target
		}
		return out[i].Text < out[j].Text
	})

	return out
}

// unicodeSanitize is a minimal port of Hugo's path sanitization rules.
// It keeps common URL/path characters and drops disallowed runes, folding
// spaces into hyphens.
//
// This is intentionally conservative to reduce surprises when content contains
// Unicode; the final canonicalPath step still applies url.PathEscape.
func unicodeSanitize(s string) string {
	// We keep this tiny and predictable; avoid importing unicode tables heavily.
	// Allow letters/digits broadly by default; the subsequent PathEscape will encode them.
	var b strings.Builder
	b.Grow(len(s))

	prependHyphen := false
	for _, r := range s {
		isAllowed := r == '.' || r == '/' || r == '\\' || r == '_' || r == '#' || r == '+' || r == '~' || r == '-' || r == '%'
		if !isAllowed {
			// keep most runes; PathEscape will handle.
			// But collapse whitespace to '-'.
			if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
				prependHyphen = b.Len() > 0
				continue
			}
			// keep rune
			isAllowed = true
		}

		if isAllowed {
			if prependHyphen {
				b.WriteByte('-')
				prependHyphen = false
			}
			b.WriteRune(r)
		}
	}

	return b.String()
}
