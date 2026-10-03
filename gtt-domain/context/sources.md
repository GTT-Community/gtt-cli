# Source Manifest

> **Canonical reference:** https://github.com/GTT-Community/gtt-method/blob/main/GTT-CANONICAL-v2.1.md

The design sources this project treats as authorised. Precedence orders sources when a conflict is
interpreted (lower number wins); it never erases one. Decided by the Solution Designer on 2026-10-01:
all five documents are finished, and the four topic-specific documents prevail over the master
specification within their topic.

## Resolved by precedence

- Method Plans: `master` describes three profiles and `gtt config methodology`; `method-plans` and the
  installed Bootstrap 1.2.0 define four plans (Light, Medium, Hard, Team) and `gtt method`.
  `method-plans` prevails: four plans, `gtt method show|set|check`; `gtt config methodology` stays as an alias.

## Sources

```gtt-sources
# policy: provenance=advisory
method-plans | GTT_CLI_Method_Plans_Automation_Policy.md | 2026-10-01 | primary | 1 | active
friction | GTT_Developer_Friction_Minimization_Rule.md | 2026-10-01 | primary | 2 | active
ade-sync | GTT_CLI_Agent_ADE_Context_Discovery_Synchronization.md | 2026-10-01 | primary | 3 | active
md-tracking | GTT_CLI_MD_Initial_Tracking_261001.md | 2026-10-01 | primary | 4 | active
master | GTT_BOOTSTRAP_1.0_CLI_CONTRACT_IMPLEMENTATION_TASK.md | 2026-10-01 | primary | 5 | active
```

---
Governance: L0 once placed at `gtt-domain/context/sources.md`. Read-only for AI agents after freeze.
