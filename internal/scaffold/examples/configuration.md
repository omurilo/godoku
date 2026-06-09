---
title: "Configuration Guide"
description: "How to configure your Godoku project"
order: 1
---

# Configuration Guide

Godoku is configured via the `godoku.yaml` file in your project root.

## Basic Configuration

```yaml
title: "My Project"
description: "Project Documentation & API Reference"
url: "https://docs.example.com"
language: "en"

sections:
  docs: "content/docs"
  guides: "content/guides"
  tutorials: "content/tutorials"

navigation:
  - label: "Docs"
    path: "/docs"
  - label: "Guides"
    path: "/guides"
  - label: "Tutorials"
    path: "/tutorials"
  - label: "API"
    path: "/api"
```

## Sections

Each section maps to a directory of markdown files and a URL prefix.

## OpenAPI Integration

Godoku parses OpenAPI v3 / Swagger specifications and generates beautiful API reference pages automatically.

Any `.yaml`, `.yml` or `.json` spec dropped into the `apis/` directory is **auto-discovered** — no configuration required. A single spec is served at `/api`; multiple specs get a catalog at `/api` and each spec at `/api/{slug}`, where the slug defaults to the file name.

### Configuring APIs

The `apis` section is optional and only needed when you want to **control ordering** or **override** the metadata derived from a spec. Listed specs come first, in the order you declare them; any remaining auto-discovered specs follow, sorted by file name.

```yaml
apis:
  - spec: apis/payments.yaml      # path relative to the project root (or a bare file name)
    slug: payments                # overrides the slug used in the URL (/api/payments)
    title: "Payments API"         # overrides info.title from the spec
    description: "Charge, refund and reconcile."
  - spec: users.yaml              # only reorders it; metadata comes from the spec
```

You do not need to list every spec — unlisted ones are still discovered and appended at the end.
