# GTT Bootstrap 1.0 + GTT CLI v1.0 — Master Implementation Task

> **GTT Governance Canonical:**  
> https://github.com/GTT-Community/gtt-method/blob/main/GTT-CANONICAL-v2.1.md

> **Document status:** Master implementation specification  
> **Scope:** Complete GTT CLI v1.0 development from zero + complete Bootstrap 1.0 contract implementation  
> **Implementation:** CLI in Go + Cobra; Bootstrap implementation according to its own runtime technology  
> **Rule:** This single document is the consolidated implementation source for the CLI/Bootstrap boundary.

## 0. Master Scope

This document intentionally consolidates **everything defined across the previous CLI implementation work and the subsequent Bootstrap 1.0 contract work**. A developer starting the CLI from zero must be able to use this document as the single implementation baseline, without needing to merge separate CLI and Bootstrap specifications manually.

The architecture remains:

```text
GTT Method
    ↓
GTT Bootstrap 1.0
    ↓
versioned contracts / capabilities
    ↓
GTT CLI v1.0
    ↓
ADE
    ↓
Human
```

The Bootstrap owns **what GTT means, how GTT governs, and the semantic contracts**. The CLI owns **installation, detection, selection, orchestration, invocation, lifecycle, compatibility, export and recovery**.

The remainder of this document preserves both bodies in full: first the Bootstrap 1.0 contract implementation task, then the complete CLI v1.0 implementation specification. Nothing from either source document is intentionally omitted.

---

# PART I — GTT Bootstrap 1.0 Contract Implementation

## 1. Objective

Complete **GTT Bootstrap 1.0** so that it exposes every contract required by **GTT CLI v1.0** without moving GTT methodology into the CLI.

The target architecture is:

```text
GTT Method
    ↓
GTT Bootstrap 1.0
    ↓
versioned contracts/capabilities
    ↓
GTT CLI v1.0
    ↓
ADE
    ↓
Human
```

The Bootstrap is responsible for **what GTT means and how GTT governs**.

The CLI is responsible for **how Bootstrap is installed, selected, invoked, validated, updated, exported and recovered**.

---

# 2. Mandatory Bootstrap 1.0 Deliverables

Bootstrap 1.0 must provide the following machine-readable contracts.

## 2.1 Bootstrap release identity

Provide an authoritative release descriptor:

```yaml
bootstrap:
  id: gtt-bootstrap
  version: 1.0.0
  schema_version: 1
  channel: stable
```

It must be distinct from:

```text
scaffold_version
```

The CLI must be able to identify the exact Bootstrap release without parsing README text or Git branch names.

---

# 3. CLI Compatibility Contract

Bootstrap must declare:

```yaml
compatibility:
  cli:
    min_version: 1.0.0
    max_version: <optional>
  schema:
    version: 1
```

If a capability requires a newer CLI:

```text
CLI detects incompatibility
→ refuses safely
→ no project modification
```

Bootstrap must never require the CLI to infer compatibility from implementation details.

---

# 4. Bootstrap Capability Registry

Bootstrap must expose a machine-readable capability registry.

Conceptually:

```yaml
capabilities:
  - id: project.detect
    version: 1
  - id: ade.detect
    version: 1
  - id: ade.install
    version: 1
  - id: template.materialize
    version: 1
  - id: methodology.profile
    version: 1
  - id: validation
    version: 1
  - id: freeze
    version: 1
  - id: session-context
    version: 1
  - id: export-policy
    version: 1
  - id: recovery
    version: 1
```

The exact schema is Bootstrap-owned.

The important invariant is:

> The CLI can negotiate capabilities without knowing GTT methodology internals.

---

# 5. Methodology Application Profiles

Bootstrap 1.0 must formally define:

```text
LIGHT
MEDIUM
HARD
```

The CLI only selects the profile.

Bootstrap defines its meaning.

## 5.1 Required profile contract

Conceptually:

```yaml
methodology:
  profiles:
    contract_version: 1

    supported:
      - id: light
        label: Light
      - id: medium
        label: Medium
      - id: hard
        label: Hard

    default: medium
```

Bootstrap must additionally expose the semantic behavior associated with each profile.

That behavior may include:

- proposal rigor;
- confirmation requirements;
- documentation depth;
- ADR expectations;
- change-process requirements;
- validation gates;
- freeze expectations;
- audit requirements;
- promotion rigor;
- evidence requirements.

The CLI must never implement these rules.

---

# 6. Required Profile Semantics

Bootstrap must define the intended application of the three levels.

### Light

For:

```text
POCs
experiments
spikes
disposable tests
exploratory development
short-lived technical work
```

The Bootstrap should define which governance steps can be reduced and which invariants remain mandatory.

### Medium

For:

```text
normal development
features
integrations
internal services
controlled prototypes
```

Bootstrap defines the normal governance path.

### Hard

For:

```text
production
critical systems
security-sensitive changes
high-consequence architecture
regulated work
```

Bootstrap defines the stricter governance path.

These are not CLI rules.

They are Bootstrap methodology semantics.

---

# 7. Non-Negotiable Governance Invariants

Even Light mode must not silently disable core human authority.

Bootstrap must explicitly define which controls remain mandatory across all profiles, including where applicable:

```text
human decision authority
provenance
evidence boundary
proposal distinction
conflict visibility
freeze semantics
protected artifact rules
```

If a control can be relaxed in Light, Bootstrap must define exactly how.

---

# 8. ADE Registry Contract

Bootstrap must expose the supported ADE registry in machine-readable form.

Each ADE entry must define at least:

```yaml
id:
name:
detect:
install:
validate:
owned_paths:
handoff:
invoke:
version:
```

The CLI must not maintain its own permanent ADE catalog.

Bootstrap 1.0 must support the ADEs already defined by the current Bootstrap implementation.

---

# 9. ADE Participation Contract

Bootstrap must define the distinction:

```text
detected
participating
primary
excluded
```

The state must support:

```text
zero or more participating secondary ADEs
exactly one Primary ADE
```

The CLI collects the human choice.

Bootstrap validates the resulting state.

---

# 10. Primary ADE Contract

Bootstrap must expose an operational contract for:

```text
set Primary ADE
validate Primary ADE
change Primary ADE
preserve secondary ADEs
```

Changing Primary ADE must not silently reinstall or remove unrelated integrations.

---

# 11. Initial Design Questionnaire Contract

Bootstrap must formally expose:

```yaml
scaffold:
  initial_design:
    questionnaire:
      template:
      output:
      version:
      contract_version:
```

The CLI must discover these values from Bootstrap.

Bootstrap owns:

- questionnaire content;
- questions;
- structure;
- elicitation guidance;
- evidence conventions;
- provenance;
- minimum viable design expectations;
- evolution.

---

# 12. Questionnaire Materialization Operation

Bootstrap must provide an operation equivalent to:

```text
template.materialize
```

for the Initial Design Questionnaire.

The CLI must not know the questionnaire content.

The CLI must not maintain a duplicate template.

---

# 13. Guided ADE Elicitation Contract

Bootstrap must provide the Primary ADE with explicit instructions for:

```text
inspect existing evidence
ask progressively
avoid redundant questions
identify missing information
identify conflicts
mark proposals
preserve provenance
ask human for decisions
avoid inventing architecture
```

The CLI only performs the handoff.

---

# 14. Initial Source Contract

Bootstrap must define how selected initial sources are represented.

The contract must preserve:

```text
path
source type
selection metadata
authority status
precedence status
version where available
```

The CLI records selection.

Bootstrap determines how sources enter grounding.

---

# 15. Source Authority Contract

Bootstrap must explicitly distinguish:

```text
selected source
    ≠
governed authority
```

If authority or precedence is required, Bootstrap must provide the mechanism.

The CLI must never infer authority.

---

# 16. Export Policy Contract

Bootstrap 1.0 must expose a machine-readable clean-export policy.

Conceptually:

```yaml
export:
  clean:
    include:
      - <patterns>
    exclude:
      - <patterns>
```

The policy must identify GTT-owned content, including ADE-owned GTT overlays.

The CLI must consume this policy.

The CLI must not duplicate an exclusion list.

---

# 17. Recovery Contract

Bootstrap must expose a versioned recovery contract.

It must define:

```text
snapshot schema
required metadata
Bootstrap identity
CLI compatibility
ADE state
Primary ADE
methodology profile
language
selected sources
operational state
recovery metadata
```

The CLI implements storage and lifecycle.

Bootstrap defines what must be preserved.

---

# 18. Session Contract

Bootstrap must expose a deterministic session-context operation.

The result must be derived from actual project state.

It must not depend on an agent's free-form recollection as authority.

The contract must identify:

```text
freeze state
change request
proposals
validation
Git state
operational state
relevant project artifacts
```

---

# 19. Validation Contract

Bootstrap must expose one authoritative validation entry point.

Conceptually:

```text
validation.run
```

It may internally execute:

```text
gtt-validate.sh
gtt-check-*.sh
```

but the CLI must consume one stable contract.

The CLI must not duplicate validation semantics.

---

# 20. Status Contract

Bootstrap must expose structured project status.

Conceptually:

```yaml
status:
  bootstrap:
  ade:
  methodology:
  sources:
  governance:
  freeze:
  validation:
  session:
```

The CLI renders it.

---

# 21. Freeze Contract

Bootstrap owns:

```text
freeze
```

and must expose an operation that the CLI can invoke.

The CLI must not implement freeze semantics.

There must be no CLI shortcut equivalent to:

```text
unfreeze
```

unless Bootstrap explicitly defines a governed transition.

---

# 22. GTTGuard Contract

Bootstrap must expose protection state and validation.

The CLI may invoke:

```text
guard.validate
guard.sync
```

but must not implement protection semantics.

---

# 23. Query / Index / Reconcile Contracts

Bootstrap must expose stable operational contracts for:

```text
index
reconcile
query
```

The CLI can expose them through inspection or diagnostics.

The CLI must not become a second retrieval/RAG implementation.

---

# 24. Clean Operation Contract

Bootstrap must distinguish:

```text
export --clean
```

from:

```text
clean
```

The first creates a separate delivery artifact.

The second removes GTT from the current project.

Bootstrap defines ownership.

CLI handles lifecycle and confirmation.

---

# 25. Bootstrap-Owned Operation Registry

Bootstrap must expose a registry that maps logical operations to implementation.

Example:

```yaml
operations:
  template.materialize:
    version: 1
  ade.detect:
    version: 1
  ade.install:
    version: 1
  validate:
    version: 1
  status:
    version: 1
  freeze:
    version: 1
  session-context:
    version: 1
  export-policy:
    version: 1
  recovery:
    version: 1
```

The implementation may be Bash, Python, Go, or another mechanism.

The CLI must not care.

---

# 26. No Arbitrary Shell Contract

