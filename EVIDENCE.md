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


### 8. Malformed and oversized submission payloads — verified locally

Malformed JSON probe:

    curl -sS -i -X POST http://127.0.0.1:8080/submissions -H 'Content-Type: application/json' --data-binary '{"public_id":'

Observed result: HTTP 400 Bad Request with
`{"error":"invalid JSON request"}`.

Oversized probe: `/tmp/nez-oversized-submission.json`, 20,078 bytes,
containing a valid public ID and a 20,000-character message.

Observed result: HTTP 413 Request Entity Too Large with
`{"error":"request body too large"}`.

Both responses included CORS headers and `Cache-Control: no-store`.
These were live local API probes.

## Section 6 acceptance checklist

| Requirement | Status | Remaining proof or notes |
|---|---|---|
| Authenticated widget CRUD | Verified manually | Unauthenticated GET /widgets returned HTTP 401. Authenticated CRUD: POST returned 201; GET returned 200; PUT returned 200 and a fresh GET confirmed the updated title; DELETE returned 204 and a subsequent GET returned 404. Temporary test widget deleted. |
| Tenant isolation for widgets and submissions | Verified | PostgreSQL integration tests passed: `TestWidgetRepositoryTenantIsolation` rejected cross-tenant get, update and delete; `TestSubmissionRepositoryListByTenantIsolationAndPagination` verified tenant isolation; `TestSubmissionAnalyticsTenantIsolationAndWindows` verified analytics tenant isolation. |
| Per-widget embed snippet | Verified manually | Dashboard and seed now generate a single script tag with `widget_id`; the legacy container embed remains supported |
| Public config cache headers | Verified locally | GET returned HTTP 200, `Cache-Control: public, max-age=60`, and `Access-Control-Allow-Origin: *` |
| Versioned JavaScript asset | Verified locally | HEAD returned HTTP 200, `Cache-Control: public, max-age=300`, JavaScript content type, and `X-Content-Type-Options: nosniff` |
| Rendering from a second origin | Verified manually | Test page on port 8002 loaded the script from port 8080; form rendered and submission succeeded |
| CORS preflight and cross-origin submission | Verified manually | OPTIONS returned HTTP 204 with allowed origin `*` and methods `POST, OPTIONS`; the cross-origin submission was accepted and appeared in the dashboard |
| Malformed and oversized payloads return JSON 4xx | Verified locally | Malformed JSON: HTTP 400, `{"error":"invalid JSON request"}`. 20,078-byte request: HTTP 413, `{"error":"request body too large"}` |
| Valid submission stored for correct widget/tenant | Verified manually | `Capstone Test` appeared under `Capstone Demo Widget` in the owning demo account's Submissions page |
| Rate limit returns 429; normal traffic recovers | Verified | 21-request burst: requests 1–20 returned HTTP 201; request 21 returned HTTP 429. After 65 seconds, a legitimate submission returned HTTP 201. |
| Honeypot blocks spam | Verified | Twenty requests with a non-empty company_website returned HTTP 201, followed by HTTP 429 when the rate limit was reached. Dashboard verification showed only the two legitimate submissions, Rate Limit Recovery and Capstone Test; no Bot probe records appeared. |
| Provider A fallback to provider B | Verified | `TestGeoLocatorFallsBackWhenPrimaryFails` passed; `TestGeoLocatorReturnsErrorWhenAllProvidersFail` passed. Verbose output captured from deterministic provider unit tests. |
| All geolocation providers unavailable, submission still succeeds | Verified | `TestSubmissionServiceStoresSubmissionWhenGeolocationFails` passed. Log: geolocation unavailable; continuing without location data. The service test confirms submission storage despite geolocation failure. |
| Confirmation side-effect failure does not block persistence | Verified | `TestWorkerSchedulesRetryOnFailure` passed. `TestJobRepositoryLifecycle` passed against PostgreSQL and confirmed the submission remained stored after the confirmation job exhausted its retries and reached failed status. |
| Required documentation and setup | Verified locally | README, capstone.yaml, BUILDLOG, and EVIDENCE document setup and use. Shell syntax checks passed; migrations 001-003 were already applied; API health reported database ok; dashboard returned HTTP 200. Verified in the existing local workspace; a fresh clone has not been independently tested. |

## Shared capstone requirements

- Layered architecture: README documents the HTTP API, authentication and tenant identity, services, PostgreSQL repositories, and background confirmation jobs.
- Boundary validation: `go test ./...` passed. Section 8 records live malformed-JSON HTTP 400 and oversized-body HTTP 413 probes.
- Background jobs and retries: `TestWorkerSchedulesRetryOnFailure` and PostgreSQL `TestJobRepositoryLifecycle` passed. The integration test verifies submission persistence after retry exhaustion.
- Persistence, migrations, indexes, and tenant scope: PostgreSQL integration tests passed for widget isolation, submission isolation and pagination, and analytics isolation. The migration helper is documented as a local development convenience, not a full migration ledger.
- Idempotency: the seed command is repeatable; the jobs migration includes a unique constraint on `(submission_id, job_type)`.
- Secret handling: `.env` is ignored and untracked. Check staged files before every commit.
- AI cost tracking: the application does not currently make AI-model calls, so per-call AI cost tracking is not applicable to runtime behaviour.

## Final evidence pass

Before submission, replace pending rows with the exact command/test name,
relevant output, or an appropriately stored screenshot/transcript. Do not
mark a row complete merely because the implementation appears to exist.
