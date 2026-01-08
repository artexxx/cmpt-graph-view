# cmpt-graph-view

A minimal graph-view component for Hugo sites.

## What it does

- Renders an interactive link graph (D3)
- Default: 1-hop neighborhood for the current page
- Toggle: full graph
- Toggle: backlinks list (titles only)
- Mouse wheel zoom (cursor over graph)

## Requirements

- Hugo Extended >= 0.154.1
- A generated `data/linkmap.json` available as `site.Data.linkmap`

## Install Component

The installation method is the same as [installing a theme](https://fixit.lruihao.cn/documentation/installation/). There
are several ways to install, choose one, for example, install through Hugo Modules:

```diff
[module]
  [[module.imports]]
    path = "github.com/hugo-fixit/FixIt"
+ [[module.imports]]
+   path = "github.com/Artexxx/cmpt-graph-view"
```

## Configuration

In order to Inject the partial `cmpt-graph-view.html` into the `custom-head` through
the [custom block](https://fixit.lruihao.cn/references/blocks/) opened by the FixIt theme in the
`layouts/_partials/custom.html` file, you need to fill in the following necessary configurations:

```toml
[params]
[params.customPartials]
# ... other partials
head = ["inject/cmpt-graph-view.html"]
# ... other partials
```