Bootstrap must not require the CLI to execute arbitrary shell strings.

Operations should identify:

```text
operation
version
trusted implementation
arguments schema
inputs
outputs
```

The CLI executes declared Bootstrap operations only.

---

# 27. Bootstrap Evolution Contract

Bootstrap must support evolution without requiring CLI changes when the change remains within existing capabilities.

Example:

```text
new questionnaire
new template
new governance content
new ADE contract using existing ADE primitives
new profile semantics using existing profile capability
```

No CLI source change should be required.

---

# 28. New Capability Rule

If Bootstrap requires a capability absent from CLI:

```text
CLI detects requirement
→ compatibility failure
→ no modification
```

Bootstrap must not expect the CLI to guess or emulate missing capability.

---

# 29. Contract Versioning

Every externally consumed contract must have a version.

At minimum:

```text
Bootstrap release
Bootstrap schema
capability
operation
methodology profile
questionnaire
recovery
export policy
ADE integration
```

---

# 30. Bootstrap Acceptance Tests

Bootstrap 1.0 is complete only when these tests pass.

## Release identity

```text
CLI can identify exact Bootstrap release.
```

## Compatibility

```text
compatible CLI → PASS
incompatible CLI → REFUSE
```

## Profiles

```text
Light → valid
Medium → valid
Hard → valid
CLI receives profile
Bootstrap applies semantics
```

## ADE

```text
detect
participate
primary
secondary
validate
change primary
```

## Questionnaire

```text
no design
→ questionnaire contract
→ materialize
→ ADE handoff
```

## Export

```text
Bootstrap policy
→ CLI export
→ correct GTT exclusions
```

## Recovery

```text
snapshot
→ clean
→ detect
→ restore
→ validate
```

## Evolution

```text
Bootstrap content/contract-compatible change
→ CLI source unchanged
→ CLI works
```

## Unsupported capability

```text
Bootstrap requires X
CLI lacks X
→ REFUSE
→ project unchanged
```

---

# 31. Final Bootstrap 1.0 Definition of Done

- [ ] Machine-readable Bootstrap release identity exists.
- [ ] Bootstrap schema version exists.
- [ ] CLI compatibility contract exists.
- [ ] Capability registry exists.
- [ ] Operation registry exists.
- [ ] Light profile exists.
- [ ] Medium profile exists.
- [ ] Hard profile exists.
- [ ] Profile semantics are Bootstrap-owned.
- [ ] Mandatory governance invariants are defined.
- [ ] ADE registry is machine-readable.
- [ ] ADE participation state is defined.
- [ ] Primary ADE contract exists.
- [ ] Questionnaire contract exists.
- [ ] Questionnaire materialization operation exists.
- [ ] Guided ADE elicitation contract exists.
- [ ] Initial source contract exists.
- [ ] Source authority semantics are Bootstrap-owned.
- [ ] Clean export policy exists.
- [ ] Recovery contract exists.
- [ ] Session context contract exists.
- [ ] Validation contract exists.
- [ ] Status contract exists.
- [ ] Freeze contract exists.
- [ ] GTTGuard contract exists.
- [ ] Query/index/reconcile contracts exist.
- [ ] Operation registry is versioned.
- [ ] Arbitrary shell execution is not required.
- [ ] Contract evolution is supported.
- [ ] Unsupported capability fails closed.
- [ ] Bootstrap acceptance tests pass.

---

# 32. Final Boundary

The implementation must preserve:

```text
BOOTSTRAP

What GTT means
What GTT requires
How GTT governs
How evidence is treated
How proposals work
How freeze works
What Light/Medium/Hard mean
How ADE integrations work
What artifacts are owned
What validation means
```

versus:

```text
CLI

How to find Bootstrap
How to install Bootstrap
How to detect ADE
How to ask the user
How to select Primary ADE
How to execute an operation
How to validate operationally
How to update
How to recover
How to export
How to clean
How to report
```

This boundary is mandatory for GTT Bootstrap 1.0 + CLI v1.0.

---

# PART II — GTT CLI v1.0 Complete Implementation Specification

# 1. Purpose

This document is the final implementation-grade specification for **GTT
CLI v1.0**, reconciled against the supplied **GTT Bootstrap 1.0**.

The CLI is a professional operational tool for:

-   detecting a host project;
-   resolving and installing a compatible GTT Bootstrap;
-   detecting represented ADEs;
-   establishing the participating ADE set;
-   establishing exactly one Primary ADE;
-   detecting whether a sufficient project/design source exists;
-   selecting initial source material explicitly;
-   requesting the Bootstrap-owned Initial Design Questionnaire when
    necessary;
-   selecting the project's GTT methodology application profile;
-   passing that profile to Bootstrap contracts;
-   invoking Bootstrap services;
-   handing the resulting initialization contract to the Primary ADE;
-   exposing operational state;
-   validating the installation;
-   resuming from deterministic project state;
-   freezing through Bootstrap;
-   changing the Primary ADE explicitly;
-   updating Bootstrap safely;
-   producing recovery snapshots;
-   exporting clean delivery artifacts;
-   removing GTT safely when explicitly requested;
-   supporting CI/CD and machine-readable output;
-   maintaining strict CLI/Bootstrap compatibility.

The CLI must remain deterministic and must not become a second GTT
engine.

------------------------------------------------------------------------

# 2. Final Architectural Principle

The final system is:

``` text
                     GTT METHOD
                         │
                         ▼
                  GTT BOOTSTRAP 1.0
                         │
        ┌────────────────┼─────────────────┐
        │                │                 │
     Methodology      Engine           ADE integrations
     Governance       Domain           Templates/contracts
     Evidence         Services         Questionnaire
     Reasoning        Validation       Execution semantics
        │                │                 │
        └────────────────┼─────────────────┘
                         │
                 explicit contracts
                         │
                         ▼
                    GTT CLI v1.0
                         │
        ┌────────────────┼─────────────────┐
        │                │                 │
    Discovery       Lifecycle          Distribution
    Selection       Execution          Recovery
    Installation    Validation         Export/Clean
    Invocation      Status/Audit       Update
                         │
                         ▼
                    Human + ADE
```

The invariant is:

> **The CLI must never become the mind of GTT.**

The CLI may know **how to operate Bootstrap**.

The CLI must not know **how GTT reasons, governs evidence, decides
architecture, evaluates proposals, or defines methodology semantics**.

------------------------------------------------------------------------

# 3. Reconciliation Against GTT Bootstrap 1.0

The supplied Bootstrap establishes the following concrete capabilities.

## 3.1 Confirmed Bootstrap responsibilities

Bootstrap 1.0 owns:

-   the GTT methodology;
-   governance rules;
-   evidence boundary;
-   grounding;
-   provenance;
-   architecture/context semantics;
-   ADR semantics;
-   proposal semantics;
-   freeze semantics;
-   human decision boundary;
-   ADE-specific integration surfaces;
-   Multi-ADE state;
-   Primary ADE state;
-   ADE install ledger;
-   the scaffold manifest;
-   Bootstrap-owned templates;
-   Initial Design Questionnaire;
-   source-manifest template;
-   working-agreements template;
-   deterministic validation scripts;
-   status generation;
-   session context derivation;
-   artifact identity;
-   technical index;
-   reconciliation;
-   query/retrieval;
-   GTTGuard;
-   provenance checks;
-   freeze;
-   clean/export ownership metadata;
-   Session Memory adapter contracts.

## 3.2 Confirmed Bootstrap services relevant to the CLI

The supplied Bootstrap contains operational services including:

``` text
.gtt/scripts/gtt-ade.sh
.gtt/scripts/gtt-template.sh
.gtt/scripts/gtt-status.sh
.gtt/scripts/gtt-validate.sh
.gtt/scripts/gtt-freeze.sh
.gtt/scripts/gtt-session-context.sh
.gtt/scripts/gtt-index.sh
.gtt/scripts/gtt-reconcile.sh
.gtt/scripts/gtt-query.sh
.gtt/scripts/gtt-guard-sync.sh
.gtt/scripts/gtt-check-*.sh
```

The CLI must consume these through the Bootstrap's declared contract
rather than permanently hard-coding their filenames.

## 3.3 Confirmed Multi-ADE model

Bootstrap 1.0 defines:

``` text
one GTT governance model
several ADE integration surfaces
exactly one Primary ADE
```

The Primary ADE is a workflow identifier.

It has **no governance authority**.

The CLI must preserve this exact model.

## 3.4 Confirmed questionnaire ownership

Bootstrap 1.0 owns:

``` text
.gtt/scaffold/templates/gtt-initial-design-questionnaire.md
```

and its materialization contract.

The CLI must:

``` text
detect need
    ↓
request Bootstrap template
    ↓
materialize through Bootstrap
    ↓
report location
    ↓
handoff to Primary ADE
```

The CLI must never contain a copy of the questionnaire or its
methodology.

## 3.5 Confirmed absence of a recipe dependency

Bootstrap 1.0 does **not** require the CLI to interpret a specific
executable recipe.

The Bootstrap's declarative scaffold manifest describes the scaffold and
template registry. It is not an executable methodology recipe.

Therefore:

``` text
CLI → specific Bootstrap recipe
```

is NOT the architecture.

The correct boundary is:

``` text
CLI
  ↓
Bootstrap package / manifest / contracts
  ↓
Bootstrap services
  ↓
GTT semantics
```

The CLI may use generic operational contracts, but must not depend on
the existence of a particular recipe document.

------------------------------------------------------------------------

# 4. Bootstrap 1.0 Contract Status

The CLI v1.0 architecture treats the following as **Bootstrap 1.0
implementation contracts**, not CLI responsibilities.

A dedicated Bootstrap implementation task defines and tracks these
contracts:

```text
GTT_BOOTSTRAP_1.0_CLI_CONTRACT_IMPLEMENTATION_TASK.md
```

The CLI therefore assumes that Bootstrap 1.0 provides:

- machine-readable release identity;
- CLI compatibility contract;
- capability registry;
- operation registry;
- Light / Medium / Hard methodology profile contract;
- profile semantics;
- ADE registry and participation contract;
- Primary ADE contract;
- Initial Design Questionnaire contract;
- guided ADE elicitation contract;
- initial source contract;
- source authority semantics;
- clean export policy;
- recovery contract;
- session-context contract;
- validation contract;
- status contract;
- freeze contract;
- GTTGuard contract;
- query/index/reconcile contracts;
- versioned evolution and capability negotiation.

These are **not duplicated in the CLI**.

The CLI implementation must consume these contracts.

If a required Bootstrap contract is absent or incompatible, the CLI must
fail closed and report the missing or incompatible capability. It must
not recreate the missing GTT behavior in Go.

## 4.1 Methodology profile boundary

The CLI exposes:

```text
LIGHT
MEDIUM
HARD
```

but does not define their governance semantics.

