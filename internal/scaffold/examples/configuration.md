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

openapi:
  - openapi.yaml

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
