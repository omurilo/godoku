---
title: "Getting Started"
description: "Learn how to get started with your documentation"
order: 1
---

# Getting Started

Welcome to your documentation site powered by **Godoku**!

## Quick Start

1. Edit files in the `content/` directory
2. Run `godoku build` to generate your site
3. Run `godoku serve -w` for development with live reload

## Writing Content

Create `.md` files in the appropriate content directory:

- `content/docs/` serves at `/docs/`
- `content/guides/` serves at `/guides/`
- `content/tutorials/` serves at `/tutorials/`

## Frontmatter

Each markdown file supports YAML frontmatter:

```yaml
---
title: "Page Title"
description: "Page description"
date: "2026-01-01"
author: "Author Name"
order: 1
draft: false
---
```

## API Documentation

Add your OpenAPI spec files and reference them in `godoku.yaml`:

```yaml
openapi:
  - openapi.yaml
```

Your API reference will be automatically generated at `/api/`.
