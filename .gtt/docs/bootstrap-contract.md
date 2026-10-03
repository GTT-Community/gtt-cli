# GTT Bootstrap 1.0 — the contract for the CLI

> **Canonical reference:** https://github.com/GTT-Community/gtt-method/blob/main/GTT-CANONICAL-v2.1.md

The Bootstrap owns **what GTT means and how GTT governs**. The CLI owns **how the Bootstrap is found, installed,
selected, invoked, validated, updated, exported and recovered**. The two meet only at the contracts below:
machine-readable data in `.gtt/contract/` plus a few deterministic scripts. The CLI never parses README text or
branch names, never keeps its own ADE catalog, questionnaire, exclusion list or validation semantics, and never
runs an arbitrary shell string.

```text
GTT Method -> GTT Bootstrap 1.0 -> versioned contracts / capabilities -> GTT CLI v1.0 -> ADE -> Human
```

## How a CLI uses it

```bash
bash .gtt/scripts/gtt-contract.sh release --json                       # which Bootstrap is this?
bash .gtt/scripts/gtt-contract.sh negotiate --cli-version 1.0.0 \
     --cli-capabilities contract.negotiate.v1,operation.execute.v1 --cli-schemas 1   # COMPATIBLE or REFUSE (exit 3)
bash .gtt/scripts/gtt-contract.sh capabilities --json                  # what does it offer?
bash .gtt/scripts/gtt-contract.sh operations --json                    # how do I invoke it?
bash .gtt/scripts/gtt-contract.sh run <operation> name=value ... --envelope
```

After installing into a **host** project the identity manifest still lists the catalog's artifacts that were not installed
(other overlays, documents). The CLI reconciles it once, deliberately — `run reconcile retire_missing=true` (dry run), then
`apply=true`, then `run index` — and only then `run validation.run` passes. `validation.run` reports one entry per check in
`checks` and the detail lines of failing checks in `messages`.

Exit codes of `gtt-contract.sh`: `0` ok, `1` violation, `2` usage, `3` **REFUSED** (incompatible — the CLI must not touch
the project), `4` a human decision is required (`confirmed_by_human=true` only after the human confirmed), `5` invalid
arguments or an operation this Bootstrap does not declare (nothing was run). Negotiation is fail-closed: a missing CLI
version, a missing schema declaration or a missing required capability is a refusal, and `negotiate` never modifies anything.

## Contract map

