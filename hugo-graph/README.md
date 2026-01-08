# Linkmap Generator

This tool scans Hugo Markdown content and produces a **full internal link graph** as JSON.

The generated file is meant to be consumed via Hugo `data/` (for example `data/linkmap.json`) and can drive a per-page graph visualization and backlinks list.

## Output format

```json
{
  "content": {
    "/path": { "title": "Article Title" }
  },
  "index": {
    "links": {
      "/source": [
        { "source": "/source", "target": "/target", "text": "Anchor text" }
      ]
    },
    "backlinks": {
      "/target": [
        { "source": "/source", "target": "/target", "text": "Anchor text" }
      ]
    }
  },
  "links": [
    { "source": "/source", "target": "/target", "text": "Anchor text" }
  ]
}
```

## Detected internal links

1. Hugo `relref` shortcodes, including `path=` form:
   - `{{< relref path="/a/b" >}}`
   - `{{< relref "/a/b" >}}`
   - `{{% relref "/a/b" %}}`

2. Reference definitions **(counted as links)**:
   - `[label]: {{< relref path="/x/y" >}}`

3. Markdown links:
   - `[Text](/some/path)`
   - `[Text](../other/page.md)`
   - `[Text](page/)`

4. Reference usages resolved via (2):
   - `[label]`

Backlinks are derived from the complete link list.

## Titles

Front matter is parsed from YAML (`---`) or TOML (`+++`) and the display title is selected with priority:

`title` → `linkTitle` → `shortTitle` → `path`

# GitHub Action

You can run the generator in CI (Docker-based) similarly to “Obsidian Link Scraper”.

## Example workflow step

Add a build step in your workflow (e.g. `.github/workflows/deploy.yml`):

```yaml
- name: Build Linkmap JSON
  uses: ./Themes/cpmt-graph-view/hugo-graph
  with:
    input: content
    output: data/linkmap.json
    pretty: "true"
```

### Notes

* This assumes your action is committed in-repo (local action usage).
* If you publish the generator as a separate action repository later, replace `uses:` with that repo ref.

---

# Typical project flow

Locally:

```bash
# Installation
go install github.com/Artexxx/cmpt-graph-view/hugo-graph@latest

# Run
hugo-graph -input=content -output=data/linkmap.json -pretty
```

CI:

1. Checkout
2. Run generator
3. Build Hugo site (now `data/linkmap.json` exists)
