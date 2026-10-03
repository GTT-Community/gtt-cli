# Glossary

> **Canonical reference:** https://github.com/GTT-Community/gtt-method/blob/main/GTT-CANONICAL-v2.1.md

Domain terms whose meaning in this project differs from the everyday meaning, or
that are ambiguous across teams.

| Term | Meaning in this project | Not to be confused with |
|---|---|---|
| Bootstrap | The versioned GTT implementation and contract provider the CLI operates | A startup script, or the act of initialising |
| Catalog | A Bootstrap package as distributed: core plus every ADE overlay | An installed project, which holds only the chosen overlays |
| Contract | Machine-readable data under `.gtt/contract/` that the CLI consumes | README text or script filenames |
| Capability | A versioned thing the Bootstrap offers, or requires from the CLI, negotiated before any change | A feature flag |
| Operation | A logical action the Bootstrap declares, with typed arguments and a trusted implementation | A shell command |
| ADE | An AI development environment (Claude Code, Kiro, GitHub Copilot, Codex) | The underlying model or its API |
| Candidate | An ADE whose signals were detected | A participating ADE |
| Participating ADE | An ADE the human chose; it receives its overlay | A detected one |
| Primary ADE | The one participating ADE that is the main workflow environment; no authority | An approver or arbiter |
| Overlay | The files GTT installs for one ADE, recorded in the ownership ledger | The ADE's own configuration in the same directory |
| Method Plan | Light, Medium, Hard or Team: how much operational work the human delegates; called profile in the 1.0 operations | A quality or maturity level |
| Operating policy | The plan's declared automation and confirmation values, executed by the CLI | The plan's semantics and gates, which belong to the Bootstrap |
| Initial MD / directive | A Markdown source at the project root that the CLI reads, applies through the Bootstrap source contract and validates | Governed context |
| Selected source | A source the human picked as initial material | A governed authority |
| Snapshot | A recovery record of GTT configuration | A backup of the application or of the governed domain |
| `clean` | Removes GTT from the current project | `export --clean`, which writes a separate delivery artifact and leaves the project intact |
| Handoff | The Bootstrap-defined instruction entry and context the CLI delivers to the Primary ADE | A prompt written by the CLI |

---
Governance: L0. Read-only for AI agents.
