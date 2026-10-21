# Build Log

## Project

FlyRank Backend Track Capstone — Embeddable Widget & Lead-Capture Platform.

## AI assistance and development notes

AI assistance was used during implementation, debugging, testing guidance,
and documentation. The repository owner ran the commands, reviewed their
results, and performed manual dashboard checks.

### Work completed

- Implemented tenant-scoped widget and submission APIs, authentication,
  analytics, rate limiting, honeypot protection, geolocation fallback, and
  background job processing.
- Built a React and TypeScript dashboard for widgets, submissions, analytics,
  and light/dark mode.
- Added widget editing and deletion, then manually tested creating, editing,
  cancelling changes, and deleting a test widget.
- Added migration, development-launch, and seed scripts.
- Added a capstone manifest and an evidence checklist distinguishing observed
  results from tests that still need proof.

### Debugging observations

- An authentication-required message appeared in the dashboard during widget
  creation. Signing out and signing in again resolved the observed issue.
  The underlying cause was not established.
- Long terminal pastes occasionally became corrupted. Shorter, focused
  commands were used afterward, and resulting files were inspected.
- The migration helper checks whether expected tables exist. It is a local
  development convenience, not a complete version-tracked migration system.
- During startup, the API can briefly refuse connections while becoming
  healthy. The launcher suppresses readiness-probe output and reports an
  explicit error with logs if the API does not become healthy.

### Verified checks

- `go test ./...` passed for the packages shown in the captured output.
- `npm run build` completed successfully.
- `npm run lint` reported zero warnings and zero errors.
- API health, dashboard HTTP response, and dashboard-to-API proxy returned
  HTTP 200 in local checks.
- Running the seed script twice with the same demo account reused the existing
  demo widget instead of creating a duplicate.
- Widget create/edit/cancel/delete behaviour was manually checked in the
  dashboard.
- The single-script embed was manually tested from a customer page on
  port 8002 while its script and API ran on port 8080. The form rendered,
  accepted a test submission, and showed a success message. The demo
  account's Submissions page then showed the `Capstone Test` record for
  `Capstone Demo Widget`. This was a manual browser test.

### Acceptance evidence and remaining work

See `EVIDENCE.md` for the current acceptance checklist. Current evidence
includes authenticated widget CRUD, PostgreSQL tenant-isolation tests, cache
and CORS headers, malformed and oversized payload rejection, rate-limit
recovery, honeypot handling, geolocation fallback and graceful degradation,
and submission persistence after confirmation-job retry exhaustion.

Documentation and setup remain in progress until the README, manifest, and
evidence are checked against the actual startup scripts and a repeatable
setup, and the latest repository state is pushed.

The initial embed implementation used a container element plus a script
tag. During final review, the loader was extended to support a single-script
embed using the `widget_id` query parameter while retaining legacy container
support. The single-script form was manually tested from a page on port 8002:
the form rendered, accepted a test submission, and the record appeared in the
demo account's Submissions dashboard.

## Limitations of AI assistance

Suggested commands and edits were checked against actual project files and
terminal output. A suggested change was not treated as successful merely
because it had been proposed. The owner should review all final code and
documentation, run the remaining acceptance tests, and ensure that the
submitted repository contains no secrets.
