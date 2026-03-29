---
title: Navigation & Ordering
description: Control page ordering with nav lists and frontmatter
order: 4
---

# Navigation & Ordering

Godoku provides flexible control over how pages appear in the sidebar and how they're ordered.

## Ordering Priority

Pages and groups are sorted using this priority:

1. **`nav` list** in `_index.md` — explicit ordering wins
2. **`order` frontmatter** — numeric, ascending
3. **Alphabetical** — by title

## The `nav` field

Use the `nav` field in `_index.md` to explicitly define the sidebar order:

```yaml
# content/docs/_index.md
---
nav:
  - getting-started    # page slug (filename without .md)
  - basics             # group directory name
  - advanced           # group directory name
  - configuration      # page slug
---
```

Items listed in `nav` appear first, in the specified order. Items not listed fall back to `order` then alphabetical sorting.

## Within Groups

Each group can also have its own `_index.md` with a `nav` field:

```yaml
# content/docs/basics/_index.md
---
title: Basics
nav:
  - setup
  - intro
  - faq
---
```

## Prev/Next Navigation

At the bottom of each page, Godoku renders prev/next links to adjacent pages. The order follows the same rules — `nav`, then `order`, then alphabetical. Navigation flows across groups:

```
Getting Started → Basics/Setup → Basics/Intro → Advanced/Plugins
```

## Hiding Empty Sections

If a section directory has no Markdown files, its navigation link is automatically hidden from the top navbar.
