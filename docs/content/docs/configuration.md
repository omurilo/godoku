---
title: Configuration
description: All configuration options for godoku.yaml
order: 2
---

# Configuration

Godoku is configured via a `godoku.yaml` file in the root of your project.

## Full Example

```yaml
title: My Documentation
description: Project documentation and API reference
url: https://docs.example.com
language: en
theme: default
redirect: /docs/getting-started

navigation:
  - label: Docs
    path: /docs
  - label: Guides
    path: /guides
  - label: Tutorials
    path: /tutorials
  - label: API
    path: /api

sections:
  docs: content/docs
  guides: content/guides
  tutorials: content/tutorials
```

## Options

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `title` | string | `"Godoku"` | Site title shown in the navbar |
| `description` | string | `"API Documentation"` | Meta description |
| `url` | string | `"http://localhost:3000"` | Site base URL |
| `language` | string | `"en"` | HTML lang attribute |
| `theme` | string | `"default"` | Theme name |
| `redirect` | string | — | Redirect homepage to this path |
| `navigation` | list | — | Navbar links |
| `sections` | object | — | Content directory paths |

## Navigation

Each navigation item has:

- **`label`** — Text displayed in the navbar
- **`path`** — URL path for the link

Empty sections are automatically hidden from the navbar. If a section has no content files, its nav link won't appear.

## Sections

Sections map URL paths to content directories:

```yaml
sections:
  docs: content/docs         # /docs/*
  guides: content/guides     # /guides/*
  tutorials: content/tutorials # /tutorials/*
```

## Redirect

Set `redirect` to automatically redirect the homepage to a specific page:

```yaml
redirect: /docs/getting-started
```

This generates both a static `<meta http-equiv="refresh">` redirect and a 301 redirect in the dev server.