The contract is:

```text
Human
  ↓
CLI asks for profile
  ↓
CLI records selection
  ↓
CLI passes profile through Bootstrap contract
  ↓
Bootstrap interprets profile
  ↓
Bootstrap applies governance behavior
```

This requirement is **assigned to and owned by Bootstrap 1.0**.

## 4.2 Bootstrap release identity boundary

The CLI consumes an authoritative Bootstrap release descriptor.

It does not infer release identity from:

- directory names;
- Git branches;
- Git commits;
- README text;
- scaffold version.

The Bootstrap release identity is an explicit dependency of CLI
compatibility.

## 4.3 Final implementation rule

The CLI is complete only when it can consume the Bootstrap 1.0 contracts
defined by the Bootstrap implementation task.

The CLI itself must not become responsible for implementing those
semantics.

# 5. Methodology Application Profiles

The CLI must support three selectable profiles:

``` text
LIGHT
MEDIUM
HARD
```

The names are operational selectors.

Their actual methodology semantics belong to Bootstrap.

## 5.1 Intended use

The user should be able to choose the intensity appropriate to the work.

Examples:

``` text
LIGHT
  quick experiments
  POCs
  disposable tests
  exploratory development
  small technical spikes

MEDIUM
  normal feature development
  integration work
  internal services
  controlled prototypes
  non-trivial changes

HARD
  production systems
  security-sensitive changes
  critical infrastructure
  major architecture changes
  regulated or high-consequence work
```

These examples are guidance for the selection UX, not a second
governance implementation.

## 5.2 What the CLI must not do

The CLI must not contain rules such as:

``` text
LIGHT = skip ADR
MEDIUM = require ADR sometimes
HARD = always require ADR
```

Those rules belong to Bootstrap.

## 5.3 Bootstrap profile contract

The CLI expects a Bootstrap capability conceptually equivalent to:

``` yaml
methodology:
  profiles:
    supported:
      - light
      - medium
      - hard

    default: medium

    semantics:
      provider: bootstrap
      contract_version: 1
```

The exact schema is owned by Bootstrap.

## 5.4 Profile selection UX

Example:

``` text
GTT Methodology Application

Choose the level of GTT application for this project:

  [1] Light
      POCs, experiments, spikes and disposable work.

  [2] Medium
      Normal development and controlled project work.

  [3] Hard
      Production, critical or high-consequence work.

Select [1-3]:
```

The CLI records the human selection.

It does not interpret the selected profile.

## 5.5 Profile persistence

The selected profile must be recorded in operational project state.

Example:

``` yaml
methodology:
  profile: medium
  selected_at: <timestamp>
  contract_version: 1
```

The location and exact schema should be Bootstrap-defined.

## 5.6 Profile changes

Changing the profile after initialization must be explicit.

The CLI must never silently change:

``` text
LIGHT → MEDIUM
MEDIUM → HARD
HARD → MEDIUM
```

The CLI asks the Bootstrap whether the change is:

-   operationally safe;
-   allowed;
-   requires confirmation;
-   requires a governed change;
-   requires migration.

The CLI does not decide those semantics.

------------------------------------------------------------------------

# 6. Final Responsibility Boundary

  Responsibility                                  Bootstrap 1.0                           CLI v1.0
  ------------------------------------ ------------------------ ----------------------------------
  GTT methodology                                           YES                                 NO
  Governance rules                                          YES                                 NO
  Evidence boundary                                         YES                                 NO
  Grounding                                                 YES                                 NO
  Provenance semantics                                      YES                                 NO
  Architecture semantics                                    YES                                 NO
  ADR semantics                                             YES                                 NO
  Proposal semantics                                        YES                                 NO
  Freeze semantics                                          YES                                 NO
  Human decision rules                                      YES                                 NO
  ADE integration definitions                               YES               Detect/select/invoke
  ADE registry                                              YES                            Consume
  Primary ADE state semantics                               YES                    Collect/operate
  Scaffold definition                                       YES                            Consume
  Questionnaire                                             YES     Detect/materialize/orchestrate
  Source manifest semantics                                 YES   Select/register through contract
  Working agreements semantics                              YES                        Orchestrate
  GTTGuard                                                  YES                    Invoke/validate
  Provenance checks                                         YES                      Invoke/report
  Artifact identity                                         YES                      Invoke/report
  Technical index                                           YES                      Invoke/report
  Session Memory                                            YES                      Invoke/report
  Status semantics                                          YES                            Present
  Validation semantics                                      YES                      Invoke/report
  Bootstrap resolution                          Contract source                                YES
  Bootstrap installation                         Content source                                YES
  Bootstrap update                               Package source                                YES
  Project root detection                         Contract input                                YES
  ADE candidate detection                   Contract definition                                YES
  Primary ADE selection UX                    State destination                                YES
  Initial source selection                        Data consumer                                YES
  Language selection                           Content provider                                YES
  Methodology profile selection               Semantic provider                                YES
  Methodology profile interpretation                        YES                                 NO
  Recovery snapshot                              State contract                                YES
  Clean export                               Exclusion contract                                YES
  Destructive clean                      Installation lifecycle                                YES
  Process execution                           Scripts/contracts                                YES
  Security of CLI operations                                ---                                YES
  CLI compatibility                        Contract declaration                                YES
  CLI exit codes                                            ---                                YES
  JSON output                                               ---                                YES

This boundary is mandatory.

------------------------------------------------------------------------

# 7. CLI Operational Model

The CLI is an operational runtime around Bootstrap.

Its core primitives are:

``` text
discover
resolve
verify
stage
install
select
invoke
validate
inspect
report
snapshot
restore
update
export
clean
```

The CLI must not contain GTT semantic primitives such as:

``` text
ground
reason
decide
approve
ratify
architect
evaluate evidence
interpret proposal
```

Those belong to Bootstrap/ADE/human interaction.

------------------------------------------------------------------------

# 8. Bootstrap Resolution

The CLI must resolve Bootstrap independently of any specific recipe.

Recommended resolution order:

``` text
1. Explicit --bootstrap / local package
2. Project-recorded compatible Bootstrap
3. Local verified cache
4. Configured registry/distribution endpoint
5. Canonical distribution source
```

Example:

``` text
gtt init --bootstrap /path/to/gtt-bootstrap-1.0.0
```

Offline:

``` text
gtt init --offline
```

Specific version:

``` text
gtt init --bootstrap-version 1.0.0
```

The CLI must record:

``` text
Bootstrap ID
Bootstrap version
Bootstrap schema
distribution channel
source
checksum
signature/integrity result
resolved timestamp
```

## 8.1 Production distribution

The recommended model is:

``` text
Canonical GTT Bootstrap repository
          ↓
versioned release artifact
          ↓
distribution/registry endpoint
          ↓
GTT CLI resolver
          ↓
verified local cache
          ↓
project installation
```

The CLI must not make every operation depend on live GitHub/network
access.

Network is primarily required for:

-   initial Bootstrap resolution;
-   explicit update;
-   cache miss;
-   requested version not locally available.

------------------------------------------------------------------------

# 9. Bootstrap Cache

The CLI may maintain a user-level cache:

``` text
~/.gtt/cache/bootstrap/<version>/
```

The cache must be outside the governed project.

Cached packages must be verified before use.

The cache is not GTT governance.

------------------------------------------------------------------------

# 10. Bootstrap Compatibility

Compatibility is a hard gate.

The CLI must validate at least:

``` text
CLI version
Bootstrap release version
Bootstrap schema version
Bootstrap capability contract versions
scaffold version
ADE overlay contract versions
methodology profile contract version
```

If compatibility fails:

``` text
STOP
NO MODIFICATION
```

Example:

``` text
Bootstrap 1.0.0 requires CLI >= 1.0.0.

Current CLI:
1.0.0

Compatibility:
PASS
```

Failure:

``` text
Bootstrap requires CLI >= 1.1.0.

Current CLI:
1.0.0

No project files were modified.
```

------------------------------------------------------------------------

# 11. Bootstrap Verification

Before execution, the CLI must verify:

``` text
package structure
release metadata
checksum
signature when supported
manifest
schema
compatibility
path safety
```

A failed integrity check is fatal.

The CLI must fail closed.

------------------------------------------------------------------------

# 12. Project Detection

`gtt init` must detect:

-   project root;
-   Git repository;
-   existing `.gtt`;
-   existing `gtt-domain`;
-   Bootstrap metadata;
-   candidate ADE representations;
-   candidate source/design documents;
-   recovery snapshots;
-   host-file conflicts.

The CLI must not assume the current directory is always the project
root.

If Git is present, repository root should normally be preferred when
appropriate.

If a project cannot be confidently identified:

``` text
No project detected.

Initialize GTT in this directory?
```

The user must explicitly confirm.

------------------------------------------------------------------------

# 13. Existing GTT Detection

If GTT is already installed:

``` text
GTT is already initialized in this project.

Bootstrap: 1.0.0
CLI:        1.0.0
Primary ADE: Claude Code

Choose:

[1] Inspect
[2] Validate
[3] Update
[4] Reconfigure operational settings
[5] Abort
```

`gtt init` must never blindly reinstall.

------------------------------------------------------------------------

# 14. Conflict Detection

The CLI must detect conflicts involving:

``` text
.gtt/
gtt-domain/
AGENTS.md
readme-gtt.md
readme-gtt.es.md
ADE overlays
SOURCE-BRIEF.*
```

A conflict is not an invitation to merge automatically.

The CLI must:

1.  report;
2.  explain the conflicting paths;
3.  avoid destructive modification;
4.  request explicit user direction when needed.

------------------------------------------------------------------------

# 15. ADE Detection

The CLI must support Multi-ADE.

Bootstrap 1.0 currently declares the ADE registry through the scaffold
manifest.

Current declared ADEs include:

``` text
Claude Code
Kiro
GitHub Copilot
Codex
```

The CLI must obtain the supported ADE registry from Bootstrap.

The CLI must not permanently hard-code this list.

## 15.1 Detection is not participation

The distinction is mandatory:

``` text
DETECTED
    ≠
PARTICIPATING
```

An ADE may be detected as a candidate without being selected for GTT
participation.

The CLI must never install an ADE merely because it detected it.

------------------------------------------------------------------------

# 16. ADE Selection

The CLI should present:

``` text
Detected ADE candidates:

[1] Claude Code
[2] GitHub Copilot
[3] Codex

Select ADEs that will participate:
> 1,2,3
```

Then:

``` text
Select Primary ADE:
> 1
```

The human must explicitly establish participation and Primary ADE.

## 16.1 Exactly one Primary ADE

The final project state must contain exactly one Primary ADE.

If multiple participating ADEs exist:

``` text
Primary ADE = exactly one
Secondary ADEs = zero or more
```

The Primary ADE has no governance authority.

------------------------------------------------------------------------