| Task section | Contract | Where | Invoked through |
|---|---|---|---|
| 2 Release identity | `bootstrap{id, version 1.2.0, schema_version, channel}`, distinct from the scaffold layout version | `.gtt/contract/release.json`; scaffold version in `.gtt/scaffold/manifest.yaml` | `release` |
| 3, 28 Compatibility | CLI min/max version, schema version, required CLI capabilities; a missing one refuses | `release.json` | `negotiate` |
| 4 Capability registry | 15 capabilities, each with the operations that realise it | `capabilities.json` | `capabilities` |
| 25, 26 Operation registry | logical op -> trusted implementation (under `.gtt/scripts/`), fixed argv, typed args, mutates?, human authority? | `operations.json` | `operations`, `run` |
| 5–7 Method Plans (profiles) | Light / Medium / Hard / Team, selected by the human and never inferred (unselected: Medium gates as a fallback, reported as not selected); operating policy, semantics, machine-enforced gates, non-relaxable invariants | `profiles.json`, state `.gtt/methodology.json` | `show profiles`, `methodology.profile.get/set` |
| 8, 9, 10 ADE registry, participation, Primary | `id, name, detect, install, validate, owned_paths, handoff, invoke, version`; detected / participating / primary / excluded; set, validate, change Primary preserving secondaries | manifest `overlays:` + `.gtt/ade.json` | `show ade-registry`, `ade.*` |
| 11, 12 Questionnaire | `scaffold.initial_design.questionnaire{template, output, version, contract_version}`; materialize | manifest `templates:` | `show initial-design`, `template.materialize` |
| 13 Guided elicitation | nine directives, each tied to the questionnaire's own section | `elicitation.json` | `show elicitation` |
| 14, 15 Initial sources / authority | selected source (path, type, version, selection metadata) is **not** a governed authority; authority and precedence are declared by a human in `gtt-domain/context/sources.md` | `.gtt/selected-sources.json` | `source.select`, `source.list` |
| 16, 24 Export / clean | GTT-owned static patterns + ADE overlays from the ownership ledger; `export --clean` (separate artifact) vs `clean` (this project) | `export-policy.json` | `export-policy`, `clean.plan` |
| 17 Recovery | snapshot schema (bootstrap identity, compatibility, ADE, Primary, profile, language, selected sources, operational state, recovery metadata) and restore rules | `recovery.json` | `recovery.snapshot`, `recovery.restore` |
| 18 Session | structured session context derived from the project (freeze, change request, proposals, validation, git, operational state, artifacts) | `gtt_project.py` | `session-context` |
| 19 Validation | one entry point; `gtt-validate.sh` and its checks stay internal | `gtt_project.py` | `validation.run` |
| 20 Status | `bootstrap, ade, methodology, sources, governance, freeze, validation, session` | `gtt_project.py` | `status` |
| 21 Freeze | the existing freeze; human authority; **no unfreeze operation exists** and the contract check fails if one is declared | `gtt-freeze.sh` | `freeze` |
| 22 GTTGuard | protection validation and registry sync | existing scripts | `guard.validate`, `guard.sync` |
| 23 Index / reconcile / query | existing scripts, exposed as stable operations (not a second retrieval implementation) | existing scripts | `index`, `reconcile`, `query`, `query.governance` |
| 29 Versioning | every consumed contract carries an integer version | `release.json` → `contracts` | `check` |
| 30 Acceptance | tests that drive the contracts as a CLI would, on disposable copies | `.gtt/tests/bootstrap-acceptance.py` | `python .gtt/tests/bootstrap-acceptance.py` |

## Method Plans (methodology profiles) — who decides what

The **human** selects the plan; the CLI records it (`methodology.profile.set profile=light|medium|hard|team
[language=en|es]`); the Bootstrap defines the meaning. *Plan* is the human-facing word and *profile* the name the 1.0
operations keep; they are the same thing. The plain-words rendering is [method-plans.md](method-plans.md). Each rule in
`profiles.json` is honest about where it is enforced:

- **Selection** (`selection.policy: ask`). The CLI asks for one of the four plans — methodological intent, never the
  derived technical policies — and never infers one. `methodology.profile.get` returns `selected: false` and
  `source: default` while nothing is selected; `profile` then names the plan whose gates apply as a fallback (`medium`),
  which a CLI must present as *not selected*, not as a choice.
- **Operating policy** (`profiles.<id>.plan.policy`: what is automatic, confirmed, proposed or left to team policy; CI,
  multi-user, traceability; vocabulary in `definitions` and `policy_values`). Declared by the Bootstrap, **executed by
  the CLI** — the Bootstrap does not implement these rules a second time, and nothing here verifies that a CLI honours
  them. Confirmation of a governed decision and of a destructive operation is `required` in every plan; the contract
  check fails otherwise.

- **Gates** (`provenance_policy`, `warnings_block_freeze`, `sources_manifest_required_for_freeze`, `open_gap_requires_affects`)
  are applied by the deterministic provenance gate. Medium equals the behaviour before profiles existed; Hard requires
  provenance for every technology row, a source manifest, `affects` on every OPEN gap, and treats warnings as failures at
  freeze.
- **Semantics** (proposal rigor, confirmations, documentation depth, ADR expectations, change process, freeze expectations,
  audit, promotion rigor, evidence) are consumed by agents through the bootstrap skill and `AGENTS.md`: instruction plane,
  not machine-enforced.
- **Invariants** that no profile weakens: human decision authority, provenance tags, evidence boundary, proposal
  distinction, conflict visibility, OPEN is not authorisation, freeze semantics, protected artifacts, selected ≠ authority.
  Only Light relaxes anything (documentation depth, ADR expectations, batching of the two confirmations, audit cadence),
  and `relaxes` states exactly how. In a frozen project selecting a **less** strict plan is refused: it is a governed
  change. The order is light < medium < hard < team.
