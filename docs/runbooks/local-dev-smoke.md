# Local operator smoke runbook

This is the short, current-path check for the local memory-store demo. It does not require paid API keys. See [Platform v1 status](../status/platform-v1.md) for the exact behavior of optional Python-backed modules.

## Prerequisites

- Go 1.22.5+
- Node 20+
- `npm`, `curl`, and `jq`

## Start the local demo

```bash
make demo
# UI:  http://localhost:3000/leads
# API: http://localhost:8080/healthz
# Stop: make demo-down
```

The launcher binds both services to `127.0.0.1`, uses the memory store, and refuses to replace a process already using ports 8080 or 3000. Do not run `npm run build` while the demo UI is using `.next` through `npm run dev`.

If separate terminals are preferred:

```bash
make demo-api
make demo-ui
```

## Async operator happy path

1. Open `http://localhost:3000/leads` and create a lead with an email and `permission_ref`.
2. Open the lead detail page and run **Email**.
   - The UI shows `queued` or `running`, then refreshes to `Email (ok)`.
   - Module actions re-enable when the run reaches a terminal status.
3. Create a second permissioned lead. On `/leads`, select both leads and run **Email Validation**.
   - The **Active bulk run** banner shows the short run ID and status.
   - Bulk buttons remain disabled while that run is active.
   - Open **View run** to inspect `/runs/{id}`; the banner dismisses and the leads list refreshes at terminal status.
4. Open `/runs`. While a visible run is `queued` or `running`, the list refreshes automatically. `domain-intel` is a useful slower path when network/tooling is available; otherwise Email still validates async state quickly.

`POST /api/leads/{id}/run` and `POST /api/pipelines/run` always return **202 Accepted** with a `run_id`. They do not return an updated lead body. Poll `GET /api/runs/{run_id}`, then fetch the lead result after the run is terminal.

## Local smoke gate

With the demo API running:

```bash
make smoke-api && make smoke-async && make smoke-platform
```

- `smoke-api` checks deterministic local company enrichment paths.
- `smoke-async` checks `202`, run polling, persisted result, and recomputed risk.
- `smoke-platform` checks async extraction and Email behavior. If Crawl4AI is not installed, extraction reports a structured error; that is an honest expected result for the default demo.

For complete repository checks:

```bash
make test-go
make test-ui
```

## Optional tooling

- Install Crawl4AI only when an extraction `ok` result is required:

  ```bash
  make install-extraction-venv
  EXTRACTION_CRAWL4AI_PYTHON="$PWD/modules/extraction/.venv/bin/python" make demo-api-ok
  ```

- `domain-intel` can use `theHarvester` when it is on `PATH`.
- `social-footprint` needs its Maigret Python wrapper for useful live results. Missing optional tools yield structured `unknown`, `skipped`, or `error` results rather than fabricated data.

## Troubleshooting

| Symptom | Fix |
|---|---|
| `make demo` reports a busy port | Stop the existing local service or run `make demo-down` if it was started by the demo launcher. |
| API is not ready | Inspect `${TMPDIR:-/tmp}/osint-lead-platform-demo/api.log`; `curl -sS http://localhost:8080/healthz` should return `{"status":"ok"}`. |
| UI is not ready | Inspect `${TMPDIR:-/tmp}/osint-lead-platform-demo/ui.log`; ensure no concurrent production build is using `.next`. |
| Extraction reports Crawl4AI missing | Expected without optional tooling; use the install command above only for an `ok` path. |
| A module action returns `409 run_in_progress` | Wait for the existing lead run to reach a terminal status before starting another one. |