# 17. Existing ADE State

When `.gtt/ade.json` exists, the CLI must use Bootstrap's ADE
service/state rather than infer a new state.

It must preserve:

-   participating ADEs;
-   primary ADE;
-   excluded ADEs;
-   install ledger.

The CLI must not hand-edit `.gtt/ade.json`.

------------------------------------------------------------------------

# 18. ADE Installation

The CLI orchestrates Bootstrap's ADE service.

Conceptually:

``` text
gtt-ade detect
gtt-ade install
gtt-ade validate
gtt-ade state
```

The exact invocation comes from the Bootstrap contract.

The CLI must:

-   install only selected participating ADEs;
-   preserve excluded ADEs;
-   record Primary ADE;
-   validate every participating integration;
-   avoid overwriting files not owned by GTT;
-   preserve the Bootstrap's rollback behavior.

------------------------------------------------------------------------

# 19. Primary ADE Changes

A Primary ADE change must be explicit.

Example:

``` text
Current Primary ADE:
Claude Code

Participating secondary ADEs:
Copilot
Codex

Select new Primary:
Copilot

Confirm? [y/N]
```

The CLI must delegate semantic validation to Bootstrap.

It must never infer Primary ADE from:

-   last-used ADE;
-   most recently modified files;
-   installed executable;
-   environment variable;
-   detected directory.

------------------------------------------------------------------------

# 20. Language Selection

The CLI must support the project interaction language where Bootstrap
supports it.

Example:

``` text
Language:

[1] Español
[2] English
```

Language may affect:

-   CLI messages;
-   Bootstrap-generated handoff;
-   questionnaire content where supported;
-   ADE-facing contract.

The CLI must not duplicate GTT methodology translations.

------------------------------------------------------------------------

# 21. Initial Design Source Detection

The CLI must inspect the project for candidate design/source material.

Examples:

``` text
README.md
docs/
design/
architecture/
requirements/
specification/
*.md
*.pdf
*.docx
```

The exact supported formats must come from Bootstrap/CLI capability
contracts.

The CLI must never decide that a document is authoritative.

------------------------------------------------------------------------

# 22. One Candidate Document

If one plausible document is found:

``` text
Initial project source detected:

docs/architecture.md

Use this as initial source material? [Y/n]
```

The user must confirm.

------------------------------------------------------------------------

# 23. Multiple Candidate Documents

If multiple candidates exist:

``` text
Candidate source documents:

[1] README.md
[2] docs/requirements.md
[3] docs/architecture.md
[4] docs/api.md

Select one or more:
> 2,3,4
```

The CLI records selection.

It does not decide precedence.

If authority/precedence is required, Bootstrap's source-manifest process
owns it.

------------------------------------------------------------------------

# 24. No Sufficient Design Document

If no sufficient source document exists:

``` text
No sufficient design source was detected.

GTT can start with the Bootstrap-owned Initial Design Questionnaire.

Create it?
[Y/n]
```

The CLI must invoke the Bootstrap template service.

Current Bootstrap contract:

``` text
.gtt/scripts/gtt-template.sh
```

and template:

``` text
.gtt/scaffold/templates/gtt-initial-design-questionnaire.md
```

Materialization target is Bootstrap-defined and currently:

``` text
gtt-domain/proposals/bootstrap/initial-design-questionnaire.md
```

The CLI must obtain this from Bootstrap rather than hard-coding the path
as a semantic rule.

------------------------------------------------------------------------

# 25. Questionnaire Ownership

The CLI must never contain:

-   questionnaire questions;
-   questionnaire methodology;
-   elicitation logic;
-   completeness scoring;
-   design reasoning prompts.

The CLI only orchestrates:

``` text
detect
→ request
→ materialize
→ report
→ handoff
```

The Primary ADE then conducts the Bootstrap-defined guided elicitation.

------------------------------------------------------------------------

# 26. Questionnaire Human Boundary

The questionnaire remains source material.

Completion does not automatically make it governed architecture.

The CLI must never claim:

``` text
Questionnaire complete = architecture approved
```

The Bootstrap defines the transition into governed context.

------------------------------------------------------------------------

# 27. Initial Source Preservation

When a source document is selected, Bootstrap must preserve it according
to its contract.

The current Bootstrap convention is:

``` text
SOURCE-BRIEF.*
```

The CLI must invoke the Bootstrap preservation mechanism.

It must not silently modify the source.

------------------------------------------------------------------------

# 28. GTT Methodology Profile Selection

During `gtt init`, after the project and Bootstrap are known, the CLI
must ask:

``` text
How strongly should GTT governance be applied to this project?

[1] Light
[2] Medium
[3] Hard
```

The selection is human-owned.

The CLI records and passes it.

Bootstrap interprets it.

------------------------------------------------------------------------

# 29. Profile Semantics --- Required Bootstrap Responsibility

The CLI must not contain a table such as:

  Profile   ADR           Proposal   Freeze
  --------- ------------- ---------- ---------
  Light     optional      light      relaxed
  Medium    conditional   normal     normal
  Hard      required      strict     strict

That would duplicate GTT methodology.

Instead Bootstrap exposes a capability/semantic contract.

Conceptually:

``` yaml
methodology:
  profile:
    id: medium
    contract_version: 1
    behavior_provider: bootstrap
```

Bootstrap may define internally:

-   proposal thresholds;
-   confirmation requirements;
-   change handling;
-   freeze expectations;
-   documentation depth;
-   audit requirements;
-   promotion rigor;
-   validation gates.

The CLI remains unaware of those semantics.

------------------------------------------------------------------------

# 30. Profile UX Guidance

The CLI may explain the intended scope without defining methodology:

``` text
Light
  For POCs, experiments, spikes and disposable work.

Medium
  For normal development and controlled project work.

Hard
  For production or high-consequence work.
```

The user chooses.

The Bootstrap decides what that means operationally.

------------------------------------------------------------------------

# 31. Profile Change

A future operational command may be:

``` text
gtt config methodology
```

or equivalent.

The CLI must ask Bootstrap whether a profile transition is permitted.

If Bootstrap says that a governed process is required:

``` text
This change requires the project's GTT change process.

No direct profile change was made.
```

The CLI does not override the response.

------------------------------------------------------------------------

# 32. Initialization Contract

After collection, CLI produces structured operational input:

``` yaml
project:
  root: /project

language: en

ade:
  detected:
    - claude
    - copilot
    - codex
  participating:
    - claude
    - copilot
  primary: claude

sources:
  selected:
    - docs/requirements.md
    - docs/architecture.md

methodology:
  profile: medium

bootstrap:
  id: gtt-bootstrap
  version: 1.0.0
```

This is input to Bootstrap.

Bootstrap transforms it into the actual ADE-facing initialization
contract.

------------------------------------------------------------------------

# 33. Primary ADE Handoff

The CLI must not embed prompts such as:

``` text
You are an architect...
```

The Bootstrap owns the instructions.

The CLI hands the Primary ADE a Bootstrap-generated contract containing
the necessary operational context.

Conceptually:

``` text
Bootstrap installed
    ↓
Bootstrap generates/defines handoff
    ↓
CLI delivers handoff
    ↓
Primary ADE
```

------------------------------------------------------------------------

# 34. ADE Invocation Modes

The CLI should support:

``` text
install
handoff
invoke
```

Example:

``` text
GTT Bootstrap completed.

Primary ADE: Claude Code

[1] Launch Claude Code
[2] Show Bootstrap handoff
[3] Exit
```

Automatic invocation may be supported where the ADE adapter declares it
safe.

The CLI must not invoke external agents silently when that invocation
could cause meaningful side effects.

------------------------------------------------------------------------

# 35. CLI Command Surface

The final v1.0 command surface is:

``` text
gtt init

gtt status

gtt inspect

gtt validate

gtt resume

gtt freeze

gtt doctor

gtt audit

gtt update

gtt export --clean

gtt clean

gtt version
```

Operational configuration may be exposed as subcommands only where
required:

``` text
gtt config primary-ade
gtt config methodology
gtt config language
```

These are operational configuration commands and must delegate semantic
validation to Bootstrap.

------------------------------------------------------------------------

# 36. `gtt init`

`init` is the primary orchestration workflow.

Final flow:

``` text
gtt init
  │
  ├─ detect project
  ├─ detect existing GTT
  ├─ resolve Bootstrap
  ├─ verify Bootstrap
  ├─ check CLI/Bootstrap compatibility
  ├─ load Bootstrap contracts
  ├─ detect ADE candidates
  ├─ choose participating ADEs
  ├─ choose Primary ADE
  ├─ choose language
  ├─ discover source documents
  ├─ select initial source set
  ├─ if insufficient → Bootstrap questionnaire
  ├─ choose methodology profile
  ├─ stage installation
  ├─ install Core
  ├─ install participating ADE integrations
  ├─ persist operational state
  ├─ validate
  ├─ create Bootstrap handoff
  ├─ optionally invoke Primary ADE
  └─ report final state
```

No architecture is invented by CLI.

------------------------------------------------------------------------

# 37. `gtt status`

Status must be deterministic.

Example:

``` text
GTT Project Status

CLI:
  1.0.0

Bootstrap:
  1.0.0
  Compatibility: PASS

ADE:
  Primary: Claude Code
  Participating: Claude Code, Copilot
  Excluded: Kiro, Codex

Methodology:
  Medium

Sources:
  Initial sources: 2

GTT:
  Validation: PASS
  Freeze: ACTIVE
  Change Request: NONE
  Session: CURRENT
```

OPEN/CONFLICT/BLOCKING information must come from Bootstrap.

The CLI may present it.

It must not classify gaps itself.

------------------------------------------------------------------------

# 38. `gtt inspect`

Inspect displays topology and installation state.

It may show:

``` text
Project root
Git root
Bootstrap
Bootstrap schema
CLI compatibility
ADE candidates
Participating ADEs
Primary ADE
Methodology profile
Source documents
GTT scaffold
Protected paths
Generated paths
Session state
Freeze state
Validation
```

Inspect must not interpret architecture.

------------------------------------------------------------------------

# 39. `gtt validate`

Validation delegates to Bootstrap.

Examples:

``` text
gtt validate
gtt validate --ci
gtt validate --json
```

The CLI must not recreate `gtt-check-*` semantics.

The Bootstrap remains responsible for deterministic governance checks.

------------------------------------------------------------------------

# 40. CI Validation

`gtt validate --ci` must:

-   be non-interactive;
-   return stable exit codes;
-   invoke Bootstrap validation;
-   not launch an ADE;
-   not modify governed architecture;
-   support machine-readable output.

------------------------------------------------------------------------

# 41. `gtt resume`

Resume reconstructs operational context from actual project state.

Sources may include:

``` text
Git activity
Bootstrap state
ADE state
session.md
change-request.md
proposals
ADR state
freeze marker
validation state
audit state
```

