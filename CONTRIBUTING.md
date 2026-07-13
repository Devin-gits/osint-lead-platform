# Contributing (Stage 1 — Research)

This repo is currently in **Stage 1: research only**. No integration code is written until a module's tool choice has an approved evaluation. This applies equally to human and AI-agent contributors.

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

### Current priority list (Stage 1)

- [ ] `web-check` — domain-intel
- [ ] `theHarvester` — domain-intel / company-enrich
- [ ] `AfterShip/email-verifier` — email-validate
- [ ] `holehe` — email-validate
- [ ] `phoneinfoga` — phone-validate
- [ ] `maigret` — social-footprint
- [ ] `spiderfoot` — orchestration
- [ ] `firecrawl` — extraction
- [ ] `crawl4ai` — extraction
- [ ] `OpenOSINT` — AI-agent orchestration layer
- [ ] `waterfall-gtm` — company-enrich architecture reference
- [ ] `h8mail` — risk/breach signal (read `docs/compliance.md` first — medium/high risk category)

Add new rows here as new candidates surface; don't duplicate an existing evaluation.

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
