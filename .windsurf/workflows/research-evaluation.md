---
description: Produce a Stage 1 tool evaluation scorecard PR
---

# Research evaluation (Stage 1)

Use when evaluating a candidate OSINT tool. **No integration code.**

## Steps

1. Confirm the tool is not already evaluated under `evaluations/` (avoid duplicates).

2. Copy `evaluations/TEMPLATE.md` to `evaluations/<tool-slug>.md`.

3. Fill every section completely:
   - Summary (≤3 sentences)
   - License (exact; flag AGPL / non-commercial)
   - Maintenance health (last commit, issues, bus factor)
   - Input/output contract — **real** command + output, not paraphrased
   - Dependencies & runtime (language, install, API keys, latency)
   - Rate limits / ToS risk — cite specific docs
   - Fit score 1–5 vs pipeline table in `README.md` / codemap
   - Recommendation: adopt as-is / fork & modify / reference only / reject + next step
   - Sources: every factual claim linked

4. Cross-check license against the upstream `LICENSE` file.

5. Address `docs/compliance.md` if the tool touches personal data or third-party platforms.

6. **Do not modify any file outside** `evaluations/<tool-slug>.md` in the research PR.

7. PR title: `research: evaluate {TOOL}`

8. Reviewer checklist is in `CONTRIBUTING.md` and `.github/PULL_REQUEST_TEMPLATE.md`.
