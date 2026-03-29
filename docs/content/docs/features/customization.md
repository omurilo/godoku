---
title: Customization
description: "Customize your Godoku site with custom CSS and JavaScript"
order: 5
---

# Customization

Godoku supports custom CSS and JavaScript files to personalize your documentation site without modifying the core templates.

## Theme Toggle

Every page includes a **light/dark theme toggle** button in the navbar. The user's preference is persisted in `localStorage` and applied instantly on page load (no flash of wrong theme).

The built-in dark theme is the default. The light theme uses GitHub-style colors.

## Setup

Create a `static/` folder in your project root and add `custom.css` and/or `custom.js`:

```
my-project/
├── content/
├── static/
│   ├── custom.css
│   └── custom.js
├── apis/
└── godoku.yaml
```

When Godoku detects these files during build, it automatically includes them **after** the default styles and scripts, so your customizations take precedence.

## Changing Colors with custom.css

Godoku uses CSS custom properties (variables) for all colors. Override them in your `custom.css` to change the entire theme:

```css
/* static/custom.css */
:root {
    --bg: #1a1b26;
    --bg-secondary: #24283b;
    --bg-tertiary: #292e42;
    --text: #c0caf5;
    --text-secondary: #a9b1d6;
    --text-muted: #565f89;
    --border: #3b4261;
    --accent: #7aa2f7;
    --accent-hover: #89b4fa;
    --success: #9ece6a;
    --warning: #e0af68;
    --danger: #f7768e;
}
```

### Available Variables

| Variable | Default | Description |
|---|---|---|
| `--bg` | `#0f1117` | Main background |
| `--bg-secondary` | `#161b22` | Navbar, sidebar, cards |
| `--bg-tertiary` | `#1c2128` | Hover states, inputs |
| `--text` | `#e6edf3` | Primary text |
| `--text-secondary` | `#8b949e` | Secondary text |
| `--text-muted` | `#6e7681` | Muted/placeholder text |
| `--border` | `#30363d` | Borders |
| `--accent` | `#58a6ff` | Links, active states |
| `--accent-hover` | `#79c0ff` | Link hover |
| `--success` | `#3fb950` | Success/2xx status |
| `--warning` | `#d29922` | Warning/3xx status |
| `--danger` | `#f85149` | Error/4xx-5xx status |
| `--code-bg` | `#1c2128` | Code block background |
| `--radius` | `8px` | Border radius |
| `--sidebar-width` | `280px` | Sidebar width |
| `--navbar-height` | `60px` | Navbar height |

### Example: Light Theme

```css
/* static/custom.css - Light theme */
:root {
    --bg: #ffffff;
    --bg-secondary: #f6f8fa;
    --bg-tertiary: #eef1f5;
    --text: #1f2328;
    --text-secondary: #59636e;
    --text-muted: #8b949e;
    --border: #d1d9e0;
    --accent: #0969da;
    --accent-hover: #0550ae;
    --success: #1a7f37;
    --warning: #9a6700;
    --danger: #d1242f;
    --code-bg: #f6f8fa;
}
```

### Example: Custom Accent Color

```css
/* static/custom.css - Purple accent */
:root {
    --accent: #a78bfa;
    --accent-hover: #c4b5fd;
}
```

## Adding Behavior with custom.js

Use `custom.js` to add analytics, widgets, or any custom behavior:

```js
// static/custom.js

// Example: Add a "Back to top" button
document.addEventListener("DOMContentLoaded", function () {
    var btn = document.createElement("button");
    btn.textContent = "↑";
    btn.className = "back-to-top";
    btn.style.cssText = "position:fixed;bottom:2rem;right:2rem;padding:0.5rem 1rem;border-radius:8px;border:1px solid var(--border);background:var(--bg-secondary);color:var(--text);cursor:pointer;display:none;z-index:50;font-size:1.2rem;";

    document.body.appendChild(btn);

    window.addEventListener("scroll", function () {
        btn.style.display = window.scrollY > 300 ? "block" : "none";
    });

    btn.addEventListener("click", function () {
        window.scrollTo({ top: 0, behavior: "smooth" });
    });
});
```

## Other Static Files

Any file placed in `static/` will be copied to `public/static/` during build. This includes images, fonts, or any other assets:

```
static/
├── custom.css
├── custom.js
├── logo.png
└── fonts/
    └── my-font.woff2
```

Reference them in your custom CSS:

```css
/* static/custom.css */
.logo {
    background-image: url('/static/logo.png');
}

@font-face {
    font-family: 'MyFont';
    src: url('/static/fonts/my-font.woff2') format('woff2');
}
```
