---
name: gtt-bootstrap
description: Detect candidate ADEs, install the GTT overlays of the ADEs the human chooses to participate (exactly one Primary), then populate gtt-domain/context/ for the first time in a new project. Use when the user says to set up GTT, bootstrap GTT, initialize GTT, or has just cloned GTT Bootstrap into a project and gtt-domain/context/ still holds template placeholders. Checks the project root for an existing solution document (offering the Bootstrap's Initial Design Questionnaire when there is none), confirms with the Solution Designer that it is finished rather than a draft, asks for whatever it does not answer, and drafts the six context files plus a permanent SOURCE-BRIEF at the project root for review before anything is written.
---

> **Canonical reference:** https://github.com/GTT-Community/gtt-method/blob/main/GTT-CANONICAL-v2.1.md

# Bootstrap the governed context

`gtt-domain/context/` ships as a template — angle-bracket placeholders and empty
table rows, not real answers. This skill turns those placeholders into the
project's actual context, confirmed by the Solution Designer before anything
is written.

## 0. Detect candidate ADEs, let the human choose, and resolve the overlays

Do this before touching the filesystem, and before step 1. Three separate
questions: which ADEs look present (detection), which ones the human wants GTT
to govern (participation), and which one is the Primary.

GTT Bootstrap ships a portable core — `AGENTS.md` and `SOURCE-BRIEF.*` (if a
source document exists) at the project root, plus the governed domain
(`gtt-domain/`: `context/`, `adr/`, `proposals/`, `backlog.md`, `change-request.md`) and
the Engine (`.gtt/`, including its documentation in `.gtt/docs/`, defined by
`.gtt/scaffold/manifest.yaml`) — plus one adapter per
supported ADE. The source repository carries every adapter — it is a
catalog, not a package to install whole. A target project receives the
portable core plus the overlay of every ADE the human chose to have
participate, never the whole catalog. Workspace hygiene: the Engine and its
documentation stay in `.gtt/`, and the governed domain in `gtt-domain/`;
never mix them, and report (never merge) a collision with a `.gtt/` or
`gtt-domain/` the project already has.

What each ADE's adapter contains (a project with several participating ADEs
has the union of their rows; the registry itself is the `overlays:` section of
the manifest, read through `gtt-ade.sh list`):

| Host ADE | Adapter | `.claude/` | `.kiro/` | `AGENTS.md` | `.copilot/copilot-instructions.md` |
|---|---|---|---|---|---|
| Claude Code | Claude | YES | NO | YES | NO |
| Kiro | Kiro | NO | YES | YES | NO |
| Codex | Portable/AGENTS | NO | NO | YES | NO |
| GitHub Copilot | Copilot | NO | NO | YES | YES |
| Other supported ADE | Explicit adapter only | only if mapped | only if mapped | per support | per support |
| Unknown ADE | Portable/unknown | NO | NO | do not guess | NO |

**Detection is not participation.** From the project root, run
`bash <catalog>/.gtt/scripts/gtt-ade.sh detect --from <catalog>`. It lists every ADE in the
registry with the signals it found (an overlay path, an ADE binary on the PATH). Those
are *candidates* — nothing more: not installed, not authorized, not governed, not
participating. Never detect from the underlying model: a Claude model is not Claude Code,
a GPT model is not Codex, and an API alone is neither. The ADE executing this bootstrap
is self-evident to that ADE and is a sensible suggestion — still only a suggestion.

**Ask, do not assume.** Show the candidates (and the registry ADEs that were not
detected: the human may want one that is not installed yet) in this shape:

> Detected these possible ADEs.
>
> - Claude Code (signals: `.claude/`, binary `claude`)
> - Kiro (signals: `.kiro/`)
>
> Which of them should participate in this project? GTT installs and governs each
> one's integration. And which one is the Primary ADE — the principal environment of
> your workflow? (The Primary ADE has no extra authority.)

Silence is not an answer. Exactly one Primary, and it must be one of the participating
ADEs. If the human names no ADE, or the only ADE has no row above and none has been
explicitly mapped for it, install the portable core only, do not invent an adapter,
and report plainly that no native adapter exists (it falls back to `AGENTS.md` where it
can read that natively).

