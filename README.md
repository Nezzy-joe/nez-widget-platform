# Nez Widget Platform

Nez Widget Platform is a multi-tenant embeddable widget and lead-capture
platform built with Go and PostgreSQL.

It provides widget management, public form rendering, validated submission
collection, abuse protection, geolocation enrichment, background jobs,
and authenticated dashboard APIs.

## Current implementation

- Tenant-scoped widget management and authentication
- Public widget configuration endpoint
- Versioned embeddable JavaScript bundle
- Public submission validation and persistence
- IP- and widget-based rate limiting and honeypot protection
- Geolocation provider fallback
- Transactional background-job creation and retry processing
- Authenticated dashboard submission listing and analytics
- PostgreSQL integration tests for tenant isolation and analytics

## Technology

- Go
- PostgreSQL
- Docker
- Vanilla JavaScript for the embeddable widget
- React, TypeScript, Vite, Tailwind CSS, GSAP, and Lenis for the
  dashboard frontend.

## API endpoints

| Method | Endpoint | Purpose |
|---|---|---|
| GET | `/health` | Health check |
| POST | `/auth/register` | Register a user |
| POST | `/auth/login` | Authenticate a user |
| GET | `/auth/me` | Get the authenticated user |
| GET, POST | `/widgets` | List or create widgets |
| GET, PUT, DELETE | `/widgets/{id}` | Retrieve, update, or delete a widget |
| GET | `/public/widgets/{publicID}` | Retrieve public widget configuration |
| GET | `/widget/v1/widget.js` | Serve the embeddable JavaScript bundle |
| POST | `/submissions` | Accept a public form submission |
| GET | `/dashboard/submissions` | List tenant-scoped submissions |
| GET | `/dashboard/analytics` | Retrieve tenant-scoped analytics |

Dashboard endpoints require a valid bearer token. Tenant ownership is derived
from the verified token, not from a client-supplied tenant ID.

## Embed a widget

First, create a widget using the authenticated widget API and obtain its
public ID.

Add this HTML to the website where the form should appear. Replace
`YOUR_PUBLIC_WIDGET_ID` with the widget's actual public ID and
`YOUR_API_HOST` with the public base URL of your deployed API.

```html
<script
  src="https://YOUR_API_HOST/widget/v1/widget.js?widget_id=YOUR_PUBLIC_WIDGET_ID"
  defer
></script>
```

For local development, the script URL can use your local API, for example:

```html
<script
  src="http://127.0.0.1:8080/widget/v1/widget.js?widget_id=YOUR_PUBLIC_WIDGET_ID"
  defer
></script>
```

The embedding page must be able to reach the API URL. When embedding across
origins, public configuration and submission requests rely on the API's CORS
configuration.

The script fetches the widget configuration from
`/public/widgets/{publicID}`, renders the configured form, and submits visitor
data to `/submissions`.

The public widget ID is an identifier, not an authentication credential.
Public submissions are validated and protected by rate limiting and a
honeypot, but the public form should not be treated as a private endpoint.

### Supported field types

The API accepts text, email, textarea, telephone, number, URL, checkbox,
and select field types.

The current embeddable script renders `select` fields as text inputs because
dropdown option lists are not yet part of the widget data model.

## Development

### Prerequisites

Use a Linux development environment with Bash, Docker Engine and Docker Compose, Go, Node.js/npm, `curl`, `ss` (provided by `iproute2`), and `jq` for demo seeding.

### First-time configuration

1. Copy the environment template: `cp .env.example .env`.
2. In `.env`, replace the placeholder `POSTGRES_PASSWORD` and use the same password in `DATABASE_URL`.
3. Replace `JWT_SECRET` with a random secret at least 32 bytes long.
4. Keep real credentials in `.env` or your deployment's secret manager. Never commit `.env`.

The development launcher starts PostgreSQL, runs `scripts/migrate.sh`, builds the API, and starts the dashboard. The migration helper checks for expected tables; it does not maintain a full migration-version ledger.

Start the complete local stack from the repository root:

```bash
./scripts/dev.sh
```

With the API running, create or reuse demo data from a second terminal by running `./scripts/seed.sh`. The script prompts for credentials and avoids creating a duplicate demo widget.

Run the Go test suite:

```bash
go test ./...
```

PostgreSQL integration tests use `DATABASE_URL`. Set it in your environment
when running those tests so they execute against your test database.

Additional recommended checks:

```bash
go test -race ./internal/handlers
go vet ./...
git diff --check
```

## Project status

The Go backend now includes widget management, public submissions, background
processing, and dashboard submission and analytics APIs. The versioned
JavaScript bundle is served by the API.

The React + TypeScript dashboard is implemented with authentication,
widget management, submissions, analytics, and light/dark mode. The frontend
build and lint checks have passed in the development environment. See
EVIDENCE.md for acceptance results and documented limitations.

## Architecture

```text
Widget owner -> React/TypeScript dashboard -> Go HTTP API
                                            -> Auth and tenant identity
                                            -> Widget/submission services
                                            -> PostgreSQL repositories
                                            -> Background confirmation jobs

External website -> Versioned widget JS
                 -> Public config and submission APIs
                 -> Validation, rate limits, honeypot
                 -> Geolocation fallback -> PostgreSQL
```
