---
title: OpenAPI Support
description: Auto-discover and render OpenAPI v3 specs as interactive API documentation
order: 2
---

# OpenAPI Support

Godoku auto-discovers OpenAPI v3 specification files from the `apis/` directory and generates beautiful API reference pages.

## Setup

Place your OpenAPI spec files (`.yaml`, `.yml`, or `.json`) in the `apis/` folder:

```
my-project/
├── apis/
│   ├── users-api.yaml
│   └── payments-api.yaml
├── content/
└── godoku.yaml
```

No additional configuration needed — Godoku scans `apis/` automatically.

## Single vs Multiple APIs

### Single API

With one spec file, endpoints render directly at `/api/`:

```
/api/                    → API index with all endpoints
/api/get--users/         → Individual endpoint page
```

### Multiple APIs

With multiple spec files, a **catalog page** is generated:

```
/api/                    → Catalog listing all APIs
/api/users-api/          → Users API index
/api/users-api/get--users/ → Endpoint page
/api/payments-api/       → Payments API index
```

The slug for each API is derived from the filename (e.g., `users-api.yaml` → `/api/users-api/`).

## $ref Resolution

Godoku resolves `$ref` references to `components/schemas` automatically:

```yaml
components:
  schemas:
    User:
      type: object
      properties:
        id:
          type: string
        name:
          type: string

paths:
  /users:
    get:
      responses:
        "200":
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/User'
```

The `User` schema properties will be rendered inline on the endpoint page, showing the full property table instead of just the reference name.

### Supported features

- **Nested `$ref`** — schemas referencing other schemas
- **`allOf` merging** — combined schemas are merged into one property table
- **`oneOf` / `anyOf`** — composition schemas
- **Cycle protection** — circular references are handled safely
