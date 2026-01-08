# cmpt-graph-view Linkmap Generator

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

## Usage

From the repository root:

```bash
cd hugo-graph
go run . -input ../content -output ../data/linkmap.json -pretty
```


Or build a binary:

```bash
cd hugo-graph
go build -o linkmap-gen .
./linkmap-gen -input ../content -output ../data/linkmap.json
```

## GitHub Action

You can run the generator in CI using the included `action.yml` (Docker-based).
