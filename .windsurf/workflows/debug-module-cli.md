---
description: Smoke-test a module CLI via stdin JSON and inspect audit stderr
---

# Debug module CLI

## Steps

1. Choose module and sample lead field:

| Module | Minimal JSON |
|--------|----------------|
| email-validate | `{"email":"support@github.com"}` |
| domain-intel | `{"domain":"owasp.org"}` |
| phone-validate | `{"phone":"+14152007986"}` |
| social-footprint | `{"email":"soxoj@example.com"}` |

2. Build from the module directory:

```bash
go build -o <name> ./cmd/<name>
```

3. For social-footprint only: ensure Python deps and wrapper path:

```bash
pip install -r requirements.txt
export SOCIAL_FOOTPRINT_WRAPPER="$PWD/wrapper/maigret_check.py"
```

4. For domain-intel optional full path: ensure `theHarvester` on PATH or set `DOMAIN_INTEL_HARVESTER_BIN`.

5. Run with separated streams:

```bash
echo '<json>' | ./<name> 2> /tmp/audit.ndjson | tee /tmp/out.json
```

6. Verify:
   - stdout is valid JSON and preserves original fields
   - namespaced result key present with expected status
   - stderr audit line(s) are valid JSON with `legal_basis`
   - exit code 0 even when status is unknown/skipped

7. Negative test (must exit non-zero):

```bash
echo 'not-json' | ./<name> ; echo exit:$?
```

8. Report findings with key fields from result + audit (redact real PII in chat if needed).
