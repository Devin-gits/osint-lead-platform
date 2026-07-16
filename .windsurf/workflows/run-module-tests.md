---
description: Run tests for an OSINT lead platform module
---

# Run module tests

## Steps

1. Identify the module directory under `modules/` (email-validate, domain-intel, phone-validate, social-footprint).

2. Run the full suite from that directory (not repo root — no root go.mod):

// turbo
3. Execute:

```bash
go test ./...
```

with `cwd` set to `modules/<name>`.

4. Optional short mode (domain-intel skips live network/subprocess):

```bash
go test -short ./...
```

5. Interpret expected network needs:

| Module | Network | Extra runtime |
|--------|---------|---------------|
| email-validate | Yes (DNS/MX for real-email tests) | none |
| domain-intel | Yes for full suite | optional theHarvester |
| phone-validate | No for default suite | none (httptest stubs) |
| social-footprint | No for default suite | none (fake runner) |

6. If tests fail, fix root cause in the module; do not skip or weaken tests without explicit user direction.

7. Report: pass/fail counts, any skipped short-mode tests, and commands re-run.
