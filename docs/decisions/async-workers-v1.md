# Async module workers (v1)

## Status

Accepted — implemented in `services/control-plane`.

## Context

Module runs like `domain-intel`, `social-footprint`, and `extraction` can take
tens of seconds. Running them synchronously inside the HTTP request tied up the
server, risked write-timeout violations, and produced a poor UI experience.

## Decision

Move all module execution into an **in-process async worker queue**.

- `POST /api/leads/{id}/run` and `POST /api/pipelines/run` immediately return
  `202 Accepted` with a `run_id`.
- A fixed pool of goroutine workers (default 2, configurable via
  `CONTROL_PLANE_WORKERS`) dequeues jobs and executes modules.
- `GET /api/runs/{id}` reports `queued` → `running` → `completed` | `failed` |
  `partial`.
- The UI polls `GET /api/runs/{id}` and refreshes the lead when the run finishes.
- Risk is recomputed at the end of each job (existing `risk.Compute`).

## Consequences

- HTTP handlers no longer block on slow modules.
- The request lifecycle is simple and consistent for single and batch runs.
- Social-footprint rate limits still live inside `Runner.runModule`; the worker
  pool limits global concurrency to the configured worker count.
- No Redis / SQS / external queue is required for v1.

## Queue model

```go
type job struct {
    runID  string
    leadID string // empty for batch jobs
    single *models.RunModulesRequest
    batch  *models.PipelineRunRequest
}
```

Jobs are submitted on a buffered channel. Workers:

1. Mark the run as `running`.
2. Execute `runner.executeSingle` or `runner.executeBatch`.
3. Finalise the run with `completed`, `failed`, or `partial`.
4. Recover from panics and mark the run `failed`.

## Concurrency rules

- Only **one active run per lead**. Submitting a second run while a lead already
  has a `queued` or `running` job returns `409 Conflict` with code
  `run_in_progress`.
- Global concurrency is bounded by `CONTROL_PLANE_WORKERS`.

## Status values

| Status | Meaning |
|---|---|
| `queued` | Accepted, waiting for a worker |
| `running` | Worker is executing modules |
| `completed` | All requested modules finished without error |
| `partial` | Some leads/modules failed; others may have succeeded |
| `failed` | Runner error or panic before any useful completion |

## API contract changes

### `POST /api/leads/{id}/run`

**Request body** unchanged:

```json
{ "modules": ["email-validate"], "permission_ref": "optional" }
```

**Response `202`**:

```json
{
  "data": {
    "run_id": "run-...",
    "status": "queued"
  }
}
```

**Response `409`** — lead already has an active run:

```json
{
  "error": {
    "code": "run_in_progress",
    "message": "lead already has an active run"
  }
}
```

### `POST /api/pipelines/run`

**Response `202`**:

```json
{
  "data": {
    "run_id": "run-...",
    "status": "queued"
  }
}
```

### `GET /api/runs/{id}`

Returns a `PipelineRun` with `status`, `started_at`, `finished_at`, `lead_ids`,
`modules_executed`, `audit_event_ids`, and optional `error`.

### `GET /api/leads/{id}`

Includes `active_run_id` when a run is queued or running for this lead.

## Configuration

| Env | Default | Notes |
|---|---|---|
| `CONTROL_PLANE_WORKERS` | `2` | Number of worker goroutines. Must be `>= 1`. |
| `MODULE_TIMEOUT` | `90s` | Passed to each module. |
| `HTTP_WRITE_TIMEOUT` | `180s` | Now primarily protects slow HTTP responses, not module execution. |

## UI behaviour

- The "Run module" buttons are disabled while `active_run_id` is present or the
  mutation is pending.
- A banner shows the active run id and status (`queued`/`running`/...).
- `useRunStatus` polls `GET /api/runs/{id}` every second while the run is active.
- When the run reaches a terminal state, the hook invalidates lead, risk,
  readiness, and audit queries.

## Limitations and residuals

- **No durable queue.** Jobs live only in memory; restarting the control-plane
  loses queued jobs. Finalised runs remain in the store.
- **Single-instance workers.** The in-process queue only works inside one
  process. Scaling beyond one replica requires an external queue (Redis/SQS) in
  a later iteration.
- **Best-effort shutdown.** `Runner.Stop()` cancels the worker context and waits
  for in-flight jobs. Runs that are mid-module may be interrupted and marked
  `failed`.
- No auto-demotion of `crm_ready` leads if a re-run lowers their readiness.

## No-go for this PR

- No Redis, SQS, or Kubernetes operators.
- No module algorithm changes.
- No real CRM connector.