The CLI must not trust free-form agent prose as authoritative state.

Session memory remains operational.

------------------------------------------------------------------------

# 42. `gtt freeze`

Freeze is a Bootstrap facade.

Conceptually:

``` text
gtt freeze
   ↓
Bootstrap freeze contract
   ↓
Bootstrap validation
   ↓
human confirmation
   ↓
freeze
```

The CLI must not implement freeze semantics.

There is no:

``` text
gtt unfreeze
```

Post-freeze architecture changes follow Bootstrap's governed change
process.

------------------------------------------------------------------------

# 43. `gtt doctor`

Doctor is CLI-native operational diagnostics.

It may check:

``` text
CLI
Git
project root
Bootstrap
Bootstrap integrity
compatibility
ADE detection
ADE state
Primary ADE
methodology profile contract
required paths
permissions
Bootstrap validation
session state
```

Doctor must distinguish:

``` text
PASS
WARN
FAIL
CANNOT DETERMINE
```

Doctor must not silently repair the project.

------------------------------------------------------------------------

# 44. `gtt audit`

Audit is an operational traceability facade.

It consumes Bootstrap audit/index/provenance information.

Supported output:

``` text
gtt audit
gtt audit --json
gtt audit --md
```

The CLI may display:

-   Bootstrap release;
-   CLI release;
-   compatibility;
-   ADE state;
-   Primary ADE;
-   selected sources;
-   methodology profile;
-   validation state;
-   freeze state;
-   proposals;
-   change requests;
-   recovery snapshots;
-   updates;
-   clean exports.

Audit is not a second governance engine.

------------------------------------------------------------------------

# 45. `gtt update`

Update must be transactional.

Flow:

``` text
current state
    ↓
resolve target Bootstrap
    ↓
verify package
    ↓
check compatibility
    ↓
check migration contract
    ↓
create snapshot when required
    ↓
stage update
    ↓
apply
    ↓
validate
    ↓
commit
```

On incompatibility:

``` text
STOP
NO MODIFICATION
```

------------------------------------------------------------------------

# 46. Update Preservation

Update must preserve unless Bootstrap migration explicitly changes them:

``` text
Primary ADE
participating ADEs
excluded ADEs
selected sources
methodology profile
language
governed project artifacts
operational state
```

The CLI must not reset these merely because Bootstrap is updated.

------------------------------------------------------------------------

# 47. `gtt export --clean`

The purpose is to create a clean QA/PRO delivery artifact while leaving
the GTT development project intact.

Example:

``` text
gtt export --clean
```

or:

``` text
gtt export --clean ./dist/project
```

The exact exclusion list is supplied by Bootstrap.

The CLI must not maintain a competing hard-coded exclusion list.

Current Bootstrap ADE ownership information is especially important
because only GTT-owned ADE paths may be removed during clean/export
operations.

------------------------------------------------------------------------

# 48. `gtt clean`

`clean` is destructive.

It removes GTT from the current project according to Bootstrap
ownership/exclusion contracts.

It must require explicit confirmation.

Example:

``` text
WARNING

This operation removes GTT from the current project.

A recovery snapshot can be created first.

Continue? [y/N]
```

------------------------------------------------------------------------

# 49. Recovery Snapshot

Before destructive clean, the CLI should offer:

``` text
Create GTT recovery snapshot? [Y/n]
```

The snapshot must be defined by Bootstrap's recovery contract.

Potential state includes:

``` text
Bootstrap version
Bootstrap checksum
CLI compatibility
ADE state
Primary ADE
methodology profile
language
selected sources
installation state
session state
audit metadata
recovery metadata
```

The snapshot is not a complete application backup.

------------------------------------------------------------------------

# 50. Recovery

A future `gtt init` must detect a compatible recovery snapshot.

Example:

``` text
GTT recovery snapshot detected.

Previous Bootstrap:
1.0.0

Primary ADE:
Claude Code

Methodology:
Medium

Restore GTT installation?
[Y/n]
```

Before modification:

``` text
verify snapshot
verify Bootstrap
verify compatibility
stage restore
validate
commit
```

------------------------------------------------------------------------

# 51. Clean vs Export

  -------------------------------------------------------------------------------
  Operation                 Current project        GTT removed New clean artifact
                                   modified                    
  ---------------------- ------------------ ------------------ ------------------
  `gtt export --clean`                   NO                 NO                YES

  `gtt clean`                           YES                YES                 NO
  -------------------------------------------------------------------------------

This distinction is fundamental.

------------------------------------------------------------------------

# 52. Operational State

The CLI must not create a competing state manifest if Bootstrap already
provides one.

Conceptual operational state:

``` yaml
gtt:
  bootstrap:
    id:
    version:
    schema:
    checksum:
  cli:
    version:
  project:
    root:
  ade:
    primary:
    participating:
  methodology:
    profile:
  language:
  sources:
  snapshot:
```

The authoritative schema is supplied by Bootstrap.

------------------------------------------------------------------------

# 53. State Precedence

The CLI must preserve:

``` text
GTT Canonical Method
        ↓
Bootstrap implementation/contracts
        ↓
Governed project artifacts
        ↓
Operational state
        ↓
Session state
        ↓
Agent notes
```

Lower layers cannot silently override higher layers.

------------------------------------------------------------------------

# 54. Session Memory

Session memory is operational.

It is not:

-   evidence;
-   architecture;
-   authority;
-   an ADR;
-   a decision record.

The CLI should invoke Bootstrap's current session service.

The current Bootstrap exposes:

``` text
gtt-session-context
```

The CLI consumes its result.

It must not invent a parallel memory system.

------------------------------------------------------------------------

# 55. Source Manifest

When multiple sources require authority/precedence/provenance handling,
the CLI may offer:

``` text
Bootstrap source-manifest template
```

The CLI does not decide:

``` text
source A > source B
```

That is Bootstrap methodology/human governance.

------------------------------------------------------------------------

# 56. Working Preferences

The CLI may expose Bootstrap-owned working-agreement/preference
facilities.

The CLI must preserve the distinction:

``` text
working preference
    ≠
governed constraint
```

A preference cannot silently override L0/L1 governance.

------------------------------------------------------------------------

# 57. GTTGuard

GTTGuard is Bootstrap-owned.

The CLI may:

``` text
validate
report
invoke synchronization
```

It must not implement GTTGuard semantics independently.

Protected artifacts are distinct from the L0/L1 human promotion
boundary.

------------------------------------------------------------------------

# 58. Evidence Boundary

The CLI must preserve:

``` text
Authorized Sources
        ↓
Grounding
        ↓
Evidence Dossier
        ↓
Reasoning
        ↓
Marked Proposal
        ↓
Human
        ↓
Freeze
```

The CLI must never introduce:

``` text
raw source
   ↓
CLI
   ↓
agent decision
```

The CLI only registers/selects initial source material.

------------------------------------------------------------------------

# 59. Human Authority

The human remains authority for:

-   ADE participation;
-   Primary ADE selection;
-   source selection;
-   methodology profile selection;
-   architectural decisions;
-   proposal approval;
-   freeze;
-   destructive operations;
-   operational configuration changes where Bootstrap requires
    confirmation.

The CLI must not make architectural judgments.

------------------------------------------------------------------------

# 60. Security

The CLI must assume repositories and packages can be untrusted.

Minimum protections:

## 60.1 Path traversal

Reject:

``` text
../../outside-project
```

and unsafe absolute paths unless explicitly permitted.

## 60.2 Symlink safety

Validate symlinks before destructive operations.

## 60.3 Archive safety

Validate:

-   archive paths;
-   expected structure;
-   checksum;
-   signature;
-   release metadata.

## 60.4 Process execution

Use structured execution APIs.

Avoid concatenating untrusted shell strings.

## 60.5 Destructive operations

Require explicit confirmation.

------------------------------------------------------------------------

# 61. Transactionality

Installation and update should use:

``` text
resolve
↓
verify
↓
stage
↓
validate staged state
↓
commit
```

Failure before commit must leave the project unchanged where practical.

Recovery/rollback must be deterministic.

------------------------------------------------------------------------

# 62. Idempotency

The following should be idempotent:

``` text
status
inspect
validate
doctor
audit
```

`init` must detect existing installation.

`update` must detect an already-installed requested version.

------------------------------------------------------------------------

# 63. Network and Offline Operation

The CLI must support:

``` text
online resolution
local cache
local Bootstrap path
offline mode
```

Offline mode must fail clearly if no compatible verified Bootstrap
exists.

It must never silently choose an unknown version.

------------------------------------------------------------------------

# 64. CLI Does Not Need an LLM

The CLI itself must not require an LLM.

The CLI must be capable of:

``` text
status
inspect
validate
doctor
audit
update
export
clean
version
```

without an LLM.

ADE-assisted initialization and guided design require an ADE.

------------------------------------------------------------------------

# 65. ADE Invocation Abstraction

Conceptual Go interface:

``` go
type ADEExecutor interface {
    ID() string
    Available() bool
    BuildCommand(contract ExecutionContract) Command
    Execute(ctx context.Context, contract ExecutionContract) error
}
```

The adapter knows:

-   detection;
-   invocation;
-   command construction;
-   exit status;
-   platform integration.

It must not know GTT methodology.

------------------------------------------------------------------------

# 66. Bootstrap Adapter Boundary

Conceptual interface:

``` go
type BootstrapRuntime interface {
    Resolve(ctx context.Context, request ResolveRequest) (Package, error)
    Verify(ctx context.Context, pkg Package) error
    Manifest(ctx context.Context) (Manifest, error)
    Compatibility(ctx context.Context) (Compatibility, error)
    Execute(ctx context.Context, operation OperationRequest) (OperationResult, error)
}
```

The CLI may also have specialized adapters for:

``` text
ADE
filesystem
Git
process execution
registry
snapshot
export
```

The interfaces are operational abstractions.

They are not a Go reimplementation of Bootstrap.

------------------------------------------------------------------------

# 67. Bootstrap Operation Contract

The CLI may call generic Bootstrap operations such as:

``` text
ade.detect
ade.state
ade.install
ade.validate

template.list
template.show
template.materialize

status
validate
freeze
session-context
index
reconcile
query
guard-sync
provenance-check
export-policy
recovery
```

The operation identifiers and schemas must be supplied by Bootstrap.

The CLI must not invent semantic behavior when an operation is unknown.

------------------------------------------------------------------------

# 68. Unknown Capability Rule

If Bootstrap declares a capability the CLI does not support:

``` text
REFUSE SAFELY
```

Do not:

``` text
skip
guess
fallback silently
execute arbitrary shell
```

Example:

``` text
Bootstrap requires capability:
methodology-profile.v2

Current CLI supports:
methodology-profile.v1

No changes were made.
Upgrade the CLI.
```