**The Primary ADE holds no authority.** It identifies the principal environment of the
workflow and nothing else. It does not outrank, approve or arbitrate for another ADE, and
no ADE — the Primary included — ratifies anything: every ADE prepares, the human decides.
Do not describe or behave as if there were an ADE hierarchy. `AGENTS.md`, `.claude/`,
`.kiro/`, `.copilot/`, `.codex/` and any ADE's memory or session history are integration
surfaces, never governance authority. Several ADEs may run at once (three terminals in one
editor), which is why every participating ADE gets its overlay.

**Install.** Once the portable core and the Engine are in place, with the choice made,
dry-run first, show the plan (a CONFLICT means an overlay file already exists — it is never
overwritten or merged; stop and report it), then apply:

```bash
bash <catalog>/.gtt/scripts/gtt-ade.sh install --from <catalog> --participating claude,codex --primary claude
bash <catalog>/.gtt/scripts/gtt-ade.sh install --from <catalog> --participating claude,codex --primary claude --apply
```

That copies only the participating overlays, writes `.gtt/ade.json` (participating, Primary,
the ADEs declined as excluded, and a ledger of the files it installed so a later clean removes
exactly those and never the host project's own ADE configuration), and validates the result,
rolling everything back on failure. Do not copy overlay files by hand and never hand-edit
`.gtt/ade.json`. Only Claude Code has a real-time write block; say so for every other
participating ADE (governed by instructions plus the CI gate, not hard-blocked).

**Re-running bootstrap.** Read the state first (`gtt-ade.sh state`). If `.gtt/ade.json`
exists: never drop a participating ADE, never re-add one recorded as excluded unless the
human says so (`--reinclude <ade>`), never change the Primary unless the human says so
(`gtt-ade.sh set-primary`). A project that predates `.gtt/ade.json` (an overlay present, no
state) is migrated with `gtt-ade.sh adopt` (dry run first): it records the existing ADE as
both Primary and participating only when exactly one overlay is present and intact; where
several are present, ask the human and pass `--participating a,b --primary a`. Never infer
an authorization the state does not show.

Once resolved, note the detected ADEs, the participating ADEs, the Primary, the files this
will install, and the ADEs this will exclude — this becomes part of the report after
step 6. Then continue to step 1.

### Method Plan

Before step 1, establish the project's Method Plan. It is the human's choice.

1. Run `bash .gtt/scripts/gtt-project.sh profile get`. If it names a selected plan, state
   it and do not ask again. If it says `plan: not selected`, ask.
2. Show the four plans - Light, Medium, Hard, Team - each with the `label` and `summary`
   it carries in `.gtt/contract/profiles.json` (`bash .gtt/scripts/gtt-contract.sh show
   profiles`), in that order, as written. Do not paraphrase them into "basic" or
   "advanced": a plan is an operating profile, not a quality level.
3. Ask for that one choice only. Never ask the human to configure the derived policies
   (index, identity, validation, ADE sync) one by one, and never pick, default or
   recommend a plan on your own. If the human asks what the difference is, answer from
   `.gtt/docs/method-plans.md` - what each plan does without asking, what it asks, what
   it requires - including exactly what Light relaxes.
4. Record the answer: `bash .gtt/scripts/gtt-project.sh profile set --profile <id>` (dry
   run), then the same with `--apply`. If the human declines to choose now, leave it
   unselected, say that the Medium gates apply meanwhile as a fallback, and list the
   selection under *Human action required* in the report.
5. For the rest of the bootstrap, apply the selected plan's `semantics` (for example,
   Light may ask Confirmation A and B in one exchange; each still needs its own answer).
   No plan lets you skip a confirmation, write governed context unconfirmed, or freeze.

## 1. Establish the regime

Two independent questions. Cross them before doing anything.

