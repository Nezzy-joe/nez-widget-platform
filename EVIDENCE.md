# Capstone Evidence

This file maps implementation and observed behaviour to the FlyRank capstone
requirements. A feature existing in code is not, by itself, proof that every
acceptance probe passes. Pending items must be tested before final submission.

## Verified evidence

### 1. Backend test suite
Command:

    go test ./...

Observed result on 2026-10-21: all listed Go packages with tests returned
`ok`; packages without test files compiled successfully.

### 2. Frontend production build and lint
Commands:

    cd web
    npm run build
    npm run lint

Observed result: production build succeeded; lint reported 0 warnings and
0 errors.

### 3. API health and database connectivity
Command:

    curl -sS -i http://127.0.0.1:8080/health

Observed response:

    HTTP/1.1 200 OK
    Content-Type: application/json

    {"status":"ok","database":"ok"}

### 4. Dashboard and API proxy
Commands:

    curl -sS -o /dev/null -w "HTTP %{http_code}\n" http://127.0.0.1:5173/
    curl -sS -i http://127.0.0.1:5173/api/health

Observed results: dashboard returned HTTP 200; the Vite proxy returned
HTTP 200 and the same healthy API JSON.

### 5. Repeatable demo seed
Command:

    ./scripts/seed.sh

Observed on the first run: authenticated the demo account and created
`Capstone Demo Widget`.

Observed on the second run with the same account: printed
`Capstone Demo Widget already exists; no duplicate created.` The public
widget ID matched the first run. Do not include account passwords or tokens
in this file.

### 6. Dashboard widget management
Manual browser checks completed during development:
- Create a widget.
- Edit and save its title.
- Edit then cancel; the last saved title remained unchanged.
- Delete a test widget.
- Original widget data remained available during the tests.

These were manual UI checks, not automated browser tests.

### 7. Single-script cross-origin submission — verified manually
On 2026-10-21, a customer test page was served at
`http://127.0.0.1:8002/nez-widget-single-script-test.html`, using one script
tag with `widget_id` to load the API asset from port 8080. No separate widget
container div was included.

Observed browser result: the `Capstone Demo Widget` form rendered and, after
submission, displayed: `Thank you! Your submission has been received.`

Observed dashboard result: after signing into the demo account, Submissions
showed one loaded record named `Capstone Test`, associated with
`Capstone Demo Widget`. This was a manual browser test, not an automated
browser test.

## Section 6 acceptance checklist

| Requirement | Status | Remaining proof or notes |
|---|---|---|
| Authenticated widget CRUD | Partial | UI create/edit/delete tested; capture unauthenticated API rejection and API-level CRUD transcript |
| Tenant isolation for widgets and submissions | Pending | Run repository integration tests with `DATABASE_URL` set to the test database; capture test names and output |
| Per-widget embed snippet | Verified manually | Dashboard and seed now generate a single script tag with `widget_id`; the legacy container embed remains supported |
| Public config cache headers | Pending | Capture status and Cache-Control headers from `/public/widgets/{publicID}` |
| Versioned JavaScript asset | Partial | Endpoint is `/widget/v1/widget.js`; capture its response headers and body status |
| Rendering from a second origin | Verified manually | Test page on port 8002 loaded the script from port 8080; form rendered and submission succeeded |
| CORS preflight and cross-origin submission | Pending | Capture OPTIONS response and a successful cross-origin submission |
| Malformed and oversized payloads return JSON 4xx | Pending | Test both cases and capture status plus JSON response |
| Valid submission stored for correct widget/tenant | Manual smoke test done | Capture submission API response and matching authenticated dashboard query |
| Rate limit returns 429; normal traffic recovers | Pending | Run a controlled burst, record 429 response(s), then prove a normal request succeeds |
| Honeypot blocks spam | Pending | Capture a request containing the honeypot field and the expected outcome |
| Provider A fallback to provider B | Unit tests included in provider package | Capture verbose test names/output or a deterministic provider test |
| All geolocation providers unavailable, submission still succeeds | Pending dedicated proof | Force both providers to fail and capture a successful stored submission without geo |
| Confirmation side-effect failure does not block persistence | Pending dedicated proof | Force the side effect to fail and capture successful storage |
| Required documentation and setup | In progress | Complete README, this checklist, and BUILDLOG; verify clean setup |

## Shared capstone requirements

- Layered architecture: inspect and document the HTTP, service, repository, and
  model boundaries during final code review.
- Boundary validation: the full Go test suite passes; capture named malformed
  input and oversized-body test output.
- Background job and retries: background-job implementation and migrations
  exist; capture retry/failure behaviour with a focused test.
- Persistence, migrations, indexes, tenant scope: migrations define the schema
  and indexes; tenant-isolation integration proof remains outstanding.
- Idempotency: the seed command is repeatable; the jobs migration includes a
  unique constraint on `(submission_id, job_type)`.
- Secret handling: `.env` was confirmed untracked and ignored by Git. Recheck
  staged files before every commit.
- AI cost tracking: the application does not currently make AI-model calls;
  per-call AI cost tracking is therefore not applicable to runtime behaviour.

## Final evidence pass

Before submission, replace pending rows with the exact command/test name,
relevant output, or an appropriately stored screenshot/transcript. Do not
mark a row complete merely because the implementation appears to exist.