------------------------------------------------------------------------

# 69. Capability Negotiation

Before modification:

``` text
CLI capabilities
        +
Bootstrap requirements
        ↓
capability negotiation
        ↓
PASS / REFUSE
```

This applies to:

-   Bootstrap schema;
-   ADE integration;
-   methodology profile contract;
-   recovery;
-   update migration;
-   export;
-   operation contracts.

------------------------------------------------------------------------

# 70. No Recipe Dependency

The CLI must not require:

``` text
GTT-BOOTSTRAP-RECIPE.md
```

or any other specific recipe file in order to operate.

The Bootstrap may internally use recipes in a future implementation.

If it does, that is a Bootstrap implementation detail.

The CLI consumes stable contracts/capabilities.

This is a critical v1.0 architectural invariant.

------------------------------------------------------------------------

# 71. Bootstrap Evolution Without CLI Source Changes

The desired evolution model is:

``` text
Bootstrap changes
       ↓
contract-compatible capability
       ↓
same CLI
```

Examples:

-   questionnaire content changes;
-   questionnaire structure changes;
-   new source template;
-   new ADE overlay;
-   Bootstrap script implementation changes;
-   governance documentation changes;
-   profile semantics change while contract remains compatible.

These should not require CLI source changes.

A CLI update is required only when the Bootstrap requires a new runtime
primitive/capability that the existing CLI does not support.

------------------------------------------------------------------------

# 72. Example of Safe Evolution

Tomorrow Bootstrap adds:

``` text
template.materialize architecture-map
```

If the CLI already supports generic:

``` text
template.materialize
```

the CLI does not change.

If Bootstrap adds:

``` text
new-runtime-capability-X
```

and CLI 1.0 does not support it:

``` text
REFUSE
```

The CLI does not guess how to implement X.

------------------------------------------------------------------------

# 73. No Arbitrary Shell Recipe Execution

The CLI must not execute arbitrary shell strings supplied by a project
document.

Allowed execution comes from:

-   verified Bootstrap package;
-   declared Bootstrap operation;
-   trusted ADE adapter;
-   explicit CLI operation.

The command execution layer must distinguish:

``` text
trusted executable
arguments
environment
working directory
```

------------------------------------------------------------------------

# 74. CLI Repository

Recommended:

``` text
gtt-cli/
├── cmd/
│   └── gtt/
│       └── main.go
├── internal/
│   ├── bootstrap/
│   │   ├── resolver/
│   │   ├── installer/
│   │   ├── manifest/
│   │   ├── compatibility/
│   │   ├── capabilities/
│   │   └── verification/
│   ├── ade/
│   │   ├── detector/
│   │   ├── primary/
│   │   └── executor/
│   ├── project/
│   │   ├── detector/
│   │   └── documents/
│   ├── methodology/
│   ├── lifecycle/
│   ├── execution/
│   ├── session/
│   ├── snapshot/
│   ├── export/
│   ├── update/
│   ├── recovery/
│   └── output/
├── test/
│   ├── integration/
│   ├── e2e/
│   └── fixtures/
├── go.mod
├── go.sum
├── README.md
└── LICENSE
```

There must be no:

``` text
internal/governance/
internal/architecture/
internal/adr/
internal/reasoning/
internal/think/
internal/gtt-rules/
```

unless a package is purely an operational adapter.

------------------------------------------------------------------------

# 75. Dependency Direction

Preferred:

``` text
cmd
 ↓
application/use cases
 ↓
operational services
 ↓
interfaces
 ↓
adapters
 ↓
Bootstrap / Git / filesystem / ADE / registry
```

Bootstrap-specific details stay behind interfaces.

Avoid circular dependencies.

------------------------------------------------------------------------

# 76. Technology

Implementation:

``` text
Go
Cobra
```

The CLI must not require:

``` text
Python runtime
Node.js runtime
LLM
```

Bootstrap may internally use Bash/Python or other technologies.

------------------------------------------------------------------------

# 77. Output Modes

Operational commands should support:

``` text
human-readable
--json
```

where defined.

Recommended:

``` text
gtt status --json
gtt inspect --json
gtt validate --json
gtt doctor --json
gtt audit --json
gtt version --json
```

JSON schemas must be versioned.

------------------------------------------------------------------------

# 78. Exit Codes

The v1.0 exit contract:

``` text
0   success
1   validation / operational failure
2   CLI/Bootstrap incompatibility
3   invalid invocation
4   user cancellation
5   project state conflict
6   Bootstrap integrity failure
7   external dependency unavailable
```

The implementation must use these consistently.

------------------------------------------------------------------------

# 79. Error Design

Every user-facing error should answer:

1.  What happened?
2.  Why?
3.  What can the user do?

Example:

``` text
GTT Bootstrap installation refused.

Reason:
Bootstrap 1.0 requires methodology-profile.v1.
The installed CLI does not support that contract.

No project files were modified.

Next step:
Upgrade GTT CLI.
```

------------------------------------------------------------------------

# 80. Logging

Internal logging:

``` text
error
warn
info
debug
```

User output remains concise.

Debug logs must never leak:

-   tokens;
-   credentials;
-   secrets;
-   private environment variables.

------------------------------------------------------------------------

# 81. Configuration

Global configuration may contain:

``` text
Bootstrap registry
cache location
default language
network behavior
release channel
```

Project state belongs to the project when appropriate.

No global configuration may silently override governed state.

------------------------------------------------------------------------

# 82. Release Channels

Recommended:

``` text
stable
preview
development
```

Production defaults to:

``` text
stable
```

A project records the resolved exact Bootstrap release.

------------------------------------------------------------------------

# 83. Version Independence

CLI and Bootstrap versions are independent.

Example:

``` text
GTT CLI       1.0.0
Bootstrap     1.0.0
Schema        1
```

Compatibility is explicit.

The CLI must not assume equal version numbers.

------------------------------------------------------------------------

# 84. Backward Compatibility

The CLI must support only a documented Bootstrap compatibility range.

Example:

``` text
CLI 1.0 + Bootstrap 1.0 → PASS
CLI 1.0 + unsupported Bootstrap → REFUSE
```

The exact supported range is part of release metadata.

------------------------------------------------------------------------

# 85. CI/CD

Primary CI command:

``` text
gtt validate --ci
```

Potential:

``` text
gtt status --json
gtt audit --json
gtt doctor --json
```

CI can detect:

-   invalid GTT structure;
-   incompatible Bootstrap;
-   invalid ADE integration;
-   protection failures;
-   provenance failures;
-   stale indexes;
-   failed Bootstrap validation.

The CLI does not interpret architectural correctness.

------------------------------------------------------------------------

# 86. Deterministic Session Handoff

Session context must be derived from actual state.

Safe:

``` text
Git
+
GTT artifacts
+
Bootstrap state
+
proposals
+
change request
+
freeze
+
validation
    ↓
derived session context
```

Unsafe:

``` text
agent says what happened
    ↓
CLI writes it as truth
```

The CLI must never use free-form agent prose as authoritative state.

------------------------------------------------------------------------

# 87. Project State Machine

CLI operational state may be modeled as:

``` text
UNINITIALIZED
    ↓
DETECTED
    ↓
CONFIGURING
    ↓
RESOLVING_BOOTSTRAP
    ↓
VERIFYING_BOOTSTRAP
    ↓
STAGING
    ↓
INSTALLING
    ↓
VALIDATING
    ↓
INITIALIZED
    ↓
ACTIVE
    ↓
FROZEN
    ↓
CHANGE_PENDING
    ↓
UPDATING
```

This is operational only.

It does not replace Bootstrap governance state.

------------------------------------------------------------------------

# 88. Init Transaction

`gtt init` should behave transactionally:

``` text
discover
↓
resolve
↓
verify
↓
collect selections
↓
stage
↓
install
↓
validate
↓
commit
```

If the transaction cannot safely complete:

``` text
rollback / restore
```

The CLI must not leave a misleading half-installed project.

------------------------------------------------------------------------

# 89. Resume After Interrupted Init

If an interrupted initialization is detected:

``` text
Incomplete GTT initialization detected.

State:
STAGING

Options:
[1] Resume
[2] Roll back
[3] Inspect
[4] Abort
```

The CLI must use recorded transaction state.

It must not infer completion from directory existence alone.

------------------------------------------------------------------------

# 90. Reconfiguration

Operational settings may include:

``` text
Primary ADE
participating ADEs
language
methodology profile
Bootstrap release/channel
```

Every reconfiguration must:

1.  load current state;
2.  validate requested change;
3.  ask Bootstrap whether migration/governance is required;
4.  stage;
5.  apply;
6.  validate;
7.  report.

------------------------------------------------------------------------

# 91. Primary ADE and Secondary ADEs

The final state model is:

``` text
GTT Governance
      │
      ├── Primary ADE
      │
      ├── Secondary ADE
      │
      └── Secondary ADE
```

All participating ADEs receive their declared integration.

The Primary ADE does not outrank them in governance.

------------------------------------------------------------------------

# 92. Existing Project Safety

GTT must not reorganize the host application.

The CLI must not move:

``` text
docs/
context/
adr/
proposals/
```

merely because those names exist.

GTT's own namespaced directories are:

``` text
.gtt/
gtt-domain/
```

A collision with those namespaces is a conflict to report.

------------------------------------------------------------------------

# 93. Bootstrap Manifest Usage

The current Bootstrap scaffold manifest defines:

``` text
engine
domain
entry_points
overlays
templates
```

The CLI may use it to understand installation structure.

It must distinguish:

``` text
scaffold version
```

from:

``` text
Bootstrap release version
```

The manifest itself is not a replacement for release metadata.

------------------------------------------------------------------------

# 94. ADE Registry Usage

The Bootstrap manifest is the ADE registry.

The CLI consumes:

``` text
id
name
path
entry
owned
detect
enforcement
scaffold
```

This enables the CLI to remain ADE-extensible without embedding an ADE
catalog.

------------------------------------------------------------------------

# 95. Clean Ownership

The Bootstrap defines which ADE paths GTT owns.

The CLI must use that ownership information for:

``` text
clean
export --clean
update
rollback
```

The CLI must never delete an ADE file merely because it resembles a GTT
file.

------------------------------------------------------------------------

# 96. Template Service

The Bootstrap's template service is operationally important.

The CLI may request:

``` text
list
show
materialize
```

The current implementation is:

``` text
.gtt/scripts/gtt-template.sh
```

The CLI should call the declared operation rather than permanently
coupling to the script filename.

------------------------------------------------------------------------

# 97. Validation Service

The Bootstrap's validation service is:

``` text
.gtt/scripts/gtt-validate.sh
```

The CLI should invoke it through the Bootstrap contract.

The CLI must not duplicate:

``` text
gtt-check-integrity
gtt-check-provenance
gtt-check-protection
gtt-check-backlog
gtt-check-adapter
...
```

