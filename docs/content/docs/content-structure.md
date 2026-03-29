---
title: Content Structure
description: How to organize Markdown content with frontmatter, sections, and nested directories
order: 3
---

# Content Structure

Godoku organizes content into **sections** (docs, guides, tutorials), each backed by a directory of Markdown files.

## Frontmatter

Every Markdown file can include YAML frontmatter:

```markdown
---
title: My Page
description: A brief description
order: 1
date: "2024-01-15"
author: Jane Doe
draft: true
---

# My Page Content
```

| Field | Type | Description |
|-------|------|-------------|
| `title` | string | Page title (defaults to filename) |
| `description` | string | Short description shown in page cards |
| `order` | int | Sort position (lower = first) |
| `date` | string | Publication date |
| `author` | string | Author name |
| `draft` | bool | If `true`, the page is excluded from the build |

## Nested Directories

Subdirectories within a section become **groups** in the sidebar:

```
content/docs/
├── getting-started.md      # Root page
├── basics/
│   ├── _index.md           # Group metadata
│   ├── intro.md
│   └── setup.md
└── advanced/
    ├── _index.md
    └── plugins.md
```

This generates a sidebar like:

- Getting Started
- **Basics**
  - Introduction
  - Setup
- **Advanced**
  - Plugins

### Group Metadata (`_index.md`)

Each group directory can have an `_index.md` file to define metadata:

```yaml
---
title: Basics
order: 1
nav:
  - setup
  - intro
---
```

| Field | Type | Description |
|-------|------|-------------|
| `title` | string | Group title in the sidebar |
| `order` | int | Group sort position |
| `nav` | list | Explicit page ordering by slug |

## Navigation Ordering

Pages and groups are sorted by:

1. **`nav` list** in `_index.md` — items listed come first, in order
2. **`order` frontmatter** — ascending numeric order
3. **Alphabetical** — by title as fallback

### Section-level nav

You can also create a `_index.md` at the section root to order both root pages and groups:

```yaml
# content/docs/_index.md
---
nav:
  - getting-started
  - basics
  - advanced
---
```

## Markdown Features

Godoku uses [Goldmark](https://github.com/yuin/goldmark) with GFM extensions:

- **Tables** — GitHub-flavored tables
- **Strikethrough** — `~~text~~`
- **Autolinks** — URLs automatically become links
- **Task lists** — `- [ ]` and `- [x]`
- **Syntax highlighting** — Fenced code blocks with Dracula theme
- **Auto heading IDs** — Headings get anchor IDs for linking
- **Raw HTML** — Inline HTML is supported

### Code Block Enhancements

Godoku supports advanced code block features inspired by modern documentation frameworks.

#### Title

Add a title bar to code blocks with `title="filename"`:

```yaml title="godoku.yaml"
site_name: My Docs
base_url: https://example.com
theme:
  accent_color: "#58a6ff"
```

#### Line Highlighting

Highlight specific lines using `{1,3-5}`:

```go {1,5-8}
package main

import "fmt"

func main() {
    fmt.Println("Hello, Godoku!")
}
```

#### Line Numbers

Show line numbers with `showLineNumbers`:

```js showLineNumbers
function greet(name) {
  return `Hello, ${name}!`;
}

const message = greet("World");
console.log(message);
```

#### Combined

All options can be combined:

```go {1,5-8} title="main.go" showLineNumbers
package main

import "fmt"

func main() {
    fmt.Println("Hello from Godoku!")
    fmt.Println("Advanced code blocks!")
}
```