| `gtt-domain/context/` has real content? | `gtt-domain/.frozen` exists? | Action |
|---|---|---|
| No | No | Normal case. Proceed to step 2. |
| No | Yes | Anomaly — freeze validates content before writing the marker, so this should be impossible. Stop. Report the inconsistency. Do not guess which side is right. |
| Yes | Yes | Normal governed state. Bootstrap does not apply. Offer `gtt-audit` instead. |
| Yes | No | **Migration case.** Say exactly: "gtt-domain/context/ is populated but gtt-domain/.frozen is absent. If this project was already governed before this version of GTT, run .gtt/scripts/gtt-freeze.sh now rather than treating this as a fresh bootstrap." Stop. Do not offer to overwrite it. |

"Real content" means not placeholders like `<layered / hexagonal / ...>` or
empty table rows.

**Earlier layouts.** Before proceeding, also check for artifacts left by an
earlier GTT layout:

- v2 layout: `INDEX.md`, `CHANGE-REQUEST.md`, `GTT-COMPLETION.md`, or
  `BACKLOG.md` sitting at the project root under those upper-case names.
- v2.1 layout before the scaffold restructure: governance nested under `gtt/` —
  `gtt/context/`, `gtt/adr/`, `gtt/proposals/`, `gtt/backlog.md`,
  `gtt/CHANGE-REQUEST.md`, `gtt/SESSION.md`, `gtt/.frozen`, or documentation
  directly under `gtt/` (`gtt/INDEX.md`, `gtt/INSTALLATION*.md`, ...).
- v2.1 layout after the scaffold restructure, before the move to `.gtt/` and `gtt-domain/`: the Engine in `gtt/` and governance and documentation at the project root (`context/`, `adr/`, `proposals/`, `docs/`, `backlog.md`, `change-request.md`, `session.md`, `.frozen`).

If any are found:

- Do not silently create new copies at the current scaffold locations
  alongside them — that produces two canonical artifacts with the same
  meaning, which is worse than either problem alone.
- Do not silently move or delete the old files either.
- Stop and tell the Solution Designer plainly what was found and where, and
  ask whether to migrate them or leave the project on its current layout for
  now. Only move on explicit confirmation.
- If the project is frozen, relocating `gtt-domain/context/` or `gtt-domain/adr/` is a governed
  change: route it through `gtt-propose-change` and a promotion script, never
  a direct move.
- If they confirm an unfrozen migration, move each file's content verbatim (no
  rewriting) to its current scaffold path, remove the old copy, and report
  exactly what moved.

## 2. Look for an existing solution document

Look at the project root only — not subdirectories, not the rest of the repo.
Anything there that isn't part of the kit itself (`AGENTS.md`, `SOURCE-BRIEF.*`,
`readme-gtt.md`, `readme-gtt.es.md`, `.claude/`, `.kiro/`, `copilot/`, `.gtt/`,
`gtt-domain/`)
and isn't ordinary project scaffolding (`package.json`, `.gitignore`, a
pre-existing `README.md`, `LICENSE`, and the like) is a candidate solution
document. The
Solution Designer does not have to name it anything in particular or tell the
agent it exists — a `.md`, `.txt`, Word, or PDF file sitting there is enough.

- **Exactly one candidate:** confirm it in one line — "Using `<name>` as the
  source document?" — rather than assuming silently, then use it.
- **No candidate at the root:** ask directly whether a document exists
  elsewhere — another path, an external doc, or paste it into chat. If there is
  none at all, offer the Initial Design Questionnaire (below).
- **More than one candidate:** ask which one. Do not guess between them. If several sources are
  legitimate, declare them in `gtt-domain/context/sources.md` (template: `gtt-template.sh materialize
  source-manifest`) with an authority and an unambiguous precedence — never by reading order — and
  cite them with `[FUENTE: id]`.
- **A `SOURCE-BRIEF.*` already at the root:** this project was already
  bootstrapped. Stop and offer the `gtt-audit` skill instead (same as step 1).

Do not scan subdirectories or the rest of the repository speculatively looking
for "the" document — the root check above is the only place this skill looks
without being told, same as Claude Code checking a fixed location for
`AGENTS.md` instead of searching for it. Beyond that, ask.