semantics.

------------------------------------------------------------------------

# 98. Status Service

The Bootstrap's status service derives operational state.

Current implementation:

``` text
.gtt/scripts/gtt-status.sh
```

The CLI should consume the resulting structured state.

The CLI may format it for humans or JSON.

------------------------------------------------------------------------

# 99. Freeze Service

The Bootstrap owns freeze.

Current implementation:

``` text
.gtt/scripts/gtt-freeze.sh
```

The CLI is only the facade.

------------------------------------------------------------------------

# 100. Query and Retrieval

The Bootstrap owns technical retrieval/query services.

Current implementation includes:

``` text
gtt-query.sh
gtt-index.sh
gtt-reconcile.sh
```

The CLI may expose operational inspection/reporting around them.

It must not implement a separate RAG/reasoning system.

------------------------------------------------------------------------

# 101. GTTGuard Boundary

The CLI must understand the existence of protection as an operational
capability.

It must not decide:

``` text
this source code should be protected
```

unless the human explicitly requests the Bootstrap protection workflow.

------------------------------------------------------------------------

# 102. Source Authority Boundary

The CLI may record:

``` text
selected source
```

It must never record:

``` text
source_is_truth = true
```

unless that is merely an opaque Bootstrap-defined state field and the
semantic decision was made through the governed process.

------------------------------------------------------------------------

# 103. Backlog Boundary

The CLI may report backlog presence/status.

It must not invent Epics or Stories.

Bootstrap governs:

``` text
backlog precedence
change process
reconciliation
validation
```

The CLI is operational only.

------------------------------------------------------------------------

# 104. Working Agreements Boundary

The CLI may expose Bootstrap template materialization for working
agreements.

It must not turn preferences into governance.

------------------------------------------------------------------------

# 105. Human Confirmation Points in `init`

The CLI must require explicit human confirmation for decisions such as:

``` text
project root when ambiguous
participating ADEs
Primary ADE
initial source documents
questionnaire creation
methodology profile
destructive operations
```

The exact additional confirmations are supplied by Bootstrap contracts.

------------------------------------------------------------------------

# 106. `init` UX Example

``` text
$ gtt init

GTT Project Initialization

Project:
  payments-api

Bootstrap:
  Resolving GTT Bootstrap...
  Version: 1.0.0
  Integrity: PASS
  Compatibility: PASS

Detected ADE candidates:
  [1] Claude Code
  [2] GitHub Copilot
  [3] Codex

Participating ADEs:
  Select: 1,2

Primary ADE:
  Select: 1

Language:
  [1] English
  [2] Español
  Select: 2

Initial design sources:
  [1] docs/requirements.md
  [2] docs/architecture.md
  Select: 1,2

Methodology application:
  [1] Light
  [2] Medium
  [3] Hard
  Select: 2

Summary:

  Primary ADE: Claude Code
  Participating: Claude Code, Copilot
  Sources: 2
  Language: Español
  Methodology: Medium

Proceed? [Y/n]

Installing GTT Core...
Installing ADE integrations...
Validating...
Generating Bootstrap handoff...

GTT initialization complete.

Next:
Launch Claude Code to continue the Bootstrap-guided initialization.
```

------------------------------------------------------------------------

# 107. No-Document UX

``` text
No sufficient design source document was detected.

GTT Bootstrap provides an Initial Design Questionnaire.

This questionnaire is owned by Bootstrap and will be materialized
as project source material.

Create questionnaire? [Y/n]
```

After creation:

``` text
Questionnaire created:
gtt-domain/proposals/bootstrap/initial-design-questionnaire.md

Next:
Launch the Primary ADE to conduct the Bootstrap-guided discovery.
```

------------------------------------------------------------------------

# 108. Methodology Profile UX

``` text
GTT Methodology Application

Choose the project's governance intensity:

1. Light
   POCs, experiments, spikes and disposable work.

2. Medium
   Normal development and controlled project work.

3. Hard
   Production or high-consequence work.

Selection: 2

Bootstrap will apply the semantics of the selected profile.
```

The CLI must not explain internal promotion rules unless Bootstrap
supplies them as user-facing documentation.

------------------------------------------------------------------------

# 109. `status` Example

``` text
GTT Project Status

Project:
  payments-api

CLI:
  1.0.0

Bootstrap:
  1.0.0
  Schema: 1
  Compatibility: PASS

ADE:
  Primary: Claude Code
  Participating: Claude Code, Copilot
  Excluded: Kiro, Codex

Methodology:
  Medium

Sources:
  2 initial sources

Governance:
  Freeze: ACTIVE
  OPEN: 2
  BLOCKING: 0
  Proposals: 3

Validation:
  PASS

Session:
  CURRENT
```

OPEN/BLOCKING classification comes from Bootstrap.

------------------------------------------------------------------------

# 110. `doctor` Example

``` text
$ gtt doctor

[PASS] CLI
[PASS] Project root
[PASS] Git
[PASS] Bootstrap package
[PASS] Bootstrap integrity
[PASS] CLI compatibility
[PASS] ADE state
[PASS] Primary ADE
[PASS] Methodology profile contract
[PASS] GTT scaffold
[PASS] Bootstrap validation
[WARN] Session state stale

Result:
11 PASS
1 WARN
0 FAIL
```

------------------------------------------------------------------------

# 111. `update` Example

``` text
$ gtt update

Current Bootstrap:
  1.0.0

Target:
  1.1.0

Checking compatibility...
PASS

Checking migration...
PASS

Creating recovery snapshot...
PASS

Staging update...
PASS

Validating...
PASS

Bootstrap updated successfully.
```

------------------------------------------------------------------------

# 112. `clean` Example

``` text
$ gtt clean

WARNING

This operation removes GTT from this project.

Primary ADE:
  Claude Code

Bootstrap:
  1.0.0

A recovery snapshot is recommended.

Create snapshot first? [Y/n]

Continue with destructive clean? [y/N]
```

No implicit deletion.

------------------------------------------------------------------------

# 113. Test Strategy

Testing must be layered.

## Unit

Test:

-   version resolution;
-   compatibility;
-   release verification;
-   manifest parsing;
-   capability negotiation;
-   ADE detection;
-   ADE selection;
-   source discovery;
-   source selection;
-   methodology profile selection;
-   path safety;
-   transaction state;
-   snapshot metadata;
-   output;
-   exit codes.

## Integration

Use real Bootstrap fixtures.

## End-to-end

Run complete workflows.

------------------------------------------------------------------------

# 114. ADE Test Matrix

  Detected                   Participating      Primary   Expected
  -------------------------- ------------------ --------- ----------------
  Claude                     Claude             Claude    Core + Claude
  Copilot                    Copilot            Copilot   Core + Copilot
  Claude + Copilot           Claude + Copilot   Claude    Core + both
  Claude + Codex             Claude + Codex     Claude    Core + both
  Claude + Copilot + Codex   all                Claude    Core + all
  Multiple                   all selected       Copilot   Core + all

The test must verify that detection does not equal participation.

------------------------------------------------------------------------

# 115. Initialization Test Matrix

## One document

``` text
detect
→ display
→ confirm
→ install
```

## Multiple documents

``` text
detect
→ display
→ explicit selection
→ install
```

## No document

``` text
detect none
→ offer Bootstrap questionnaire
→ materialize
→ handoff
```

## User declines questionnaire

``` text
do not pretend design exists
return safely
```

## Methodology profile

Test:

``` text
Light
Medium
Hard
```

and verify the CLI passes the selected profile to Bootstrap without
applying methodology logic itself.

------------------------------------------------------------------------

# 116. Update Test Matrix

Test:

``` text
same version
compatible upgrade
unsupported version
incompatible CLI
incompatible Bootstrap
incompatible profile contract
failed integrity
failed installation
failed validation
interrupted update
rollback
```

Expected:

``` text
no unsafe partial update
```

------------------------------------------------------------------------

# 117. Recovery Test Matrix

Test:

``` text
create snapshot
validate snapshot
clean GTT
detect snapshot
resolve Bootstrap
restore
validate restored installation
```

Application source must remain untouched by the recovery process except
where explicitly declared by the Bootstrap contract.

------------------------------------------------------------------------

# 118. Clean Export Test Matrix

Verify:

``` text
development project unchanged
GTT excluded
GTT-owned ADE overlays excluded
application source preserved
tests preserved
deployment preserved
destination created
no governance artifacts in delivery
```

The exact exclusion list comes from Bootstrap.

------------------------------------------------------------------------

# 119. Bootstrap Evolution Test

Mandatory acceptance test:

``` text
1. Install CLI 1.0.
2. Install Bootstrap 1.0.
3. Add/change a Bootstrap-owned template or compatible contract.
4. Do not modify CLI source.
5. Run the CLI.
6. CLI consumes the changed Bootstrap capability.
7. CLI succeeds.
```

This proves the CLI is not hard-coded to methodology content.

------------------------------------------------------------------------

# 120. Unsupported Capability Test

Mandatory:

``` text
Bootstrap requires capability X.
CLI does not support X.
```

Expected:

``` text
REFUSE
NO MODIFICATION
ACTIONABLE ERROR
```

Never skip or guess.

------------------------------------------------------------------------

# 121. Security Test Matrix

Test:

``` text
path traversal
absolute path injection
symlink escape
malicious archive
checksum mismatch
signature mismatch
unsafe process argument
destructive operation without confirmation
untrusted recipe/script content
```

The CLI must fail closed.

------------------------------------------------------------------------

# 122. No-LLM Test

Verify that these work without an LLM:

``` text
status
inspect
validate
doctor
audit
update
export
clean
version
```

ADE-assisted flows may require an ADE.

------------------------------------------------------------------------

# 123. CLI Architecture Review

The implementation must answer:

### Can Bootstrap methodology change without CLI source changes?

``` text
YES
```

provided the existing contract/capability surface remains compatible.

### Can Bootstrap change questionnaire content without CLI source changes?

``` text
YES
```

### Can Bootstrap add an ADE integration without embedding that ADE's methodology in Go?

``` text
YES
```

### Can the CLI support multiple ADEs?

``` text
YES
```

### Can the user select exactly one Primary ADE?

``` text
YES
```

### Can the project start without a design document?

``` text
YES
```

through Bootstrap's questionnaire.

### Can the user choose Light/Medium/Hard?

``` text
YES
```

### Does the CLI interpret Light/Medium/Hard?

``` text
NO
```

### Can clean QA/PRO artifacts be generated without destroying development?

``` text
YES
```

### Can GTT be removed with portable recovery?

``` text
YES
```

### Can the CLI operate without an LLM?

``` text
YES
```

### Does the CLI contain GTT methodology?

``` text
NO
```

------------------------------------------------------------------------

# 124. Implementation Phases

## Phase 1 --- Foundation

