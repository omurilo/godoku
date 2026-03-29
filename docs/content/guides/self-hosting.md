---
title: Self-Hosting with Godoku
description: Use Godoku to document your own project and publish alongside your code
order: 2
---

# Self-Hosting with Godoku

You can include a Godoku site inside your project repository under a `docs/` folder. This is exactly how Godoku documents itself.

## Project Structure

```
my-project/
├── docs/
│   ├── apis/
│   │   └── my-api.yaml
│   ├── content/
│   │   ├── docs/
│   │   └── guides/
│   └── godoku.yaml
├── src/
├── go.mod
└── README.md
```

## Setup

1. Create the `docs/` directory in your repo root
2. Initialize Godoku inside it:

```bash
cd docs
godoku init .
```

3. Add your Markdown content and OpenAPI specs
4. Build and preview:

```bash
godoku build
godoku serve -w
```

## Dev Server

During development, run the dev server with watch mode from the `docs/` directory:

```bash
cd docs && godoku serve -w
```

Any changes to Markdown files, OpenAPI specs, or `godoku.yaml` will trigger an automatic rebuild.