Wherever it comes from, once it is processed a copy becomes `SOURCE-BRIEF.*` at
the project root (step 6) — permanent, not archived away into `.gtt/docs/` —
so the reasoning behind the context stays visible and traceable right at the
project root, the one GTT artifact a human should never have to go looking
for under `.gtt/`.

### The Initial Design Questionnaire (no sufficient design document)

If the human has no design document, or only one too thin to be a source, do not
improvise an interview: offer the Bootstrap's Initial Design Questionnaire. It is owned
by the Bootstrap (`.gtt/scaffold/templates/gtt-initial-design-questionnaire.md`, declared
under `templates:` in the manifest); there is exactly one, and you never copy or
paraphrase it anywhere else.

1. Materialize the working copy: `bash .gtt/scripts/gtt-template.sh materialize initial-design-questionnaire`
   (dry run), then `--apply`. It creates `gtt-domain/proposals/bootstrap/initial-design-questionnaire.md`,
   never overwrites an existing one and registers it in the index. To resume an earlier
   session, open the existing copy instead.
2. Read the questionnaire's *Operating Contract* (section 1) and follow it: inspect the
   project before asking, ask progressively — never the whole form at once — and populate
   the document as the conversation goes.
3. Keep provenance honest: `[FUENTE: archivo:línea]` for what the project shows, `[VACÍO]`
   for what is not established, `[CONFLICTO]` when sources or answers disagree,
   `[PROPUESTA]` for an option you suggest. A proposal is never a decision, an ADE
   inference is never a human decision, and you never fill `Human decision` for the human.
4. Stop when the human has reviewed the summary (section 24) and Readiness (section 23) is
   `READY` or `READY_WITH_OPEN_ITEMS`, or when they decide to continue later. `NOT_READY`
   is the same as an unfinished draft (step 3): stop.
5. From then on the questionnaire is the source document: step 3's confirmation is about
   it; step 4 maps it to the six files — a `[VACÍO]` becomes an empty cell, an unresolved
   `[CONFLICTO]` is asked again, and a `[PROPUESTA]` becomes context only if a
   `Human decision` records it; step 6 preserves the completed questionnaire verbatim as
   `SOURCE-BRIEF.md`. The working copy under `gtt-domain/proposals/bootstrap/` is removed
   by `gtt-freeze.sh` together with the other bootstrap drafts.

The questionnaire never becomes governed architecture by being filled in.

## 3. Confirm the document is finished — or stop

If a document exists, do not start mapping it yet. Ask the Solution Designer
directly: is this finished — the real decisions, not a draft you're still
thinking through?

- **Confirmed finished:** move to step 4.
- **Still a draft, unsure, or "sort of":** stop here. Say plainly that
  `gtt-bootstrap` needs a finished document to work from, and that patching a
  draft with interview questions is not the same thing as the Solution
  Designer actually deciding it. Suggest reviewing it with an LLM for
  inconsistencies first if they haven't already. Do not proceed to step 4.
  They come back and run this skill again once it's ready.
- **No document exists at all:** run the Initial Design Questionnaire (above);
  its review, with Readiness `READY` or `READY_WITH_OPEN_ITEMS`, is this step's
  confirmation, so continue to step 4 once that holds. The interview itself is how
  the design gets defined this time.

This is a different confirmation from step 5. This one is about whether the
Solution Designer's *own* thinking is settled. Step 5 is about whether the
*derived* context files accurately capture it. Conflating them lets an
unfinished design slip through disguised as a completed bootstrap.

## 4. Ask what is still missing

Read the confirmed document and map its content onto the six files below. For
anything it does not answer, add it to the question list — do not infer or
invent an answer from adjacent context.

Group questions by file, not by field — six short rounds, not forty
one-line questions. Only ask about what the document actually left open.

| File | Ask for |
|---|---|
| `solution-vision.md` | The problem, who it's for, what success looks like, what it deliberately will not become |
| `architecture.md` | Architectural style, modules and boundaries, integration strategy, data ownership, deployment topology, known deviations |
| `stack.md` | Language, runtime, framework, compute model, datastores, messaging, identity, secrets, IaC, CI/CD, observability, testing |
| `constraints.md` | Cloud provider, compute model, IaC, runtime/language version, datastore, comms style, data residency, compliance, budget ceilings, explicit out-of-scope |
| `principles.md` | Design principles that would actually cause a PR to be rejected, and the trade-off each one accepts |
| `glossary.md` | Domain terms whose meaning here differs from the everyday meaning |