``` text
Go module
Cobra
configuration
logging
output
exit codes
process executor
project-root detection
```

## Phase 2 --- Bootstrap Runtime

``` text
Bootstrap resolver
release metadata
integrity verification
manifest
compatibility
capability negotiation
cache
installer
transaction engine
```

## Phase 3 --- Project Initialization

``` text
project detection
existing GTT detection
ADE candidate discovery
source discovery
source selection
language
methodology profile
questionnaire fallback
```

## Phase 4 --- ADE

``` text
participation
Primary ADE
secondary ADEs
installation
validation
invocation
handoff
```

## Phase 5 --- Lifecycle

``` text
status
inspect
validate
resume
freeze
doctor
audit
```

## Phase 6 --- Distribution

``` text
update
snapshot
restore
export --clean
clean
```

## Phase 7 --- Hardening

``` text
security
transactionality
CI
JSON
integration tests
E2E
release
documentation
```

------------------------------------------------------------------------

# 125. First Repository Commit

Start with:

``` text
gtt-cli/
├── cmd/gtt/main.go
├── internal/
│   ├── output/
│   ├── execution/
│   └── project/
├── go.mod
├── go.sum
└── README.md
```

Then add Bootstrap integration.

Do not create placeholder governance packages.

------------------------------------------------------------------------

# 126. Initial Cobra Tree

``` text
gtt
├── init
├── status
├── inspect
├── validate
├── resume
├── freeze
├── doctor
├── audit
├── update
├── export
│   └── --clean
├── clean
└── version
```

Optional operational configuration:

``` text
gtt config
├── primary-ade
├── methodology
└── language
```

Only implement `config` subcommands when the corresponding Bootstrap
contract exists.

------------------------------------------------------------------------

# 127. Definition of Done --- CLI Core

-   [ ] Go project builds reproducibly.
-   [ ] Cobra tree exists.
-   [ ] Project root detection works.
-   [ ] Bootstrap 1.0 can be resolved.
-   [ ] Bootstrap release identity is verified.
-   [ ] Bootstrap checksum/signature is verified when required.
-   [ ] Bootstrap manifest is parsed.
-   [ ] CLI/Bootstrap compatibility is enforced.
-   [ ] Capability negotiation exists.
-   [ ] Bootstrap cache exists.
-   [ ] Offline/local Bootstrap resolution works.
-   [ ] Existing GTT is detected.
-   [ ] ADE candidates are detected.
-   [ ] Participating ADEs can be selected.
-   [ ] Exactly one Primary ADE can be selected.
-   [ ] Multiple ADEs can be installed.
-   [ ] Language can be selected.
-   [ ] Initial sources can be selected.
-   [ ] No-document questionnaire path works.
-   [ ] Questionnaire is never embedded in CLI source.
-   [ ] Methodology profile can be selected.
-   [ ] Profile is passed to Bootstrap.
-   [ ] Profile semantics are not duplicated in CLI.
-   [ ] Installation is transactional.
-   [ ] Bootstrap validation is invoked.
-   [ ] ADE handoff works.
-   [ ] Primary ADE invocation works where supported.
-   [ ] `status` works.
-   [ ] `inspect` works.
-   [ ] `validate` works.
-   [ ] `resume` works.
-   [ ] `freeze` delegates to Bootstrap.
-   [ ] `doctor` works.
-   [ ] `audit` works.
-   [ ] `update` works with strict compatibility.
-   [ ] `export --clean` works.
-   [ ] `clean` is protected.
-   [ ] Recovery snapshot works.
-   [ ] Recovery is tested.
-   [ ] CI mode works.
-   [ ] JSON output exists for defined commands.
-   [ ] Multi-ADE integration tests pass.
-   [ ] No GTT methodology is duplicated in Go.

------------------------------------------------------------------------

# 128. Definition of Done --- Bootstrap Boundary

-   [ ] CLI does not depend on a specific recipe file.
-   [ ] CLI consumes Bootstrap contracts.
-   [ ] Bootstrap release identity is machine-readable.
-   [ ] Bootstrap capability schema is versioned.
-   [ ] ADE registry is Bootstrap-owned.
-   [ ] Questionnaire is Bootstrap-owned.
-   [ ] Template evolution does not require CLI changes.
-   [ ] Bootstrap defines methodology profile semantics.
-   [ ] Light/Medium/Hard contract exists.
-   [ ] CLI only selects and passes the profile.
-   [ ] Bootstrap owns proposal/freeze/governance semantics.
-   [ ] Bootstrap owns validation semantics.
-   [ ] Bootstrap owns clean/export ownership semantics.
-   [ ] Bootstrap owns recovery semantics.
-   [ ] Bootstrap owns session semantics.
-   [ ] CLI remains operational.

------------------------------------------------------------------------

# 129. Definition of Done --- Evolution Test

The definitive architecture test is:

``` text
Can the Bootstrap evolve its methodology/content/contracts
without forcing a CLI source change whenever the change remains
within the CLI's existing operational capability set?
```

Expected:

``` text
YES
```

Second test:

``` text
Can Bootstrap require a new runtime primitive?
```

Expected:

``` text
YES
```

But then:

``` text
CLI detects unsupported capability
→ refuses safely
→ requests compatible CLI
```

------------------------------------------------------------------------

# 130. Final Architectural Contract

``` text
                         GTT METHOD
                              │
                              ▼
                     GTT BOOTSTRAP 1.0
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
   Methodology           Engine/Domain            ADEs
   Governance            Services                Templates
   Evidence              Validation              Contracts
   Profiles              State                   Questionnaire
        │                     │                     │
        └─────────────────────┼─────────────────────┘
                              │
                     explicit contracts
                              │
                              ▼
                         GTT CLI 1.0
                              │
       ┌──────────────┬───────┼────────┬──────────────┐
       │              │       │        │              │
     Init           Status  Validate  Update        Export
       │
       ├── Project discovery
       ├── ADE detection
       ├── ADE participation
       ├── Primary ADE
       ├── Source selection
       ├── Language
       ├── Methodology profile selection
       ├── Questionnaire orchestration
       ├── Bootstrap installation
       └── ADE handoff
                              │
                              ▼
                         Primary ADE
                              │
                              ▼
                            Human
                              │
                              ▼
                           Freeze
```

------------------------------------------------------------------------

# 131. Final Rule for Future Developers

When deciding whether logic belongs in CLI:

``` text
Does it explain how GTT thinks,
governs, reasons, evaluates evidence,
designs architecture, interprets proposals,
or decides methodology?

        YES
        ↓
    BOOTSTRAP

Does it determine how to discover,
resolve, install, select, invoke,
validate, update, export, clean,
recover, or operate Bootstrap?

        YES
        ↓
       CLI
```

For the methodology profiles:

``` text
CLI
  → asks
  → records
  → passes

Bootstrap
  → defines
  → interprets
  → enforces
```

For questionnaire:

``` text
Bootstrap
  → owns
  → evolves
  → defines

CLI
  → detects need
  → requests
  → materializes
  → reports
```

For ADEs:

``` text
Bootstrap
  → defines registry and integration contracts

CLI
  → detects
  → asks human
  → installs
  → selects Primary
  → invokes
```

For governance:

``` text
Bootstrap
  → owns semantics

CLI
  → exposes operations
```

The boundary must survive the entire GTT CLI v1.x lifecycle.

------------------------------------------------------------------------

# 132. Final Engineering Position

The final GTT architecture is intentionally asymmetric:

``` text
Bootstrap = intelligence + methodology + governance + implementation
CLI       = operational runtime + lifecycle + distribution + UX
ADE       = AI execution environment
Human     = decision authority
```

The CLI is not a miniature GTT.

It is the **professional operational doorway into GTT Bootstrap**.

The Bootstrap is not a CLI dependency on a recipe.

It is a **versioned implementation and contract provider**.

The CLI must therefore be capable of operating against evolving
Bootstrap implementations without becoming coupled to their internal
documents, prompts, questionnaire text, scripts, or methodology prose.

That is the final architectural boundary for GTT CLI v1.0.


---

# 133. Bootstrap 1.0 Contract Dependency — Applied

The requirements identified during the Bootstrap/CLI reconciliation are now explicitly assigned to **Bootstrap 1.0**.

The CLI v1.0 implementation must treat these as external Bootstrap contracts:

```text
Bootstrap 1.0
├── release identity
├── compatibility
├── capabilities
├── operations
├── ADE registry
├── Primary ADE semantics
├── questionnaire
├── source authority
├── methodology profiles
│   ├── Light
│   ├── Medium
│   └── Hard
├── governance semantics
├── validation
├── freeze
├── GTTGuard
├── session
├── export policy
└── recovery
```

The CLI provides:

```text
GTT CLI 1.0
├── resolve
├── verify
├── install
├── detect
├── select
├── invoke
├── lifecycle
├── update
├── validate facade
├── export
├── clean
├── recovery
└── UX
```

Therefore the CLI implementation must **not** add local methodology rules to compensate for Bootstrap behavior.

If Bootstrap 1.0 has not yet physically implemented one of the required contracts, that is a **Bootstrap implementation task**, not a reason to move the behavior into the CLI.

---

# 134. Bootstrap Task Reference

The implementation work required on Bootstrap is defined in:

```text
GTT_BOOTSTRAP_1.0_CLI_CONTRACT_IMPLEMENTATION_TASK.md
```

The task establishes the missing machine-readable contracts required for the CLI.

The CLI baseline therefore remains stable while Bootstrap evolves toward the required contract surface.

---

# 135. Final CLI/Bootstrap Acceptance Boundary

The complete system passes architecture review when:

```text
Bootstrap 1.0
    ↓
exposes required contracts
    ↓
CLI 1.0
    ↓
consumes contracts
    ↓
Human selects operational choices
    ↓
Bootstrap applies GTT semantics
```

and never:

```text
CLI
    ↓
reimplements GTT methodology
```

This is the final v1.0 boundary.

---

# PART III — Consolidated Final Boundary

The following rule is the final architectural test for all implementation work:

```text
Does this logic define how GTT thinks, governs, reasons, evaluates evidence,
designs architecture, interprets context, defines methodology, or makes
GTT semantic decisions?

        YES
        ↓
    GTT Bootstrap 1.0

Does this logic determine how to discover, install, select, invoke, validate,
update, export, recover, or operate the Bootstrap?

        YES
        ↓
       GTT CLI 1.0
```

The CLI must never become a second GTT engine. If a capability is missing from Bootstrap and belongs semantically to GTT, the implementation must extend Bootstrap 1.0 and expose a versioned contract; the CLI then consumes that contract.

## Final Development Rule

A developer must be able to start from zero with this document alone and implement the CLI operationally while treating Bootstrap 1.0 as the semantic implementation boundary. No separate interpretation or manual reconciliation of earlier CLI specifications is required.
