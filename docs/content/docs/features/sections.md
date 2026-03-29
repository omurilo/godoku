---
title: Sections & Sidebar
description: How docs, guides, and tutorials sections work with grouped sidebar navigation
order: 1
---

# Sections & Sidebar

Godoku generates pages for three built-in sections: **docs**, **guides**, and **tutorials**. Each section has its own sidebar navigation.

## Section Pages

Each section generates:

- **Index page** (`/docs/`) — lists all pages as cards
- **Individual pages** (`/docs/my-page/`) — full content with sidebar

## Sidebar Navigation

The sidebar shows all pages in the section, with support for:

- **Root pages** — listed at the top
- **Groups** — subdirectories shown as collapsible sections
- **Active state** — current page is highlighted
- **Prev/Next navigation** — bottom navigation to adjacent pages

## Empty Section Hiding

Sections with no content files are automatically hidden from the top navbar. No need to manually configure which sections to show.