The fewer answers exist going in, the more of this step runs. That is
expected, not a failure state — a project with a thin source document just
needs more of the conversation to happen here instead.

If an answer is genuinely not decided yet, leave it empty rather than filling
it with a plausible guess — say so explicitly. An empty cell is a decision not
yet made; a guessed one is architecture invented by the agent, which is the
exact failure GTT exists to prevent.

Classify every empty decision as a gap in the `gtt-gaps` block of `stack.md` (section 8), tagged
`[VACÍO: GAP-001]` where it appears: **BLOCKING** if the design cannot be frozen without it,
**OPEN** if the current design stands without it (an OPEN carries an explicit `scope:` and is never
an authorisation). Where sources disagree write `[CONFLICTO: a vs b]`; never pick a winner silently.
A `[PROPUESTA]` never goes into the six files. `.gtt/scripts/gtt-check-provenance.sh --pre-freeze`
tells the Solution Designer what still blocks the freeze.

## 5. Confirm before drafting

Summarize what will go into each of the six files — not the full file text,
enough to review in one pass — and get explicit confirmation from the
Solution Designer before writing anything. Silence is not confirmation.

## 6. Write the context, then stop

The project is pre-freeze, so these paths are writable. Write directly the six
files under `gtt-domain/context/`, using the exact target filenames (`stack.md`,
`architecture.md`, `constraints.md`, `principles.md`, `solution-vision.md`,
`glossary.md`).

If a source document existed (step 2), also write it at the project root as
`SOURCE-BRIEF`, keeping the original file extension, unmodified and not
paraphrased.

The accepted-decisions folder is writable in this same window: if
`ADR-001-context-governance.md` is not already there, write it too.

Then stop, without touching the freeze marker. End with a note telling the
Solution Designer to review and then run the freeze script to ratify.

## After bootstrap

Report, then point out three things.

**Report** (from step 0): the Method Plan the human selected (or `not selected`), detected ADEs,
participating ADEs, the Primary ADE, files
installed, ADEs explicitly excluded, and whether native support exists for each
participating ADE. Give this report in chat, and also append it to `.gtt/docs/gtt-completion.md`,
following the exact shape in `AGENTS.md` → *Completion report* — that file
is the durable record; chat output alone is lost once the session ends.
Append, don't overwrite, if the file already has an entry from a prior
bootstrap or re-run.

Before stopping, check `gtt-domain/backlog.md`: if the source document or the
conversation surfaced Epics or Stories, ask whether they should be recorded
there now (via `gtt-propose-change`, form 4, same as any other backlog
change) — do not silently leave them out, and do not invent ones that
weren't actually stated. If nothing like that came up, say so plainly and
move on; an empty backlog is a valid state, not a gap to fill by guessing.

If the host project already contains source files with `@GTTGuard` markers
(a prior partial setup, or code copied in before GTT was bootstrapped), run
`bash .gtt/scripts/gtt-guard-sync.sh` once the workspace is in place so
`.gtt/protection/registry.yaml` reflects them from the start, rather than
leaving it stale until someone happens to run the sync later.

Point out three things:

- `stack.md`'s "Locked by" column should reference `ADR-001` for now; later
  decisions get their own ADR through the normal change flow.
- If a `SOURCE-BRIEF.*` was created, it now sits permanently at the project
  root — the original design intent, kept for anyone who later asks why the
  context says what it says.
- Run `bash .gtt/scripts/gtt-validate.sh` (every participating ADE's integration is
  checked against `.gtt/ade.json`) and, in each participating ADE, confirm only its
  own always-loaded file (`.claude/CLAUDE.md` for Claude Code, `AGENTS.md` itself
  for Kiro/Codex/Copilot), `AGENTS.md`, and `constraints.md` load — the same check
  the README asks for after any setup.
