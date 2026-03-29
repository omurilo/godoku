---
title: Getting Started
description: Install Godoku and create your first documentation site in minutes
order: 1
---

# Getting Started

Godoku is a fast, zero-dependency static site generator built in Go, designed for documentation sites and API references. It takes Markdown files and OpenAPI specs and generates a beautiful, dark-themed static site.

## Installation

### From source

```bash
go install github.com/omurilo/godoku/cmd/godoku@latest
```

### From releases

Download the latest binary from the [GitHub Releases](https://github.com/omurilo/godoku/releases) page.

## Quick Start

### 1. Initialize a new project

```bash
mkdir my-docs && cd my-docs
godoku init .
```

This creates the following structure:

```
my-docs/
├── apis/
│   └── openapi.yaml        # Example OpenAPI spec
├── content/
│   ├── docs/
│   │   └── getting-started.md
│   ├── guides/
│   │   └── configuration.md
│   └── tutorials/
├── godoku.yaml              # Site configuration
```

### 2. Build the site

```bash
godoku build
```

The static site is generated in the `public/` directory.

### 3. Start the dev server

```bash
godoku serve -w
```

This starts a local server at `http://localhost:3000` with file watching — any changes to your content will trigger an automatic rebuild.

## CLI Reference

| Command | Description |
|---------|-------------|
| `godoku init [path]` | Initialize a new project |
| `godoku build` | Build the static site |
| `godoku serve` | Start the development server |
| `godoku serve -w` | Start with file watching |
| `godoku serve -p 8080` | Start on a custom port |
| `godoku version` | Show version |
