<!--
Copy this file to evaluations/<tool-slug>.md (e.g. evaluations/holehe.md) for the tool you were assigned.
Fill in every section. Do not touch any other file in this PR.
-->

# Evaluation: {TOOL_NAME}

- **Repo:** {REPO_URL}
- **Target module:** {CATEGORY} (e.g. `email-validate`)
- **Evaluator:** {agent/human name}
- **Date:** {YYYY-MM-DD}

## 1. Summary

<!-- 3 sentences max: what it does. -->

## 2. License

- **License:** {exact license name}
- **Commercial/internal-business use allowed without restriction?** yes / no / conditional — explain
- **AGPL / "no commercial use" style clauses?** flag explicitly if present, with a link to the license file.

## 3. Maintenance health

- **Last commit:** {date}
- **Open issues:** {count}
- **Contributors:** {count} — single-maintainer risk? yes/no
- **Release cadence:** {notes}

## 4. Input / output contract

<!-- Show a REAL example (actual command + actual output), not paraphrased. -->

```
# input
...

# output
...
```

## 5. Dependencies & runtime

- **Language / runtime:** {e.g. Python 3.11}
- **Install method:** {pip / docker / binary / etc.}
- **Required API keys / accounts:** {list, or "none"}
- **Expected latency for a single lookup:** {measured or documented}

## 6. Rate limits / ToS risk

<!-- Does it hit third-party sites/APIs in a way that could violate their ToS at scale?
Cite the specific docs/README section or license clause. -->

## 7. Fit score (1-5)

**Score:** {1-5}

**Justification** (must connect to our actual pipeline stage from `README.md`'s pipeline table — not generic praise):

## 8. Recommendation

**Adopt as-is / Fork & modify / Reference only / Reject**

**Reasoning + concrete next step if adopted:**

## Sources

<!-- Link every factual claim above to the exact page/file/doc it came from. -->
