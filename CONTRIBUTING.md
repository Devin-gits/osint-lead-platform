# Contributing (Stage 2 — Build modules)

The repo has moved from Stage 1 (research) into **Stage 2: implement approved modules**. Historical evaluations live in `evaluations/`; adoption decisions are in `docs/decisions/stage-1-decision.md`. New work targets `modules/<name>/` for modules that the decision doc has already cleared, and must follow the module contract in `docs/architecture.md` / `docs/codemap/01-module-contract.md`.

## Scoping a research task

Each task = one tool, one deliverable file: `evaluations/<tool-slug>.md`, copied from `evaluations/TEMPLATE.md` and filled in completely. Do not touch any other file.

### Prompt to hand an AI agent

> You are a research contributor on the `osint-lead-platform` repo, Stage 1 (research only — do not write integration code). Your task: produce `evaluations/{tool-slug}.md` evaluating **{TOOL}** ({REPO_URL}) for the "{CATEGORY}" module of a lead-enrichment/validation platform.
>
> Copy `evaluations/TEMPLATE.md` to `evaluations/{tool-slug}.md` and fill in every section:
> 1. Summary (3 sentences).
> 2. License — exact license, and whether it permits commercial/internal-business use without restriction (flag AGPL/"no commercial use" clauses explicitly).
> 3. Maintenance health — last commit date, open issue count, single-maintainer risk.
> 4. Input/output contract — a real example, not paraphrased.
> 5. Dependencies & runtime — language, install method, required API keys, expected latency.
> 6. Rate limits / ToS risk — cite the specific doc/README/license section.
> 7. Fit score (1-5) for our pipeline (see README.md's pipeline table) + justification.
> 8. Recommendation: adopt as-is / fork & modify / reference only / reject, with reasoning.
>
> Cite every claim with a link. Do not modify anything outside `evaluations/{tool-slug}.md`. Open a PR titled `research: evaluate {TOOL}` against `main`.

### Stage 2 implementation backlog

Only modules explicitly moved to Stage 2 in `docs/decisions/stage-1-decision.md` may have a `modules/<name>/` branch open:

- [x] `modules/email-validate` — AfterShip email-verifier
- [x] `modules/domain-intel` — web-check-lite + theHarvester (subprocess)
- [x] `modules/phone-validate` — nyaruka/phonenumbers + optional numverify
- [x] `modules/social-footprint` — Maigret (Python library via wrapper)
- [ ] `modules/extraction` — Crawl4AI primary, Firecrawl adapter optional
- [ ] `modules/company-enrich` — blocked pending evaluation of `local-enrichment-tool`
- [ ] lightweight pipeline runner / orchestration
- [ ] storage, retention, and CRM-ready scoring

Research-only exceptions: if a new tool is needed for an approved module, open a research PR under `evaluations/<tool-slug>.md` using `evaluations/TEMPLATE.md`.

## PR review process

Every PR is reviewed against this checklist before merge:

1. **Scope discipline** — touches only the assigned evaluation file.
2. **Sourcing** — every factual claim links to the actual page/doc it came from.
3. **License accuracy** — cross-checked against the repo's real `LICENSE` file.
4. **Compliance flag** — addressed per `docs/compliance.md` for anything touching personal data or third-party platforms.
5. **Fit score justification** — tied to the actual pipeline, not generic praise.
6. **Actionable recommendation** — unambiguous, with a concrete next step if adopted.

Reviewer will aggregate merged evaluations into a decision doc before any Stage 2 (build) branch opens.

## Moving to Stage 2

A module only advances to `modules/<name>/` once:
- Its evaluation(s) are merged
- The reviewer has published a decision (adopt/fork/reject) in an aggregated decision doc
- `docs/architecture.md`'s module contract is confirmed for that module
