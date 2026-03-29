---
title: Deploying to GitHub Pages
description: How to deploy your Godoku site to GitHub Pages with GitHub Actions
order: 1
---

# Deploying to GitHub Pages

Godoku generates a fully static site in the `public/` directory, which can be deployed anywhere. Here's how to deploy to GitHub Pages.

## GitHub Actions Workflow

Create `.github/workflows/docs.yml` in your repository:

```yaml
name: Deploy Docs

on:
  push:
    branches: [main]
  workflow_dispatch:

permissions:
  contents: read
  pages: write
  id-token: write

concurrency:
  group: pages
  cancel-in-progress: false

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'

      - name: Install Godoku
        run: go install github.com/omurilo/godoku/cmd/godoku@latest

      - name: Build site
        run: cd docs && godoku build

      - uses: actions/upload-pages-artifact@v3
        with:
          path: docs/public

  deploy:
    needs: build
    runs-on: ubuntu-latest
    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}
    steps:
      - id: deployment
        uses: actions/deploy-pages@v4
```

## Setup

1. Go to your repository **Settings** → **Pages**
2. Under **Source**, select **GitHub Actions**
3. Push the workflow file to the `main` branch

The site will be available at `https://<username>.github.io/<repo>/`.

## Custom Domain

To use a custom domain, add a `CNAME` file to your `docs/public/` output (or configure it in GitHub Pages settings), and update `url` in `godoku.yaml` accordingly.
