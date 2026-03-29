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

branding:
  logo_light: /static/logo_light.svg
  logo_dark: /static/logo_dark.svg
  logo_alt: My Project Logo
  logo_link: /
  favicon: /static/favicon.ico

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

footer:
  copyright: "© 2026 My Project. All rights reserved."
  position: center
  columns:
    - title: Product
      links:
        - label: Documentation
          href: /docs
        - label: API Reference
          href: /api
    - title: Community
      links:
        - label: GitHub
          href: https://github.com/org/repo
        - label: Discord
          href: https://discord.gg/invite
  social:
    - icon: github
      href: https://github.com/org/repo
    - icon: x
      href: https://x.com/handle
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
| `branding` | object | — | Logo and favicon config |
| `footer` | object | — | Configurable footer |
| `navigation` | list | — | Navbar links |
| `sections` | object | — | Content directory paths |

## Branding

Customize the navbar logo and site favicon:

```yaml
branding:
  logo_light: /static/logo_light.svg      # Image URL (replaces title text)
  logo_dark: /static/logo_dark.svg      # Image URL (replaces title text)
  logo_alt: My Project         # Logo alt text (defaults to title)
  logo_link: /                 # Where the logo links to (defaults to /)
  favicon: /static/favicon.ico # Favicon URL
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `logo` | string | — | Logo image URL. If empty, site title text is shown |
| `logo_alt` | string | site title | Alt text for the logo image |
| `logo_link` | string | `/` | URL the logo links to |
| `favicon` | string | — | Favicon URL |

:::tip {title="Using static files"}

Place logo and favicon in your `static/` folder and reference them as `/static/logo.svg`. They'll be copied to `public/static/` during build.

:::

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

## Footer

Configure a custom footer with columns, social links, and copyright:

```yaml
footer:
  position: center   # start, center (default), or end
  copyright: "© 2026 My Project. All rights reserved."
  columns:
    - title: Product
      links:
        - label: Documentation
          href: /docs
        - label: GitHub
          href: https://github.com/org/repo  # external links auto-detected
    - title: Company
      links:
        - label: About
          href: /about
        - label: Blog
          href: /blog
  social:
    - icon: github
      href: https://github.com/org/repo
    - icon: x
      href: https://x.com/handle
    - icon: linkedin
      href: https://linkedin.com/company/name
    - icon: discord
      href: https://discord.gg/invite
```

### Footer Options

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `position` | string | `center` | Alignment: `start`, `center`, `end` |
| `copyright` | string | — | Copyright notice text |
| `columns` | array | — | Link columns with title and links |
| `social` | array | — | Social media icon links |

### Supported Social Icons

`github`, `x`, `linkedin`, `discord`, `youtube`, `instagram`, `reddit`, `telegram`, `bluesky`

External links (starting with `http://` or `https://`) automatically open in a new tab with an external link icon.
