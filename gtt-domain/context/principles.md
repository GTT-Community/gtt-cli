# Principles

> **Canonical reference:** https://github.com/GTT-Community/gtt-method/blob/main/GTT-CANONICAL-v2.1.md

Design principles in force. A principle earns its place only if it rules
something out.

- **The CLI is never the mind of GTT** - rules out any Go code that decides what a plan, a proposal, a
  freeze, a source authority or a validation check means. Accepts: a missing Bootstrap contract blocks
  the feature until the Bootstrap ships it, instead of being patched in the CLI.
- **Contracts, not filenames** - rules out calling a Bootstrap script by its path or parsing README text;
  only the contract entry point and declared operations are used. Accepts: one extra process hop per call.
- **Fail closed** - rules out skipping, guessing or silently falling back when compatibility, integrity
  or a capability is in doubt. Accepts: a refusal where a lenient tool would have continued.
- **Detected is not participating** - rules out installing or recording an ADE because its files or
  binary exist. Accepts: one explicit human choice during init.
- **The developer decides only what is theirs** - rules out prompts for work that follows from project
  state and the selected plan (ids, index, validation, tracking, continuing). Accepts: the CLI must be
  able to justify every automatic step from declared policy.
- **No silent destruction** - rules out overwriting a file GTT does not own or one that changed locally,
  and any delete without confirmation. Accepts: conflicts stop the flow and are reported.
- **Dependencies point inward** - rules out a use case importing an adapter, and an adapter importing
  the Cobra layer. Accepts: an interface per external system, even when there is one implementation.
- **Brief by default, detail on demand** - rules out multi-screen reports for a successful operation.
  Accepts: `--verbose` and `--json` carry what the default hides.

---
Governance: L0. Read-only for AI agents.