- **Team** carries the Hard gates (the check fails if they differ) plus collaboration requirements — CI required, actor
  traceability, second-person review — which are semantics and operating policy, not gates. It is marked
  `maturity: baseline`: dedicated team plans are planned and will refine it.

## Developer experience — minimum friction, full governance

`profiles.json` → `developer_experience` declares the rule a CLI and an agent follow around operations. The rule is one;
its effect differs by plan, and that difference is never a second definition: `methodology.profile.get` returns
`interaction` (`without_asking`, `asks_for`, `id_resolution`, `maintain_reconciles_moves`), derived from the selected
plan's `plan.policy`. The rule:
resolve from governed context, plan, policies, project state and deterministic rules before asking (`decision_rule`);
what is done without a question (`automatic_operations`, each naming a declared operation without human authority);
that a protected operation is delivered as one executable script the human runs (`protected_operations`,
`agent_executes: false`); the closed list of reasons to stop (`stop_conditions`); continuity; and brief reports with
detail on demand (`report`). `methodology.profile.get` returns it with the plan. The contract check fails if a flag is
off, if an automatic operation is undeclared or needs human authority, or if `agent_executes` is not `false`.

Two operations realise it (capability `developer-experience`):

- `artifact.next-id kind=adr|epic|story [requested=ID]` — the next free id, from the identity manifest (active and
  retired: a retired id is never reused), file names under `gtt-domain/` and `gtt-domain/backlog.md`. Read-only: it
  reserves nothing. Its `resolution` is `use` under Light and Medium and `propose` under Hard and Team (the plan's
  `identity_resolution`).
- `maintain [verbose=true]` — GTTGuard registry sync, index rebuild and `gtt-validate.sh` in one run, a few lines by
  default, the full output with `verbose`. Plan-dependent step: under Light (`reference_updates: automatic`) it also
  reconciles unambiguous moves and reports them; under Medium, Hard, Team or no selection it stops with the exact
  command. An ambiguous move or a missing artifact is a decision in every plan. It never touches governed context and
  never freezes; when a human action is pending it exits 1.

## What a snapshot preserves — and what it does not

`recovery.snapshot` preserves GTT **configuration** (Bootstrap identity, ADE state, profile, language, selected sources,
freeze marker time). It does **not** preserve the governed domain content (context, ADRs, backlog): that lives in version
control and is dropped by `export --clean` on purpose. `recovery.restore` never overwrites existing state, never writes a
freeze marker, refuses a newer schema, another bootstrap id, or a relaxing profile in a frozen project, and re-installs ADE
overlays only when given the catalog (`from=`).

## Evolution without CLI changes

A new template (manifest `templates:`), new questionnaire text, new governance content, a new ADE built from the
existing ADE primitives (manifest `overlays:` + a session declaration), new profile semantics and a new operation declared
over an existing implementation all work with an unchanged CLI (`bootstrap-acceptance.py` exercises each). A change that
needs something the CLI lacks is declared in `release.json` → `requires_cli_capabilities` (or by raising the minimum CLI
version) and the CLI refuses before touching the project.

## Honest limits

- The contracts are data and deterministic scripts; whether a CLI (or an ADE) behaves accordingly is theirs to prove.
  The acceptance tests exercise the Bootstrap side only, on Windows/Git Bash with Python 3.13.
- Profile `semantics` and the elicitation directives are instructions to agents; nothing verifies an ADE follows them.
- `developer_experience` is declared policy plus two deterministic services; that an agent or a CLI asks less, stops
  only for the listed reasons and reports briefly is theirs to honour, not something the Bootstrap verifies.
- A plan's operating policy (`plan.policy`) is data for the CLI; the Bootstrap's own scripts do not change behaviour
  with it. Team's CI and actor-traceability requirements are therefore not machine-enforced by the Bootstrap.
- `contract.negotiate.v1` and `operation.execute.v1` are the CLI capability ids this Bootstrap requires; they name the
  behaviours defined here, not a published CLI version.
