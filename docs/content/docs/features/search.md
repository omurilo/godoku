---
title: Search
description: "Built-in full-text search powered by Fuse.js"
order: 6
---

# Search

Godoku includes a built-in client-side search that works out of the box — no configuration needed.

## How It Works

During build, Godoku generates a `search-index.json` file in `public/` containing all indexable content:

- **Docs, Guides & Tutorials** — title, description, and body text
- **API Endpoints** — method, path, summary, and description

On the client side, [Fuse.js](https://www.fusejs.io/) performs fuzzy search against this index, loaded lazily on first interaction.

## Usage

The search input appears in the **navbar** on every page. Users can:

- **Click** the search input to start typing
- **Press `/`** anywhere to focus search instantly
- **Arrow keys** (`↑` / `↓`) to navigate results
- **Enter** to open the selected result
- **Escape** to close

Results show the page title, section badge, and description.

## Search Weights

Fuse.js is configured with the following relevance weights:

| Field | Weight | Description |
|---|---|---|
| `title` | 0.4 | Page or endpoint title |
| `description` | 0.3 | Frontmatter description or API summary |
| `content` | 0.2 | Body text (first 500 characters) |
| `section` | 0.1 | Section name (Docs, Guides, API, etc.) |

## Search Index

The generated `search-index.json` contains an array of entries:

```json
[
  {
    "title": "Getting Started",
    "description": "Install and create your first site",
    "section": "Docs",
    "url": "/docs/getting-started/",
    "content": "Godoku is a fast, zero-dependency..."
  },
  {
    "title": "GET /users/{id}",
    "description": "Retrieve a user by ID",
    "section": "API",
    "url": "/api/get-users-id/",
    "content": "Returns the user object..."
  }
]
```

The index is fetched only once (on first search focus) and cached in memory for the session.
