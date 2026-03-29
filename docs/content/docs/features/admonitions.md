---
title: Admonitions
description: "Callout boxes for notes, tips, warnings, and more"
order: 7
---

# Admonitions

Admonitions are callout boxes that highlight important information. Use them to draw attention to notes, tips, warnings, or critical details.

## Syntax

Wrap content with `:::type` and `:::`:

```markdown
:::info

This is an informational callout.

:::
```

You can also set a custom title:

```markdown
:::warning {title="Watch out!"}

This has a custom title instead of the default.

:::
```

## Types

### Note

:::note

This is a **note** admonition. Use it for general remarks or observations.

:::

```markdown
:::note

This is a **note** admonition.

:::
```

### Info

:::info

This is an **info** admonition. Use it for informational content.

:::

```markdown
:::info

This is an **info** admonition.

:::
```

### Tip

:::tip

This is a **tip** admonition. Use it for helpful suggestions.

:::

```markdown
:::tip

This is a **tip** admonition.

:::
```

### Warning

:::warning

This is a **warning** admonition. Use it for potential issues.

:::

```markdown
:::warning

This is a **warning** admonition.

:::
```

### Danger

:::danger

This is a **danger** admonition. Use it for critical warnings.

:::

```markdown
:::danger

This is a **danger** admonition.

:::
```

### Caution

:::caution

This is a **caution** admonition. Use it for things that need careful attention.

:::

```markdown
:::caution

This is a **caution** admonition.

:::
```

## Custom Titles

Use `{title="..."}` to set a custom title:

:::tip {title="Use sua conta do Github"}

You can write any custom title using the title attribute.

:::

```markdown
:::tip {title="Use sua conta do Github"}

You can write any custom title using the title attribute.

:::
```

## Rich Content

Admonitions support full Markdown inside:

:::info

You can use **bold**, *italic*, `code`, and even:

- Lists
- [Links](/)
- Code blocks

:::

```markdown
:::info

You can use **bold**, *italic*, `code`, and even:

- Lists
- [Links](/)
- Code blocks

:::
```
