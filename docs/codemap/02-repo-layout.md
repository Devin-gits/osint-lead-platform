# Repo layout

```
osint-lead-platform/
├── README.md                    # high-level; may lag stage status
├── CONTRIBUTING.md              # Stage 1 research process (eval PRs)
├── LICENSE                      # MIT
├── docs/
│   ├── architecture.md          # pipeline + module contract (draft)
│   ├── compliance.md            # hard rules + risk table
│   ├── decisions/
│   │   └── stage-1-decision.md  # adopt/reject gate for modules
│   ├── research/
│   │   └── osint-tooling-research.md
│   └── codemap/                 # this map
├── evaluations/                 # Stage 1 scorecards (one file per tool)
│   └── TEMPLATE.md
├── modules/
│   ├── email-validate/          # Go 1.22.5
│   ├── domain-intel/            # Go 1.22.5, no third-party Go deps
│   ├── phone-validate/          # Go 1.22.5
│   └── social-footprint/        # Go 1.22.5 + Python wrapper
├── .github/
│   ├── PULL_REQUEST_TEMPLATE.md
│   └── ISSUE_TEMPLATE/tool-evaluation.md
└── .windsurf/workflows/         # agent workflows (local tooling)
```

## No monorepo Go workspace

There is **no** root `go.mod` or `go.work`. Each module is built/tested from its own directory:

```bash
cd modules/email-validate && go test ./...
```

## Module file maps

### email-validate

| Path | Role |
|------|------|
| `validate.go` | `Validator`, `Result`, `AuditRecord`, `Validate` |
| `cmd/email-validate/main.go` | CLI |
| `validate_test.go`, `cmd/.../main_test.go` | tests |
| `go.mod` | AfterShip/email-verifier v1.4.1 |

### domain-intel

| Path | Role |
|------|------|
| `domainintel.go` | `Analyzer.Analyze`, concurrent fan-out |
| `webcheck.go` | DNS/SSL/WHOIS reimplementation |
| `harvester.go` | theHarvester subprocess + allowlist |
| `cmd/domain-intel/main.go` | CLI |
| `Makefile` | optional helpers |

### phone-validate

| Path | Role |
|------|------|
| `phonevalidate.go` | `Validator.Validate`, merge |
| `local.go` | libphonenumber offline scanner |
| `numverify.go` | optional HTTP API |
| `cmd/phone-validate/main.go` | CLI |

### social-footprint

| Path | Role |
|------|------|
| `socialfootprint.go` | `Validator.Check`, caps, result assembly |
| `handles.go` | handle derivation from email / domain_intel |
| `maigret.go` | curated platforms, subprocess runner |
| `ratelimit.go` | per-lead min interval |
| `wrapper/maigret_check.py` | embeds Maigret library |
| `requirements.txt` | `maigret==0.6.2` |

## Dependencies summary

| Module | Go deps | External runtime |
|--------|---------|------------------|
| email-validate | AfterShip/email-verifier | network (DNS/MX) |
| domain-intel | stdlib only | network; optional `theHarvester` on PATH |
| phone-validate | nyaruka/phonenumbers | optional NUMVERIFY_API_KEY |
| social-footprint | stdlib only | Python 3.10+ + maigret; network for live checks |

## Test inventory

| Module | Test files | Notes |
|--------|------------|-------|
| email-validate | package + CLI | live DNS for real emails |
| domain-intel | package + CLI | `-short` skips live network/subprocess |
| phone-validate | package + CLI | offline local; httptest stub for numverify |
| social-footprint | package + handles + ratelimit + CLI | fake runner; no live Maigret required |

## Git / default branch

Remote default: `main`. Local history may include feature branches (`feat/*`). Module implementations landed as PRs #14–#17 after decision doc #13.
