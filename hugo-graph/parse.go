package main

import (
	"regexp"
	"strings"
)

func firstNonEmpty(groups []string, idx1 int, idx2 int) string {
	if idx1 >= 0 && idx1 < len(groups) && groups[idx1] != "" {
		return groups[idx1]
	}
	if idx2 >= 0 && idx2 < len(groups) && groups[idx2] != "" {
		return groups[idx2]
	}
	return ""
}

var (
	// 1) Hugo relref shortcodes:
	//    {{< relref path="/a/b" >}}, {{< relref "/a/b" >}}, {{% relref "/a/b" %}}
	reRelrefAny = regexp.MustCompile(`(?is)\{\{\s*(?:<|%)\s*relref(?:\s+path\s*=\s*)?\s*(?:"([^"]+)"|\x27([^\x27]+)\x27)\s*(?:>|%)\s*\}\}`)

	// 2) Reference definitions (COUNTED AS LINKS):
	//    [label]: {{< relref path="/x/y" >}}
	reRefDefLine = regexp.MustCompile(`(?im)^\s*\[([^\]]+)\]\s*:\s*(.*)$`)

	// 3) Markdown links [Text](url)
	reMDLink = regexp.MustCompile(`(?is)\[([^\]]+)\]\(\s*([^) \t\r\n]+)(?:\s+["'][^"']*["'])?\s*\)`)

	// YAML/TOML front matter blocks at file start.
	reYAMLFm = regexp.MustCompile(`(?s)\A---\s*\n(.*?)\n---\s*\n`)
	reTOMLFm = regexp.MustCompile(`(?s)\A\+\+\+\s*\n(.*?)\n\+\+\+\s*\n`)

	reYAMLTitle = regexp.MustCompile(`(?m)^\s*(title|linkTitle|shortTitle)\s*:\s*(.+?)\s*$`)
	reTOMLTitle = regexp.MustCompile(`(?m)^\s*(title|linkTitle|shortTitle)\s*=\s*(".*?"|'.*?'|.+?)\s*$`)
)

func extractDisplayTitle(md string, fallback string) string {
	titleMap := map[string]string{}

	if m := reYAMLFm.FindStringSubmatch(md); len(m) == 2 {
		fm := m[1]
		matches := reYAMLTitle.FindAllStringSubmatch(fm, -1)
		for _, mm := range matches {
			if len(mm) != 3 {
				continue
			}
			titleMap[mm[1]] = unquote(mm[2])
		}
	} else if m := reTOMLFm.FindStringSubmatch(md); len(m) == 2 {
		fm := m[1]
		matches := reTOMLTitle.FindAllStringSubmatch(fm, -1)
		for _, mm := range matches {
			if len(mm) != 3 {
				continue
			}
			titleMap[mm[1]] = unquote(mm[2])
		}
	}

	for _, k := range []string{"title", "linkTitle", "shortTitle"} {
		if v := strings.TrimSpace(titleMap[k]); v != "" {
			return v
		}
	}

	return fallback
}

func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

func extractLinks(md string, sourceURL string, pageMap map[string]string) []Link {
	mdClean := stripCodeFences(md)

	links := make([]Link, 0, 16)

	// 1) Reference definitions: [label]: ...relref...
	refdefs := map[string]string{} // lower(label) -> target
	refMatches := reRefDefLine.FindAllStringSubmatch(mdClean, -1)
	for _, m := range refMatches {
		if len(m) != 3 {
			continue
		}
		label := strings.TrimSpace(m[1])
		body := m[2]

		rm := reRelrefAny.FindStringSubmatch(body)
		if len(rm) != 3 {
			continue
		}

		rawPath := strings.TrimSpace(firstNonEmpty(rm, 1, 2))
		if tgt, ok := resolveTarget(rawPath, sourceURL, pageMap); ok {
			refdefs[strings.ToLower(label)] = tgt
			// definition itself is a link
			links = append(links, Link{Source: sourceURL, Target: tgt, Text: label})
		}
	}

	// 2) Inline relref anywhere
	relrefs := reRelrefAny.FindAllStringSubmatch(mdClean, -1)
	for _, m := range relrefs {
		if len(m) != 3 {
			continue
		}
		rawPath := strings.TrimSpace(firstNonEmpty(m, 1, 2))
		if tgt, ok := resolveTarget(rawPath, sourceURL, pageMap); ok {
			links = append(links, Link{Source: sourceURL, Target: tgt, Text: "relref"})
		}
	}

	// 3) Markdown links [Text](url)
	mdLinks := reMDLink.FindAllStringSubmatch(mdClean, -1)
	for _, m := range mdLinks {
		if len(m) != 3 {
			continue
		}
		text := strings.TrimSpace(m[1])
		rawURL := strings.TrimSpace(m[2])
		if tgt, ok := resolveTarget(rawURL, sourceURL, pageMap); ok {
			links = append(links, Link{Source: sourceURL, Target: tgt, Text: text})
		}
	}

	// 4) Reference usages [label] -> resolve via (1)
	if len(refdefs) > 0 {
		links = append(links, extractReferenceUsages(mdClean, sourceURL, refdefs)...)
	}

	// canonicalize
	for i := range links {
		links[i].Source = canonicalPath(links[i].Source)
		links[i].Target = canonicalPath(links[i].Target)
	}

	return links
}

func extractReferenceUsages(md string, sourceURL string, refdefs map[string]string) []Link {
	// Go regexp does not support lookbehind; do a simple bracket scan with filtering.
	out := make([]Link, 0, 8)

	// Find all "[...]" occurrences (non-greedy).
	re := regexp.MustCompile(`\[[^\]\r\n]+\]`)
	locs := re.FindAllStringIndex(md, -1)
	for _, loc := range locs {
		if len(loc) != 2 {
			continue
		}
		start, end := loc[0], loc[1]
		// Exclude images: ![alt]
		if start > 0 && md[start-1] == '!' {
			continue
		}

		// Exclude reference definitions: [label]: ...
		after := md[end:]
		after = strings.TrimLeft(after, " \t")
		if strings.HasPrefix(after, ":") {
			continue
		}

		label := strings.TrimSpace(md[start+1 : end-1])
		if label == "" {
			continue
		}
		if tgt, ok := refdefs[strings.ToLower(label)]; ok {
			out = append(out, Link{Source: sourceURL, Target: tgt, Text: label})
		}
	}

	return out
}
