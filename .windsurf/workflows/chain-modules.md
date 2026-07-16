---
description: Manually chain domain → email → phone → social modules via pipes
---

# Chain modules (manual pipeline)

There is no orchestrator yet. Compose CLIs with pipes. Order matters for social-footprint secondary handles.

## Steps

1. Build all four binaries (from each module dir):

```bash
# modules/domain-intel
go build -o domain-intel ./cmd/domain-intel
# modules/email-validate
go build -o email-validate ./cmd/email-validate
# modules/phone-validate
go build -o phone-validate ./cmd/phone-validate
# modules/social-footprint
go build -o social-footprint ./cmd/social-footprint
```

2. Put binaries on PATH or use absolute paths. For social-footprint set `SOCIAL_FOOTPRINT_WRAPPER`.

3. Start from a multi-field lead, e.g.:

```json
{
  "name": "Jane Doe",
  "email": "jane.smith@example.com",
  "phone": "+14152007986",
  "company": "Example Corp",
  "domain": "example.com"
}
```

4. Recommended chain (enrich then validate):

```bash
echo '<lead-json>' \
  | domain-intel 2>>/tmp/pipeline-audit.ndjson \
  | email-validate 2>>/tmp/pipeline-audit.ndjson \
  | phone-validate 2>>/tmp/pipeline-audit.ndjson \
  | social-footprint 2>>/tmp/pipeline-audit.ndjson \
  > /tmp/pipeline-out.json
```

5. Inspect `/tmp/pipeline-out.json` for keys:
   - `domain_intel`
   - `email_validate`
   - `phone_validate`
   - `social_footprint`

6. Inspect `/tmp/pipeline-audit.ndjson` — expect multiple audit lines (2 from domain, 1 email, 2 phone, N social).

7. Note: any module may mark sub-results `unknown` without stopping the pipe (exit 0). Only invalid JSON breaks the chain.

8. Optional: skip expensive steps (omit domain-intel or social-footprint) for faster smoke tests.
