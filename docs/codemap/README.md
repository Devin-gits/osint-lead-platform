# Codemap index

Human-readable map of the **osint-lead-platform** codebase for humans and AI agents.

**Truth hierarchy (when docs disagree):**

1. Implemented code under `modules/<name>/`
2. `docs/decisions/stage-1-decision.md`
3. `docs/architecture.md` / `docs/compliance.md`
4. Root `README.md` (often lagging — still says Stage 1 empty modules)

| Doc | Contents |
|-----|----------|
| [00-overview.md](00-overview.md) | Purpose, pipeline, stage status |
| [01-module-contract.md](01-module-contract.md) | stdin/stdout/stderr contract, lead schema |
| [02-repo-layout.md](02-repo-layout.md) | Tree, Go modules, deps, tests |
| [03-email-validate.md](03-email-validate.md) | Email validation module |
| [04-domain-intel.md](04-domain-intel.md) | Domain intel module |
| [05-phone-validate.md](05-phone-validate.md) | Phone validation module |
| [06-social-footprint.md](06-social-footprint.md) | Social footprint module |
| [07-compliance-and-license.md](07-compliance-and-license.md) | GDPR, ToS, GPL boundaries |
| [08-decisions-and-backlog.md](08-decisions-and-backlog.md) | Stage 1 decisions, next work |
| [09-agent-playbook.md](09-agent-playbook.md) | Patterns to copy when changing code |

**Workflows** (agent automation): `.windsurf/workflows/`
