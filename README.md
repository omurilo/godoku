# Godoku

A fast, zero-dependency static site generator for documentation and API references, built in Go.

Godoku takes your Markdown files and OpenAPI v3 specs and generates a beautiful, dark-themed documentation site — complete with sidebar navigation, syntax highlighting, and an interactive API playground.

## Features

- **Markdown-powered** — Write docs in Markdown with GFM extensions (tables, task lists, strikethrough, autolinks)
- **OpenAPI v3** — Auto-discover specs from `apis/` and generate full API reference pages with `$ref` resolution
- **API Playground** — Interactive request builder on every endpoint page
- **Nested sections** — Subdirectories become sidebar groups with customizable ordering
- **Nav ordering** — Control page order with `nav` lists in `_index.md` or `order` frontmatter
- **Prev/Next navigation** — Automatic page-to-page links across sections and groups
- **Dev server** — Live rebuild on file changes with `godoku serve -w`
- **Single binary** — No runtime dependencies, templates and assets are embedded
- **Fast** — Builds in milliseconds

## Quick Start

```bash
# Install
go install github.com/omurilo/godoku/cmd/godoku@latest

# Create a new project
mkdir my-docs && cd my-docs
godoku init .

# Start the dev server
godoku serve -w
```

Open `http://localhost:3000` to see your site.

## Project Structure

```
my-docs/
├── apis/                    # OpenAPI specs (auto-discovered)
│   └── openapi.yaml
├── content/
│   ├── docs/                # /docs/* pages
│   │   ├── _index.md        # Section nav ordering
│   │   ├── getting-started.md
│   │   └── advanced/        # Sidebar group
│   │       ├── _index.md    # Group metadata & nav ordering
│   │       └── plugins.md
│   ├── guides/              # /guides/* pages
│   └── tutorials/           # /tutorials/* pages
└── godoku.yaml              # Site configuration
```

## Configuration

```yaml
title: My Documentation
description: Project docs and API reference
url: https://docs.example.com
language: en
redirect: /docs/getting-started

navigation:
  - label: Docs
    path: /docs
  - label: API
    path: /api

sections:
  docs: content/docs
  guides: content/guides
  tutorials: content/tutorials
```

## CLI

| Command | Description |
|---------|-------------|
| `godoku init [path]` | Scaffold a new project |
| `godoku build` | Generate static site to `public/` |
| `godoku serve [-w] [-p port]` | Dev server with optional watch mode |
| `godoku version` | Print version |

## Documentation

Full documentation is available at [omurilo.github.io/godoku](https://omurilo.github.io/godoku).

## License

[MIT](LICENSE)
