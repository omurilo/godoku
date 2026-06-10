---
title: API Playground
description: Interactive playground for testing API endpoints directly from the documentation
order: 3
api:
  method: GET
  path: /users/{id}
  baseUrl: https://api.example.com
  summary: Fetch a single user by id.
  params:
    - name: id
      in: path
      required: true
      type: string
      description: The user identifier.
    - name: expand
      in: query
      type: string
  requestExample: |
    GET /users/42 HTTP/1.1
    Host: api.example.com
    Authorization: Bearer <token>
  responses:
    - status: "200"
      description: User found.
      example: |
        {
          "id": "42",
          "name": "Ada Lovelace",
          "email": "ada@example.com"
        }
    - status: "404"
      description: No user with that id.
      example: |
        { "error": "not_found" }
---

# API Playground

Every API endpoint page includes an interactive **Playground** section where users can send real HTTP requests directly from the browser.

## Features

- **Server selector** — choose from the servers defined in the OpenAPI spec
- **Parameter inputs** — auto-generated fields for path, query, and header parameters with type hints
- **Request body editor** — textarea for JSON request bodies (shown only when the endpoint accepts a body)
- **Custom headers** — add authorization tokens or custom headers
- **Send button** — fires the request via `fetch()` from the browser

## Response Display

After sending a request, the playground shows:

- **Status code** — color-coded (green for 2xx, red for 4xx/5xx)
- **Response time** — in milliseconds
- **Response headers** — all exposed headers
- **Response body** — auto-formatted JSON with pretty-printing

## CORS Considerations

Since requests are made from the browser via `fetch()`, the target API server must allow CORS from your documentation domain. If you're testing locally with `godoku serve`, the API server needs to accept requests from `http://localhost:3000`.

For APIs that don't support CORS, consider using a CORS proxy or testing with browser extensions that disable CORS restrictions during development.
