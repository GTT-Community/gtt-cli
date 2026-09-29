# GTT CLI v1.0 --- Technical Work Specification

> **GTT Governance Canonical:**
> https://github.com/GTT-Community/gtt-method/blob/main/GTT-CANONICAL-v2.1.md

> **Document status:** Implementation Work Specification\
> **Target:** GTT CLI v1.0\
> **Language:** English\
> **Implementation:** Go + Cobra\
> **Architecture principle:** The intelligence and governance live in
> GTT Bootstrap. The CLI is the operational surface.

------------------------------------------------------------------------

## 1. Purpose

This document is the implementation-grade specification for building the
**GTT CLI v1.0 from zero**.

The CLI is not a second GTT engine.

It is a professional operational tool responsible for:

-   detecting and initializing a project;
-   resolving and installing a compatible GTT Bootstrap;
-   detecting and integrating all represented ADEs;
-   establishing one user-selected Primary ADE;
-   collecting the initial project/design source material;
-   detecting when no sufficient design source exists and requesting the
    Bootstrap-provided initial design questionnaire;
-   delivering a precise bootstrap contract to the Primary ADE;
-   invoking existing Bootstrap capabilities instead of reimplementing
    them;
-   exposing project/GTT operational state;
-   validating installation and project integrity;
-   resuming work from deterministic project state;
-   freezing through the Bootstrap contract;
-   maintaining CLI/Bootstrap compatibility;
-   producing portable GTT recovery snapshots;
-   generating clean QA/PRO delivery artifacts without destroying the
    governed development project.

The CLI must remain intentionally small.

The central design rule is:

``` text
GTT Method
    ↓
GTT Bootstrap
    ↓
Engine + Domain + ADE integrations + Governance contracts
    ↓
GTT CLI
    ↓
Operational lifecycle / installation / orchestration / distribution
```

The CLI must **consume and orchestrate the Bootstrap**. It must not
reproduce its methodology.

------------------------------------------------------------------------

# 2. Design Principles

## 2.1 Bootstrap is the source of truth

The GTT Bootstrap contains the implementation of GTT.

The CLI must not become an alternative implementation of:

-   governance;
-   evidence rules;
-   grounding;
-   dossier construction;
-   architecture reasoning;
-   ADR semantics;
-   proposal semantics;
-   freeze semantics;
-   agent reasoning;
-   GTT methodology;
-   architectural policy;
-   context interpretation.

If a rule belongs to GTT itself, it belongs in Bootstrap.

The CLI may only contain operational rules required to safely:

-   discover the project;
-   install Bootstrap;
-   resolve versions;
-   detect ADEs;
-   select Primary ADE;
-   select initial sources;
-   preserve installation integrity;
-   execute Bootstrap contracts;
-   update Bootstrap;
-   export/clean project state;
-   recover a previous GTT installation.

### Non-goal

The CLI must never evolve into:

``` text
gtt architecture
gtt adr
gtt proposal
gtt think
gtt reasoning
gtt governance-engine
```

Those concepts are implemented by the Bootstrap and its governed domain.

------------------------------------------------------------------------

# 3. Architectural Boundary

## 3.1 Responsibility split

  Responsibility                                         GTT Bootstrap                          GTT CLI
  ---------------------------- --------------------------------------- --------------------------------
  GTT methodology                                                  YES                               NO
  Governance rules                                                 YES                               NO
  Evidence boundary                                                YES                               NO
  Grounding                                                        YES                               NO
  Dossier                                                          YES                               NO
  Architecture/context rules                                       YES                               NO
  ADR semantics                                                    YES                               NO
  Proposal semantics                                               YES                               NO
  Freeze semantics                                                 YES                               NO
  Agent contracts                                                  YES                               NO
  ADE-specific integration                                         YES       Detect/install/orchestrate
  Project detection                                  Contract consumed                              YES
  Bootstrap resolution                                 Contract source                              YES
  Version compatibility                  Contract source + enforcement                              YES
  Installation                                          Content source                              YES
  Primary ADE selection                                 State/contract                              YES
  Initial document selection                               Data source                              YES
  Questionnaire generation                             Template source   Detect/materialize/orchestrate
  Session state derivation                     Bootstrap scripts/state                     Orchestrates
  Status                         Bootstrap evidence + CLI presentation                              YES
  Update                                       Bootstrap package + CLI                              YES
  Clean export                            Bootstrap exclusion contract                              YES
  Destructive clean                             Installation lifecycle                              YES
  Recovery snapshot                                       State source                              YES

The boundary is mandatory.

------------------------------------------------------------------------

# 4. Existing Bootstrap Reality

The CLI must be designed around the existing Bootstrap rather than
inventing a new internal implementation.

The current Bootstrap architecture includes the concept of:

``` text
.gtt/
    scripts/
    index/
    protection/
    session-adapters/
    scaffold/
    docs/

gtt-domain/
    context/
    adr/
    proposals/
    backlog.md
    change-request.md
    session.md
    .frozen

AGENTS.md

ADE integrations
    .claude/
    .kiro/
    .copilot/
    Codex-compatible AGENTS.md integration
```

The exact artifact list, paths, scripts, adapter names and compatibility
declarations must be obtained from the selected Bootstrap manifest and
contracts.

The CLI must not hard-code a duplicate artifact catalog if the Bootstrap
already provides one.

------------------------------------------------------------------------

# 5. Core Concept: Bootstrap Contract

The CLI and Bootstrap communicate through explicit contracts.

At minimum, the CLI must consume a Bootstrap manifest describing:

``` yaml
bootstrap:
  name:
  version:
  schema_version:

cli_compatibility:
  min_cli_version:
  max_cli_version:

project:
  required_paths:
  protected_paths:
  generated_paths:

ade:
  supported:
  overlays:
  detection:
  primary_selection:
  compatibility:

scripts:
  validate:
  status:
  freeze:
  session_context:
  index:
  reconcile:
  guard_sync:

scaffold:
  questionnaire:
  initialization:
  required_artifacts:

clean:
  excluded_paths:
  preserved_paths:

snapshot:
  required_state:
  restore_requirements:
```

This is illustrative.

The real schema must be defined by the Bootstrap.

The CLI consumes it.

------------------------------------------------------------------------

# 6. Multi-ADE Model

GTT must support projects where multiple AI Development Environments /
coding agents are represented.

Examples:

``` text
Claude only
    → Core + Claude integration

Claude + Copilot
    → Core + Claude + Copilot integrations

Claude + Codex + Copilot
    → Core + Claude + Codex + Copilot integrations
```

The CLI must never assume that one project has exactly one ADE.

## 6.1 Primary ADE

A project has exactly one **Primary ADE**.

The Primary ADE is the principal agent environment used for the
project's main GTT bootstrap interaction.

The user explicitly selects it.

The CLI must communicate that this is a significant project decision
because it affects the main interaction and execution path.

Example:

``` text
Detected ADEs:

  [1] Claude Code
  [2] GitHub Copilot
  [3] Codex

Select Primary ADE:

  > Claude Code

This selection defines the Primary ADE for this project.
Other detected ADEs remain installed and governed.
```

The CLI must not silently choose a Primary ADE when multiple ADEs are
detected.

If exactly one supported ADE is detected, the CLI may propose it as the
default, but the selection should remain explicit.

------------------------------------------------------------------------

# 7. Secondary ADEs

Secondary ADEs are not ignored.

This is important because a project can be modified by:

-   Claude Code;
-   GitHub Copilot;
-   Codex;
-   Kiro;
-   another supported ADE.

A secondary agent may:

-   edit source files;
-   edit configuration;
-   delete files;
-   modify infrastructure;
-   alter tests;
-   modify GTT artifacts.

Therefore, if an ADE is represented in the project, its GTT integration
must be installed and governed according to the Bootstrap contract.

The CLI must install:

``` text
GTT Core
+
all selected/detected supported ADE integrations
```

while recording:

``` text
Primary ADE
Secondary ADEs
Bootstrap version
CLI version
compatibility state
```

------------------------------------------------------------------------

# 8. CLI Technology

## 8.1 Language

Use:

``` text
Go
```

## 8.2 CLI framework

Use:

``` text
Cobra
```

## 8.3 Objectives

The binary should be:

-   fast;
-   portable;
-   deterministic;
-   dependency-light;
-   script-friendly;
-   suitable for CI;
-   suitable for local developer workflows;
-   easy to distribute;
-   easy to version;
-   easy to test.

The CLI must not embed a Python runtime or require Node.js.

The Bootstrap may continue to use Bash, Python or other implementation
technologies internally.

The CLI invokes Bootstrap capabilities rather than rewriting them.

------------------------------------------------------------------------

# 9. Proposed Repository

``` text
gtt-cli/
├── cmd/
│   └── gtt/
│       └── main.go
│
├── internal/
│   ├── bootstrap/
│   │   ├── resolver/
│   │   ├── installer/
│   │   ├── manifest/
│   │   └── compatibility/
│   │
│   ├── ade/
│   │   ├── detector/
│   │   └── primary/
│   │
│   ├── project/
│   │   ├── detector/
│   │   └── documents/
│   │
│   ├── lifecycle/
│   ├── execution/
│   ├── session/
│   ├── snapshot/
│   ├── export/
│   ├── update/
│   └── output/
│
├── test/
│   ├── integration/
│   └── fixtures/
│
├── go.mod
├── go.sum
├── README.md
└── LICENSE
```

There must be no package such as:

``` text
internal/governance/
internal/architecture/
internal/adr/
internal/reasoning/
internal/think/
internal/gtt-rules/
```

unless it is purely an operational adapter with no duplicated GTT
semantics.

------------------------------------------------------------------------

# 10. Command Surface

The v1.0 command surface should remain small.

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

Do not expose internal methodology as CLI commands.

------------------------------------------------------------------------

# 11. Command Responsibilities

## 11.1 `gtt init`

Main project initialization workflow.

Responsibilities:

1.  detect current project;
2.  detect existing GTT installation;
3.  detect available ADEs;
4.  select Primary ADE;
5.  select language;
6.  detect candidate initial documents;
7.  require user selection of initial source document(s);
8.  offer questionnaire when no suitable document exists;
9.  resolve compatible Bootstrap;
10. install Core;
11. install all required ADE integrations;
12. persist operational installation state;
13. validate;
14. construct bootstrap contract for Primary ADE;
15. optionally invoke the Primary ADE;
16. return control to the user.

The CLI does not design the project.

------------------------------------------------------------------------

## 11.2 `gtt status`

Operational state.

Status must be derived from actual state, not agent prose.

Examples:

``` text
GTT Project Status

Bootstrap:       vX.Y.Z
CLI:             v1.0.0
Compatibility:   OK

Primary ADE:     Claude Code
Other ADEs:      Copilot, Codex

Freeze:          ACTIVE
Change Request:  NONE
OPEN gaps:       2
Blocking gaps:   0
Proposals:       3
Validation:      PASS
Session:         CURRENT
```

The exact fields must be derived from Bootstrap contracts.

------------------------------------------------------------------------

# 12. Deterministic Session Handoff

Session state is operational information.

It is not:

-   evidence;
-   governed architecture;
-   authority;
-   a decision record.

The CLI may derive a session handoff from actual project state.

The principle is:

``` text
Project state
    ↓
CLI / Bootstrap status
    ↓
Derived session context
```

not:

``` text
Agent prose
    ↓
session truth
```

The session handoff must never become an alternative authority.

The existing Bootstrap session mechanism should be reused.

The CLI should invoke:

``` text
gtt-status
gtt-session-context
```

or their current Bootstrap equivalents.

The exact implementation belongs to Bootstrap.

------------------------------------------------------------------------

# 13. `gtt inspect`

`inspect` is for understanding the installation and project topology.

It may display:

``` text
Project
GTT installation
Bootstrap
ADEs
Primary ADE
Protected paths
Generated paths
Source documents
Scaffold
Session state
Compatibility
```

It should not interpret architecture.

------------------------------------------------------------------------

# 14. `gtt validate`

Validation is deterministic.

The CLI should invoke the Bootstrap validation contract.

Example:

``` text
gtt validate
```

and:

``` text
gtt validate --ci
```

CI mode must:

-   be non-interactive;
-   produce stable exit codes;
-   avoid interactive ADE invocation;
-   be machine-readable when requested;
-   never modify the project unless the Bootstrap validator explicitly
    defines generated validation state.

Example:

``` text
0 = valid
1 = validation failure
2 = invalid CLI/Bootstrap compatibility
3 = invalid invocation
```

The exact exit code contract should be frozen in v1.0.

------------------------------------------------------------------------

# 15. `gtt freeze`

Freeze is a facade over the Bootstrap freeze mechanism.

The CLI must not implement freeze logic itself.

Conceptually:

``` text
gtt freeze
    ↓
Bootstrap freeze contract
    ↓
deterministic validation
    ↓
human-authorized freeze
```

The human remains the authority for ratification.

The CLI does not decide that a design is correct.

There is no:

``` text
gtt unfreeze
```

A post-freeze architectural change follows the governed change process.

------------------------------------------------------------------------

# 16. `gtt resume`

Resume reconstructs the current operational state.

It should expose information such as:

``` text
last freeze
active change request
open gaps
blocking gaps
proposals
recent repository activity
session context
validation state
primary ADE
Bootstrap version
```

The source must be actual artifacts/state.

The CLI must not trust a free-form handoff written by an agent.

------------------------------------------------------------------------

# 17. `gtt doctor`

`doctor` is CLI-native diagnostics.

It should inspect:

``` text
CLI binary
Go/runtime-independent installation state
Git
project root
Bootstrap installation
Bootstrap version
CLI/Bootstrap compatibility
ADE detection
Primary ADE
ADE integration
required paths
permissions
protection
session state
validation
```

Example:

``` text
$ gtt doctor

[PASS] CLI
[PASS] Git
[PASS] Project root
[PASS] Bootstrap
[PASS] Compatibility
[PASS] Primary ADE
[PASS] ADE integrations
[PASS] GTT structure
[PASS] Protection
[WARN] Session state is stale

Result: 9 PASS, 1 WARN, 0 FAIL
```

Doctor may recommend corrective commands but must not silently modify
the project.

------------------------------------------------------------------------

# 18. `gtt audit`

Audit exposes traceability.

The CLI should consume Bootstrap audit/index information.

Possible output modes:

``` text
gtt audit
gtt audit --json
gtt audit --md
```

Audit can report:

-   Bootstrap version;
-   installation state;
-   ADE integrations;
-   Primary ADE;
-   source documents;
-   generated artifacts;
-   freeze state;
-   proposals;
-   change requests;
-   validation events;
-   update history;
-   clean exports;
-   recovery snapshots.

Audit is observability, not a second governance system.

------------------------------------------------------------------------

# 19. `gtt update`

Update is mandatory for v1.0.

It must not perform a blind overwrite.

Workflow:

``` text
current state
    ↓
discover available version
    ↓
resolve compatibility
    ↓
check migration requirements
    ↓
optional recovery snapshot
    ↓
install/update
    ↓
validate
    ↓
report result
```

If compatibility fails:

``` text
STOP
NO MODIFICATION
```

The CLI must never install a known-incompatible combination and merely
print a warning.

------------------------------------------------------------------------

# 20. Strict Compatibility

Compatibility is a hard gate.

At minimum:

``` text
CLI version
Bootstrap version
Bootstrap schema version
ADE integration version
project installation schema
```

must be checked.

Example:

``` text
CLI:        1.0.0
Bootstrap:  2.1.0
Schema:     3
Primary:    Claude Code adapter 2.0

Compatibility: PASS
```

Incompatible:

``` text
CLI 1.0.0 cannot operate with Bootstrap schema 5.

No changes were made.
Install a compatible Bootstrap/CLI combination.
```

No partial installation.

------------------------------------------------------------------------

# 21. Bootstrap Resolution

The CLI needs a resolver.

Conceptually:

``` text
Local project state
        ↓
requested version / channel
        ↓
Bootstrap registry/source
        ↓
manifest
        ↓
compatibility
        ↓
installation
```

The canonical Bootstrap repository remains the source of truth.

The CLI should support version pinning.

Recommended concepts:

``` text
stable
specific version
```

Avoid silently tracking `main` in production projects.

------------------------------------------------------------------------

# 22. One Canonical Bootstrap Repository

There must be one canonical Bootstrap source repository containing:

``` text
GTT Core
+
all supported ADE integrations
+
contracts
+
templates
+
scripts
+
scaffold
+
documentation required by the implementation
```

The CLI selects the relevant components during installation.

This avoids maintaining independent Bootstrap repositories per ADE.

Future distribution optimization is allowed:

``` text
Canonical Bootstrap
        ↓
distribution/build process
        ↓
optimized artifacts
```

but the source of truth remains one Bootstrap.

------------------------------------------------------------------------

# 23. `gtt init` --- Complete Workflow

The initialization flow is the most important UX of v1.0.

``` text
gtt init
   │
   ├── Detect project
   │
   ├── Detect existing GTT
   │
   ├── Detect ADEs
   │
   ├── Select Primary ADE
   │
   ├── Select language
   │
   ├── Detect initial documents
   │
   ├── User selects document(s)
   │
   ├── If none → request Bootstrap questionnaire
   │
   ├── Resolve Bootstrap
   │
   ├── Check compatibility
   │
   ├── Install Core
   │
   ├── Install all required ADE integrations
   │
   ├── Validate
   │
   ├── Generate operational initialization contract
   │
   ├── Deliver contract to Primary ADE
   │
   ├── Optionally launch Primary ADE
   │
   └── Return control to human
```

------------------------------------------------------------------------

# 24. Project Detection

The CLI should detect:

-   project root;
-   Git repository;
-   supported project indicators;
-   existing `.gtt`;
-   existing `gtt-domain`;
-   existing Bootstrap metadata;
-   candidate design documents;
-   detected ADEs.

It should not assume the current working directory is always the project
root.

If Git exists, use repository root when appropriate.

If the directory is not a project:

``` text
No project detected.

Initialize a new GTT project here?
```

The exact UX should be clear and safe.

------------------------------------------------------------------------

# 25. Existing GTT Detection

If GTT is already installed:

``` text
GTT is already initialized in this project.

Bootstrap: vX.Y.Z
Primary ADE: Claude Code

Choose:
  [1] Continue / inspect
  [2] Update
  [3] Reconfigure supported operational settings
  [4] Abort
```

Do not overwrite an existing installation during `init`.

If migration is required, route to update/migration.

------------------------------------------------------------------------

# 26. ADE Detection

ADE detection should be implemented as adapters.

Conceptually:

``` go
type Detector interface {
    ID() string
    Detect(projectRoot string) DetectionResult
}
```

Possible results:

``` text
Claude Code
Kiro
GitHub Copilot
Codex
```

Detection is evidence of representation in the project/environment.

The CLI should not infer that an agent is installed globally merely
because a generic editor exists.

Use the Bootstrap's declared detection contract where available.

------------------------------------------------------------------------

# 27. Primary ADE Selection

If multiple ADEs are detected:

``` text
Detected ADEs:

  1. Claude Code
  2. GitHub Copilot
  3. Codex

Select Primary ADE:
```

After selection:

``` text
Primary ADE: Claude Code

This defines the main AI development environment for this GTT project.
Other supported ADEs remain installed and governed.

Confirm? [y/N]
```

The selection must be stored in project installation state according to
Bootstrap contract.

------------------------------------------------------------------------

# 28. Language Selection

During initialization:

``` text
Select project interaction language:

  [1] Español
  [2] English
```

This setting may influence:

-   CLI messages;
-   generated questionnaire;
-   initialization contract;
-   ADE handoff;
-   generated bootstrap-facing text where supported.

The CLI must not duplicate GTT methodology in two languages.

Language-specific content belongs to Bootstrap templates/contracts where
appropriate.

------------------------------------------------------------------------

# 29. Initial Design Document Selection

The user must explicitly select the initial project/design source
document(s).

The CLI must not silently choose one.

Candidate documents may include:

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

The exact supported formats depend on the Bootstrap contract.

Example:

``` text
Candidate project documents:

  [1] README.md
  [2] docs/product-requirements.md
  [3] docs/architecture.md
  [4] docs/api-design.md

Select one or more:
> 2,3,4
```

------------------------------------------------------------------------

# 30. Single Document Case

If exactly one plausible document is detected:

``` text
Initial project document detected:

  docs/architecture.md

Use this as initial source material? [Y/n]
```

Do not silently consume it.

The user owns the decision about what constitutes the starting source
material.

------------------------------------------------------------------------

# 31. Multiple Documents

When multiple documents exist, allow:

``` text
one document
multiple documents
```

The CLI must preserve the selected set.

The CLI must not decide which document is authoritative.

If precedence between documents is needed, that is part of the governed
process and Bootstrap methodology.

The CLI merely captures the user's initial source selection.

------------------------------------------------------------------------

# 32. Iterative Documentation Model

Initialization must explain an important principle to the user.

The initial documents are the starting source material.

They are not permanently frozen truth merely because they were selected
during `gtt init`.

Later, the user may provide:

-   a more complete design document;
-   updated requirements;
-   additional architecture documentation;
-   security requirements;
-   infrastructure documentation;
-   operational constraints.

Those materials become new evidence.

Once architecture is governed/frozen, changes to governed architecture
must follow the normal GTT change/ADR/proposal/freeze process.

The CLI should communicate this clearly without embedding the
methodology itself.

Example:

``` text
GTT initialization uses these documents as initial source material.

Documentation can evolve.
New or improved documents may be introduced later.

Once governed architecture is frozen, changes to governed artifacts
must follow the project's GTT change process rather than silently
overwriting governed context.
```

------------------------------------------------------------------------

# 33. No Design Document

If no suitable design document exists, the CLI must **not implement or
contain the questionnaire itself**.

The questionnaire is a Bootstrap artifact.

The CLI workflow is:

``` text
No suitable design source detected
        ↓
Resolve compatible Bootstrap
        ↓
Read Bootstrap scaffold/template contract
        ↓
Obtain GTT-INITIAL-DESIGN-QUESTIONNAIRE.md
        ↓
Materialize it in the project
        ↓
Prepare Primary ADE handoff
        ↓
Primary ADE conducts guided design elicitation
```

The CLI must not embed the questionnaire content in Go.

The CLI must not maintain a second questionnaire version.

The Bootstrap is responsible for:

-   questionnaire content;
-   questionnaire structure;
-   elicitation guidance;
-   provenance conventions;
-   Minimum Viable Governed Design expectations;
-   future questionnaire evolution.

The CLI is responsible only for:

-   detecting that sufficient design source material is missing;
-   obtaining the questionnaire from the Bootstrap;
-   materializing it;
-   reporting its location;
-   handing control to the Primary ADE.

------------------------------------------------------------------------

# 34. Bootstrap-Owned Initial Design Questionnaire

The initial design questionnaire is located and versioned by the GTT
Bootstrap.

Recommended Bootstrap structure:

``` text
.gtt/
└── scaffold/
    └── templates/
        └── GTT-INITIAL-DESIGN-QUESTIONNAIRE.md
```

The exact path must come from the Bootstrap manifest rather than being
hard-coded by the CLI.

The CLI should consume a Bootstrap contract conceptually equivalent to:

``` yaml
scaffold:
  initial_design:
    questionnaire:
      template: <Bootstrap-defined path>
      output: <Bootstrap-defined project path>
      version: <Bootstrap-defined contract/version>
```

The actual schema is owned by Bootstrap.

### Important boundary

The questionnaire is:

``` text
Bootstrap artifact
        ↓
materialized into project
        ↓
source material for guided elicitation
```

It is not:

``` text
CLI methodology
```

and it is not automatically:

``` text
governed architecture
```

------------------------------------------------------------------------

# 34.1 Guided ADE Elicitation

The preferred user experience is not:

``` text
CLI creates empty questionnaire
        ↓
User manually fills 50 questions
        ↓
ADE reads it
```

The preferred experience is:

``` text
CLI creates questionnaire
        ↓
Primary ADE reads Bootstrap instructions
        ↓
ADE inspects project and available source material
        ↓
ADE interviews the human progressively
        ↓
ADE writes answers into questionnaire
        ↓
ADE identifies missing information
        ↓
ADE identifies conflicts
        ↓
ADE marks proposals
        ↓
Human clarifies / decides
        ↓
Minimum Viable Governed Design
```

The ADE should conduct a professional discovery conversation.

It should not force the human to answer the entire questionnaire in one
pass.

For example:

``` text
ADE:
"I found no complete design document.

I created the GTT Initial Design Questionnaire from the Bootstrap.
I'll use it to guide the initial design conversation.

Let's start with the problem:
What are you trying to build and what problem should it solve?"
```

The ADE then progressively updates the questionnaire.

------------------------------------------------------------------------

# 34.2 Evidence-Aware Elicitation

Before asking the human a question, the ADE should inspect available
project evidence when the Bootstrap contract permits it.

For example:

``` text
README.md
docs/
existing source code
existing configuration
existing infrastructure
selected initial documents
```

If the answer is already established by evidence, the ADE should not
unnecessarily ask the human to repeat it.

If information is missing:

``` text
[VACÍO]
```

should remain visible.

If sources disagree:

``` text
[CONFLICTO]
```

should remain visible.

If the ADE suggests an option:

``` text
[PROPUESTA]
```

must be used.

The ADE must not turn an inference into a decision.

------------------------------------------------------------------------

# 34.3 Human Decision Boundary

The questionnaire may contain:

``` text
facts
requirements
constraints
unknowns
conflicts
proposals
pending decisions
```

It must not silently contain:

``` text
unapproved architectural decisions
```

The human remains the authority for decisions.

The questionnaire therefore acts as a bridge:

``` text
Initial project understanding
        ↓
Human + ADE elicitation
        ↓
Minimum viable design understanding
        ↓
GTT governed design process
```

The exact transition into governed context is defined by Bootstrap.

------------------------------------------------------------------------

# 34.4 Questionnaire Evolution

The CLI must not own questionnaire evolution.

If GTT changes the required minimum design information:

``` text
Bootstrap update
        ↓
new questionnaire contract
        ↓
CLI automatically consumes new version
```

The CLI should not require a code change merely because questionnaire
content changes.

This is a key architectural invariant.

------------------------------------------------------------------------

# 34.5 Questionnaire Completion

The CLI must not determine whether the architecture is complete.

It may report operational status such as:

``` text
Questionnaire created
Questionnaire updated
Questionnaire exists
Questionnaire not found
```

The Bootstrap/ADE process determines whether the resulting material is
sufficient to continue.

The CLI must not implement:

``` text
"architecture completeness score"
```

or another methodology-specific assessment.

# 35. ADE Bootstrap Contract

After installation, the CLI must deliver a precise contract to the
Primary ADE.

The contract must be generated from Bootstrap templates/contracts, not
manually duplicated in Go.

Conceptually it must communicate:

``` text
GTT Bootstrap is installed.

Primary ADE:
    Claude Code

Other governed ADEs:
    Copilot
    Codex

Initial source documents:
    docs/architecture.md
    docs/requirements.md

Language:
    English

Required action:

1. Read the installed GTT Bootstrap instructions.
2. Read AGENTS.md and applicable ADE instructions.
3. Read the selected initial source material.
4. Inspect the project.
5. Populate the applicable GTT scaffold.
6. Preserve evidence/provenance requirements.
7. Identify missing information.
8. Identify conflicts.
9. Mark proposals where information is not established.
10. Do not invent undocumented architectural decisions.
11. Ask the human when a decision is required.
12. Do not treat the source document as governed truth automatically.
13. Follow the installed Bootstrap contracts.
```

The exact text should come from Bootstrap.

------------------------------------------------------------------------

# 36. CLI Must Not Populate Architecture

The CLI may create:

-   installation metadata;
-   questionnaire template;
-   Bootstrap files;
-   ADE integration files;
-   operational state;
-   recovery snapshot.

The CLI must not create architectural decisions such as:

``` text
Use PostgreSQL
Use Redis
Use Kafka
Use microservices
Use Kubernetes
Use AWS
Use REST
```

unless these are explicitly present in user-selected source material and
the Bootstrap contract is merely preserving source material.

The CLI must never infer them.

------------------------------------------------------------------------

# 37. Agentic Bootstrap Phase

After initialization, the Primary ADE becomes the main execution
environment.

The ADE should:

``` text
read source
    ↓
inspect project
    ↓
read Bootstrap instructions
    ↓
recover evidence
    ↓
identify missing information
    ↓
identify conflicts
    ↓
populate scaffold
    ↓
produce proposals where required
    ↓
ask human for decisions
```

The CLI orchestrates this transition.

It does not replace the ADE.

------------------------------------------------------------------------

# 38. Human Authority

The human remains the decision authority.

The CLI does not:

-   approve architecture;
-   classify architectural truth;
-   resolve conflicts;
-   ratify proposals;
-   infer intent;
-   decide when a design is correct.

The CLI can enforce that required operational steps exist.

The Bootstrap governs what those steps mean.

------------------------------------------------------------------------

# 39. Evidence Boundary

The underlying GTT model separates:

``` text
AUTHORIZED SOURCES
        ↓
GROUNDING
        ↓
EVIDENCE DOSSIER
        ↓
WORKERS / THINK AGENTS
        ↓
MARKED PROPOSALS
        ↓
HUMAN
        ↓
FREEZE
```

The CLI must preserve this architecture.

It must not introduce a shortcut such as:

``` text
CLI
  ↓
raw document
  ↓
agent
```

The CLI may identify and register initial sources.

The Bootstrap governs how those sources are grounded and transformed
into evidence.

------------------------------------------------------------------------

# 40. Source Material Is Not Automatically Authority

The following distinction must remain explicit:

``` text
Source document
    ≠
Governed context
```

A source document may contain:

-   facts;
-   assumptions;
-   proposals;
-   outdated decisions;
-   conflicts;
-   missing information.

The Bootstrap's grounding and governance process determines how that
material is treated.

The CLI only establishes the initial source set.

------------------------------------------------------------------------

# 41. Post-Freeze Changes

The CLI must respect the re-entrant nature of GTT.

Conceptually:

``` text
FREEZE
   ↓
WORK
   ↓
DISCOVERY
   ↓
CHANGE REQUEST
   ↓
IMPACT ANALYSIS
   ↓
THINK
   ↓
HUMAN DECISION
   ↓
NEW FREEZE
   ↓
WORK
```

The CLI must not introduce:

``` text
unfreeze
```

A frozen project is not made ungoverned by a CLI switch.

------------------------------------------------------------------------

# 42. `gtt export --clean`

A central production requirement is the ability to maintain:

``` text
GTT development project
```

while generating:

``` text
clean QA/PRO delivery
```

without destroying GTT.

Command:

``` text
gtt export --clean
```

Optional destination:

``` text
gtt export --clean ./dist/project
```

Conceptually:

``` text
GTT development project
        │
        ├── source
        ├── tests
        ├── deployment
        ├── GTT
        └── ADE integrations
                │
                ▼
        gtt export --clean
                │
                ▼
        clean delivery artifact
```

The source development project remains GTT-governed.

------------------------------------------------------------------------

# 43. Clean Export Rules

The clean export must be based on a Bootstrap-defined exclusion
contract.

Potential exclusions:

``` text
.gtt/
gtt-domain/
ADE-specific GTT integration
GTT control files
GTT session state
GTT development metadata
```

Potentially preserved:

``` text
application source
tests
build files
deployment files
configuration required by application
documentation required for delivery
```

The exact exclusion/preservation list must come from Bootstrap.

The CLI must not maintain a second hard-coded list that can drift from
Bootstrap.

------------------------------------------------------------------------

# 44. `gtt clean`

`clean` is different from `export --clean`.

### Export

Creates another clean artifact.

The development project remains intact.

### Clean

Removes GTT from the current project.

Because this is destructive, it must require explicit confirmation.

Example:

``` text
WARNING

This operation removes GTT from the current project.

The current GTT development state will no longer be available
from this working tree unless a recovery snapshot is preserved.

Continue? [y/N]
```

------------------------------------------------------------------------

# 45. `dev-gtt` Recovery Snapshot

Before destructive clean, offer:

``` text
Save portable GTT recovery snapshot? [Y/n]
```

The snapshot is not a raw backup of the entire repository.

It is a portable GTT recovery package containing the information
necessary to reconstruct the GTT installation and operational state.

Potential contents:

``` text
Bootstrap version
CLI compatibility
installation manifest
Primary ADE
secondary ADEs
language
selected initial sources
installation configuration
session state
audit metadata
recovery metadata
required project references
```

The exact contents must be defined by Bootstrap.

------------------------------------------------------------------------

# 46. Restore from `dev-gtt`

Future `gtt init` should be able to detect a recovery snapshot.

Example:

``` text
GTT recovery snapshot detected.

Project previously used:
  Bootstrap: vX.Y.Z
  Primary ADE: Claude Code

Restore GTT installation? [Y/n]
```

Restore must run compatibility validation before modification.

------------------------------------------------------------------------

# 47. Clean vs Export

  -------------------------------------------------------------------------
  Operation               Current project      GTT removed New clean
                                 modified                  artifact
  ---------------------- ---------------- ---------------- ----------------
  `gtt export --clean`                 NO               NO YES

  `gtt clean`                         YES              YES NO
  -------------------------------------------------------------------------

This distinction is fundamental.

------------------------------------------------------------------------

# 48. Installation State

The CLI needs a small operational state record.

It should not duplicate governed architecture.

Example conceptual state:

``` yaml
gtt:
  bootstrap_version:
  bootstrap_schema:
  cli_version:
  installed_at:

project:
  root:
  repository:

ade:
  primary:
  installed:

sources:
  initial:
    - path

language:

snapshot:
  last:
```

The exact schema should be defined by Bootstrap.

If Bootstrap already provides an installation manifest, reuse it.

Do not create two competing manifests.

------------------------------------------------------------------------

# 49. State Precedence

The CLI must distinguish:

``` text
Governed state
Operational state
Session state
User preferences
Agent notes
```

The CLI must never elevate operational state into governed authority.

For example:

``` text
session.md
```

must not become an architectural decision source merely because it
exists.

Likewise, a CLI-generated handoff must not enter the grounding corpus
unless the Bootstrap explicitly defines it as source material.

------------------------------------------------------------------------

# 50. Working Preferences

Working preferences are not the same as project architecture.

The CLI should not create free-form project instructions that can
override GTT governance.

If Bootstrap defines user/team working preferences, the CLI should
merely install/reference them according to that contract.

A preference must never override governed constraints.

------------------------------------------------------------------------

# 51. ADE Invocation

The CLI should provide an execution layer.

Conceptually:

``` go
type ADEExecutor interface {
    ID() string
    Available() bool
    BuildCommand(contract ExecutionContract) Command
    Execute(contract ExecutionContract) error
}
```

The CLI should not embed agent-specific reasoning.

Each ADE adapter should only know:

-   how to detect the ADE;
-   how to prepare the handoff;
-   how to invoke it;
-   how to return exit status.

The Bootstrap owns the instructions.

------------------------------------------------------------------------

# 52. Invocation Modes

The CLI should support:

``` text
install only
handoff
invoke
```

Possible UX:

``` text
gtt init
```

may end with:

``` text
GTT Bootstrap completed.

Primary ADE: Claude Code

Ready to continue the GTT initialization with Claude Code.

[1] Launch Claude Code
[2] Show bootstrap contract
[3] Exit
```

If direct invocation is technically available and safe, automatic
invocation may be supported.

The CLI must never make destructive external actions without clear user
intent.

------------------------------------------------------------------------

# 53. ADE Independence

The GTT project must remain usable even if the Primary ADE changes.

Example:

``` text
Claude Code → Primary
Copilot → Secondary
```

Later:

``` text
Copilot → Primary
Claude Code → Secondary
```

The CLI should support changing Primary ADE through an explicit
operational command or configuration flow.

This must be treated as a significant project change in UX.

The CLI must not silently change Primary ADE because another agent is
detected.

------------------------------------------------------------------------

# 54. Bootstrap Update and Primary ADE

Updating Bootstrap must preserve:

``` text
Primary ADE
Secondary ADEs
selected source metadata
project state
```

unless a migration contract explicitly changes them.

------------------------------------------------------------------------

# 55. Security Model

The CLI must assume that project contents may be untrusted.

Security concerns include:

-   malicious scripts in repositories;
-   path traversal;
-   symlink attacks;
-   malicious Bootstrap packages;
-   arbitrary execution;
-   unsafe ADE invocation;
-   accidental deletion.

Minimum requirements:

## 55.1 Path safety

Never allow a manifest entry to escape project/bootstrap boundaries.

Reject:

``` text
../../outside-project
```

and unsafe absolute paths where not explicitly allowed.

## 55.2 Symlink safety

Validate symlinks before destructive operations.

## 55.3 Archive safety

If Bootstrap is downloaded as an archive:

-   validate archive paths;
-   reject path traversal;
-   verify expected structure;
-   verify integrity;
-   verify version metadata.

## 55.4 Destructive operations

Require explicit confirmation.

## 55.5 External commands

Use structured process execution rather than shell string concatenation.

------------------------------------------------------------------------

# 56. Integrity Verification

Bootstrap distribution should support integrity verification.

Possible mechanism:

``` text
version
checksum
signature
```

The exact trust mechanism should be defined by the distribution
strategy.

The CLI should fail closed if a required integrity check fails.

------------------------------------------------------------------------

# 57. Network Behavior

The CLI should make network operations explicit.

Examples:

``` text
gtt init
```

may need network access to obtain Bootstrap.

Offline operation should be possible when a local Bootstrap/cache is
available.

Potential:

``` text
gtt init --offline
```

If v1.0 supports it, the resolver should report:

``` text
No compatible Bootstrap available locally.
```

rather than silently falling back to an unknown version.

------------------------------------------------------------------------

# 58. Bootstrap Cache

The CLI may maintain a local cache.

Conceptually:

``` text
~/.gtt/
    cache/
        bootstrap/
            vX.Y.Z/
```

The cache is CLI operational state.

It is not project governance.

The exact location should respect platform conventions.

------------------------------------------------------------------------

# 59. Configuration

Avoid excessive global configuration.

Project-specific operational state should live with the project where
appropriate.

Global configuration may include:

``` text
Bootstrap registry
cache location
default language
network behavior
telemetry preference, if ever introduced
```

No global setting may silently override project governance.

------------------------------------------------------------------------

# 60. CLI UX

The CLI should feel professional.

Use:

``` text
clear headings
short explanations
explicit choices
safe defaults
deterministic errors
actionable next steps
```

Avoid:

``` text
verbose internal implementation details
methodology lectures
agent jargon when unnecessary
ambiguous confirmations
silent destructive operations
```

Example:

``` text
$ gtt init

GTT Project Initialization

Project: payments-api

Detected ADEs:
  1. Claude Code
  2. GitHub Copilot

Primary ADE:
  Select one: 1

Language:
  1. English
  2. Español

Initial source documents:
  1. docs/requirements.md
  2. docs/architecture.md

Select documents:
  1,2

Bootstrap:
  Resolving compatible version...
  Installing Core...
  Installing Claude Code integration...
  Installing GitHub Copilot integration...
  Validating...

GTT initialization complete.

Primary ADE: Claude Code
Bootstrap: vX.Y.Z
Validation: PASS
```

------------------------------------------------------------------------

# 61. Error Design

Every error should answer:

1.  What happened?
2.  Why?
3.  What can the user do?

Bad:

``` text
error: exit status 1
```

Good:

``` text
GTT Bootstrap installation failed.

Reason:
The selected Bootstrap requires CLI >= 1.2.0.
Current CLI: 1.0.0

No project files were modified.

Next step:
Upgrade the GTT CLI and run `gtt init` again.
```

------------------------------------------------------------------------

# 62. No Partial Installation

Installation should be transactional where practical.

Preferred flow:

``` text
resolve
↓
validate
↓
stage
↓
verify
↓
commit
↓
validate project
```

If installation fails before commit:

``` text
project unchanged
```

If failure occurs after a recoverable mutation:

``` text
rollback
```

or use a documented transactional installation mechanism.

------------------------------------------------------------------------

# 63. Idempotency

Commands should be idempotent where possible.

Examples:

``` text
gtt validate
gtt status
gtt inspect
gtt doctor
```

should never alter project architecture.

`gtt init` should detect an existing installation rather than
reinstalling blindly.

`gtt update` should detect that the requested version is already
installed.

------------------------------------------------------------------------

# 64. CI/CD Integration

The CLI must support noninteractive use.

Primary CI command:

``` text
gtt validate --ci
```

Potential future:

``` text
gtt audit --json
gtt status --json
```

CI should be able to detect:

-   invalid installation;
-   incompatible versions;
-   invalid GTT structure;
-   invalid protected artifacts;
-   failed Bootstrap validation.

The CLI must return stable exit codes.

------------------------------------------------------------------------

# 65. JSON Output

Operational commands should progressively support:

``` text
--json
```

Recommended candidates:

``` text
gtt status --json
gtt inspect --json
gtt validate --json
gtt doctor --json
gtt audit --json
```

The JSON schema should be documented and versioned.

Do not expose unstable internal structs as public API.

------------------------------------------------------------------------

# 66. Logging

Provide structured logging internally.

User-facing output should remain concise.

Suggested levels:

``` text
error
warn
info
debug
```

Example:

``` text
--verbose
--debug
```

Debug output must not leak:

-   secrets;
-   access tokens;
-   credentials;
-   private environment variables.

------------------------------------------------------------------------

# 67. Exit Codes

Define a stable exit-code contract.

Suggested:

``` text
0   success
1   validation / operational failure
2   incompatible version
3   invalid command or arguments
4   user cancellation
5   project state conflict
6   Bootstrap integrity failure
7   external dependency unavailable
```

The exact v1.0 contract should be frozen before release.

------------------------------------------------------------------------

# 68. Testing Strategy

Testing must be layered.

## Unit tests

Test:

-   version resolution;
-   compatibility;
-   manifest parsing;
-   ADE detection;
-   document selection;
-   questionnaire generation;
-   path validation;
-   clean-export filtering;
-   snapshot metadata;
-   command parsing.

## Integration tests

Use fixture projects.

Example:

``` text
fixtures/
├── empty-project/
├── single-ade/
├── multi-ade/
├── existing-gtt/
├── incompatible-bootstrap/
├── no-design-document/
├── multiple-design-documents/
├── frozen-project/
└── clean-export/
```

## End-to-end tests

Validate:

``` text
gtt init
→ install
→ validate
→ status
→ resume
→ freeze
→ export --clean
→ clean
```

with real Bootstrap fixtures.

------------------------------------------------------------------------

# 69. Multi-ADE Test Matrix

At minimum:

  Project                    Primary   Secondary         Expected
  -------------------------- --------- ----------------- ----------------
  Claude                     Claude    none              Core + Claude
  Copilot                    Copilot   none              Core + Copilot
  Claude + Copilot           Claude    Copilot           Core + both
  Claude + Codex             Claude    Codex             Core + both
  Claude + Copilot + Codex   Claude    Copilot + Codex   Core + all
  Multiple ADEs              Copilot   Claude + Codex    Core + all

The test must verify that secondary ADE integrations are not
accidentally omitted.

------------------------------------------------------------------------

# 70. Initialization Test Matrix

## One document

``` text
one candidate
→ show
→ confirm
→ install
```

## Multiple documents

``` text
multiple candidates
→ show
→ user selects one or more
→ install
```

## No document

``` text
none
→ offer questionnaire
→ generate questionnaire
```

## User declines questionnaire

``` text
exit without pretending that project design exists
```

------------------------------------------------------------------------

# 71. Update Test Matrix

Test:

``` text
same version
compatible upgrade
compatible downgrade if supported
incompatible CLI
incompatible Bootstrap
incompatible ADE
failed integrity
failed installation
failed validation
```

Expected rule:

``` text
No unsafe partial update.
```

------------------------------------------------------------------------

# 72. Clean Export Test Matrix

Test:

``` text
source project remains unchanged
GTT excluded
ADE integration excluded
application source preserved
tests preserved
deployment preserved
destination created
destination does not contain governance artifacts
```

The actual inclusion/exclusion rules must be generated from Bootstrap.

------------------------------------------------------------------------

# 73. Snapshot Test Matrix

Test:

``` text
create snapshot
validate snapshot
remove GTT
detect snapshot
restore
validate restored installation
```

Verify that recovery does not silently alter application source.

------------------------------------------------------------------------

# 74. Project State Model

The CLI can model operational state as:

``` text
UNINITIALIZED
     ↓
DETECTED
     ↓
CONFIGURING
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

This is an operational state machine.

It must not replace the Bootstrap's governance state model.

------------------------------------------------------------------------

# 75. Source Selection State

The CLI should record:

``` text
initial_sources:
  - path
  - selection_time
  - source_type
```

It should not record:

``` text
source_is_truth: true
```

because source material is not automatically governed truth.

------------------------------------------------------------------------

# 76. Primary ADE State

Record:

``` text
primary_ade:
  id
  version/integration
  selected_at
```

Secondary ADEs:

``` text
secondary_ades:
  - id
  - version/integration
```

Do not infer Primary ADE from most recent usage.

------------------------------------------------------------------------

# 77. Changing Primary ADE

A future command may be:

``` text
gtt config primary-ade
```

or equivalent.

This is intentionally not in the minimal initial command surface unless
needed by implementation.

If implemented, UX must say:

``` text
Changing the Primary ADE changes the project's main GTT execution environment.

Existing secondary ADE integrations remain governed.

Continue?
```

No silent changes.

------------------------------------------------------------------------

# 78. Bootstrap Adapter Layer

The CLI should isolate Bootstrap-specific details.

Example:

``` go
type Bootstrap interface {
    Manifest() Manifest
    Install(ctx context.Context, target Project) error
    Validate(ctx context.Context, target Project) error
    Status(ctx context.Context, target Project) (Status, error)
    Freeze(ctx context.Context, target Project) error
    SessionContext(ctx context.Context, target Project) (SessionContext, error)
    ExportClean(ctx context.Context, target Project, destination string) error
}
```

This is an abstraction example, not a requirement to make the Bootstrap
a Go library.

The actual implementation can invoke scripts/processes.

------------------------------------------------------------------------

# 79. Process Execution

The CLI should have one centralized process-execution layer.

Responsibilities:

-   command execution;
-   environment;
-   working directory;
-   stdout/stderr;
-   timeout;
-   exit code;
-   cancellation;
-   logging;
-   security.

Do not scatter `exec.Command` throughout the codebase.

------------------------------------------------------------------------

# 80. Bootstrap Script Execution

Existing Bootstrap scripts should be reused.

Examples include:

``` text
gtt-validate.sh
gtt-status.sh
gtt-freeze.sh
gtt-session-context.sh
gtt-query.sh
gtt-index.sh
gtt-reconcile.sh
gtt-guard-sync.sh
```

These names are examples from the current Bootstrap and must ultimately
be resolved from the actual Bootstrap manifest.

The CLI should call the manifest-defined operation rather than assuming
filenames forever.

------------------------------------------------------------------------

# 81. Migration Strategy

The first CLI implementation does not need to rewrite Bootstrap scripts
into Go.

Recommended:

``` text
Go CLI
   ↓
Bootstrap contract
   ↓
existing scripts
```

Later:

``` text
Go CLI
   ↓
Bootstrap API/contract
```

if the Bootstrap itself evolves.

This keeps the CLI thin and reduces duplicated logic.

------------------------------------------------------------------------

# 82. CLI Internal Packages

## `bootstrap`

Responsible for:

-   resolution;
-   manifest;
-   installation;
-   compatibility;
-   integrity.

## `ade`

Responsible for:

-   detection;
-   selection;
-   invocation.

## `project`

Responsible for:

-   project root;
-   candidate documents;
-   questionnaire orchestration.

## `lifecycle`

Responsible for:

-   init;
-   status;
-   resume;
-   freeze;
-   clean.

## `session`

Responsible only for operational session context.

## `snapshot`

Responsible for dev-gtt recovery snapshots.

## `export`

Responsible for clean delivery artifacts.

## `update`

Responsible for update lifecycle.

## `output`

Responsible for:

-   terminal UX;
-   JSON;
-   errors;
-   progress.

------------------------------------------------------------------------

# 83. Dependency Direction

Preferred dependency direction:

``` text
cmd
 ↓
application/use cases
 ↓
operational services
 ↓
adapters
 ↓
external systems
```

Avoid circular dependencies.

Bootstrap-specific implementation details should remain behind
interfaces.

------------------------------------------------------------------------

# 84. CLI Does Not Need an LLM

The CLI itself should not require an LLM.

It should be deterministic.

The intelligence lives in:

``` text
Bootstrap
+
ADE
+
agents
```

This is an important architectural property.

The CLI must work even when no model is available for:

``` text
status
inspect
validate
doctor
audit
update
export
clean
```

Only ADE-assisted initialization/continuation requires an agent
environment.

------------------------------------------------------------------------

# 85. No Hidden Agent Logic

Do not implement prompts such as:

``` text
You are an architect...
```

inside Go source.

Those belong to Bootstrap contracts/templates.

The CLI may pass structured variables into a Bootstrap-generated
contract.

------------------------------------------------------------------------

# 86. Questionnaire Ownership

The questionnaire should be a Bootstrap artifact.

The CLI:

``` text
detects absence
→ requests template
→ materializes questionnaire
→ tells user what to do next
```

It should not own the questionnaire's methodology.

------------------------------------------------------------------------

# 87. Initialization Contract Data

The CLI should produce structured input such as:

``` yaml
project:
  root: /project
  language: en

ade:
  primary: claude
  detected:
    - claude
    - copilot
    - codex

sources:
  selected:
    - docs/requirements.md
    - docs/architecture.md

bootstrap:
  version: X.Y.Z
```

The Bootstrap transforms this into the actual ADE-facing contract.

------------------------------------------------------------------------

# 88. Human Interaction Boundaries

The CLI asks the human when the decision is operational:

``` text
Which ADE?
Which language?
Which source documents?
Create questionnaire?
Confirm destructive action?
Confirm Primary ADE?
```

The CLI does not ask the human to manually perform operations that it
can safely automate.

For example:

``` text
Do you want me to copy .gtt/?
```

should not be necessary.

The installation process owns that.

------------------------------------------------------------------------

# 89. Initialization Completion

At the end of `gtt init`, the user should know:

``` text
GTT installed
Bootstrap version
Primary ADE
Secondary ADEs
Initial sources
Language
Validation result
Next action
```

Example:

``` text
GTT initialization complete.

Bootstrap:       v2.1.0
CLI:             v1.0.0
Primary ADE:     Claude Code
Other ADEs:      Copilot, Codex
Initial sources: 3
Language:        English
Validation:      PASS

Next:
Launch Claude Code to complete the governed project bootstrap.
```

------------------------------------------------------------------------

# 90. Operational Contract with Claude Code

For Claude Code specifically, the CLI must not attempt to replace
Claude's native instruction mechanisms.

Instead:

``` text
GTT Bootstrap
    ↓
AGENTS.md / Claude integration / Bootstrap contract
    ↓
Claude Code
```

The CLI should prepare the project so Claude Code naturally discovers
the correct GTT instructions.

If Claude Code supports a native invocation mechanism, use it through
the ADE adapter.

Do not create competing Claude memory or instruction systems.

------------------------------------------------------------------------

# 91. ADE-Neutral Governance

Even though the Primary ADE is explicit, the GTT project remains
ADE-neutral at the governance level.

The model is:

``` text
GTT
 ├── Core
 ├── Claude integration
 ├── Kiro integration
 ├── Copilot integration
 └── Codex integration
```

The governance contract remains the same.

Only the integration mechanism changes.

------------------------------------------------------------------------

# 92. Future Local GTT Service

The CLI should not block a future local GTT service.

Possible future architecture:

``` text
             GTT CLI
                │
                ▼
          Local GTT Service
          /       |       \
       Claude   Copilot   Codex
```

The service could periodically derive session memory/state.

However, v1.0 should not require a daemon.

The Bootstrap's existing session mechanism remains authoritative for
implementation.

The CLI can provide command-driven updates first.

------------------------------------------------------------------------

# 93. Session Memory Safety

Session memory/handoff must remain:

``` text
operational history
```

not:

``` text
architecture authority
```

If a service later updates session memory periodically, it must derive
it from real project state.

It must not ask an agent:

``` text
What did we decide?
```

and write that answer as authoritative state.

The safe model is:

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
freeze state
+
validation
        ↓
derived session handoff
```

------------------------------------------------------------------------

# 94. Source of Truth Hierarchy

At the CLI boundary, maintain the distinction:

``` text
GTT Canonical Method
        ↓
Bootstrap implementation/contracts
        ↓
Governed project artifacts
        ↓
Operational state
        ↓
Session handoff
        ↓
Agent notes
```

Lower layers must never silently override higher layers.

------------------------------------------------------------------------

# 95. Documentation Requirements

The CLI repository should contain:

``` text
README.md
ARCHITECTURE.md
DEVELOPMENT.md
CONTRIBUTING.md
RELEASE.md
```

Only if these are useful for the CLI project.

GTT-specific governed Markdown must follow the GTT canonical header
rule.

------------------------------------------------------------------------

# 96. Versioning

Use semantic versioning:

``` text
MAJOR.MINOR.PATCH
```

The CLI version and Bootstrap version are independent but compatibility
is explicit.

Example:

``` text
GTT CLI       1.0.0
Bootstrap     2.1.0
Schema        4
```

------------------------------------------------------------------------

# 97. Release Channels

Recommended:

``` text
stable
preview
development
```

Production projects should default to:

``` text
stable
```

A project should record the resolved Bootstrap version.

------------------------------------------------------------------------

# 98. Backward Compatibility

CLI v1.0 should not attempt to support arbitrary historical Bootstrap
versions.

Support a documented compatibility matrix.

Example:

  CLI   Bootstrap   Result
  ----- ----------- ------------------
  1.0   2.1         PASS
  1.0   2.0         PASS if declared
  1.0   1.x         REFUSE
  0.x   2.1         REFUSE

The actual matrix must be defined by release.

------------------------------------------------------------------------

# 99. Migration

If an update requires migration:

``` text
current version
    ↓
migration plan
    ↓
snapshot
    ↓
migration
    ↓
validation
```

No migration should be hidden inside a simple copy operation.

------------------------------------------------------------------------

# 100. Clean Architecture Rule

The CLI must be architecturally boring.

That is a feature.

It should be:

``` text
resolver
installer
detector
orchestrator
executor
validator facade
snapshotter
exporter
```

It should not become:

``` text
AI framework
governance framework
architecture engine
agent framework
RAG engine
```

------------------------------------------------------------------------

# 101. Definition of Done --- CLI Core

The CLI is not ready when commands merely exist.

The minimum v1.0 implementation is complete when:

-   [ ] Go project builds reproducibly.
-   [ ] Cobra command tree exists.
-   [ ] Bootstrap can be resolved.
-   [ ] Bootstrap manifest can be parsed.
-   [ ] CLI/Bootstrap compatibility is enforced.
-   [ ] Project root can be detected.
-   [ ] Existing GTT can be detected.
-   [ ] ADEs can be detected.
-   [ ] Primary ADE can be selected.
-   [ ] Multiple ADEs can be installed.
-   [ ] Language can be selected.
-   [ ] Initial source documents can be selected.
-   [ ] No-document questionnaire path works.
-   [ ] Bootstrap is installed transactionally.
-   [ ] Bootstrap validation is invoked.
-   [ ] ADE bootstrap contract is generated from Bootstrap.
-   [ ] Primary ADE invocation/handoff works.
-   [ ] `status` works.
-   [ ] `inspect` works.
-   [ ] `validate` works.
-   [ ] `resume` works.
-   [ ] `freeze` delegates to Bootstrap.
-   [ ] `doctor` works.
-   [ ] `audit` works.
-   [ ] `update` works with strict compatibility.
-   [ ] `export --clean` works.
-   [ ] `clean` is protected and destructive.
-   [ ] `dev-gtt` snapshot works.
-   [ ] recovery is tested.
-   [ ] CI mode works.
-   [ ] JSON output works for defined commands.
-   [ ] integration tests cover multi-ADE.
-   [ ] no GTT methodology is duplicated in Go.

------------------------------------------------------------------------

# 102. Definition of Done --- Architecture

The implementation must pass this architectural review:

### Question 1

Can GTT methodology change without changing Go code?

Expected:

``` text
YES
```

### Question 2

Can Bootstrap add an ADE integration without implementing GTT logic in
the CLI?

Expected:

``` text
YES
```

### Question 3

Can the CLI install multiple ADE integrations?

Expected:

``` text
YES
```

### Question 4

Can the user select a Primary ADE explicitly?

Expected:

``` text
YES
```

### Question 5

Can the user start with no design document?

Expected:

``` text
YES → questionnaire
```

### Question 6

Can the project evolve with new documentation?

Expected:

``` text
YES
```

### Question 7

Can a clean QA/PRO artifact be produced without destroying the GTT
project?

Expected:

``` text
YES
```

### Question 8

Can GTT be removed while preserving a portable recovery snapshot?

Expected:

``` text
YES
```

### Question 9

Can the CLI operate without an LLM?

Expected:

``` text
YES
```

### Question 10

Does the CLI contain the GTT engine?

Expected:

``` text
NO
```

------------------------------------------------------------------------

# 103. First Implementation Milestone

Build in this order.

## Phase 1 --- Foundation

``` text
Go module
Cobra
configuration
logging
output
exit codes
process executor
```

## Phase 2 --- Bootstrap

``` text
manifest parser
resolver
compatibility
installer
integrity
```

## Phase 3 --- Project

``` text
root detection
existing GTT detection
source discovery
source selection
language selection
questionnaire materialization
```

## Phase 4 --- ADE

``` text
ADE detection
Primary ADE selection
multi-ADE installation
ADE contract
ADE invocation
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
export --clean
clean
restore
```

## Phase 7 --- Hardening

``` text
security
transactionality
integration tests
CI
JSON
documentation
release
```

------------------------------------------------------------------------

# 104. Recommended First Repository Commit

The first implementation commit should establish only the foundation:

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

Then incrementally add Bootstrap integration.

Do not create placeholder governance packages.

------------------------------------------------------------------------

# 105. Recommended Initial Cobra Tree

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

Keep the tree small.

------------------------------------------------------------------------

# 106. Implementation Rule for Claude

When implementing this specification:

1.  Start from the repository root.
2.  Inspect the actual Bootstrap contract before inventing interfaces.
3.  Do not recreate Bootstrap logic in Go.
4.  Prefer invoking existing Bootstrap scripts/contracts.
5.  Make compatibility explicit.
6.  Implement multi-ADE from the start.
7.  Treat Primary ADE as explicit user state.
8.  Require explicit initial source selection.
9.  Request and materialize the Bootstrap-provided questionnaire only
    when appropriate.
10. Keep architecture/design decisions out of the CLI.
11. Implement clean export without destroying development state.
12. Implement `dev-gtt` as a portable recovery snapshot.
13. Add tests before broadening the command surface.
14. Keep the CLI deterministic.
15. Do not add features simply because they are technically possible.

------------------------------------------------------------------------

# 107. Implementation Rule for Bootstrap Evolution

If the CLI discovers that a capability required by this specification is
missing from Bootstrap, do not duplicate that capability in Go
automatically.

Instead determine:

``` text
Does this belong in Bootstrap?
```

If yes:

``` text
extend Bootstrap contract
```

Then:

``` text
update CLI adapter
```

This preserves the architecture:

``` text
Bootstrap = intelligence + governance + implementation
CLI       = operational interface
```

------------------------------------------------------------------------

# 108. Final Architectural Contract

The entire system should be understandable through this model:

``` text
                         GTT METHOD
                              │
                              ▼
                     GTT BOOTSTRAP
                              │
             ┌────────────────┼────────────────┐
             │                │                │
            Core            Domain          ADEs
             │                │                │
             └────────────────┼────────────────┘
                              │
                              ▼
                          GTT CLI
                              │
        ┌─────────────┬───────┼────────┬─────────────┐
        │             │       │        │             │
      Init          Status  Validate  Update      Export
        │
        ▼
  Project Setup
        │
        ├── Initial Sources
        ├── Language
        ├── Primary ADE
        └── Multi-ADE
        │
        ▼
     ADE / Agents
        │
        ▼
      Human
        │
        ▼
      Freeze
```

The most important invariant is:

``` text
The CLI must never become the mind of GTT.
```

The Bootstrap is the mind and implementation.

The CLI is the professional operational doorway into that system.

------------------------------------------------------------------------

# 109. Immediate Engineering Task

Start implementation from zero with:

``` text
1. Create Go module.
2. Add Cobra.
3. Create root command.
4. Create command tree.
5. Create process execution abstraction.
6. Create project-root detector.
7. Create Bootstrap manifest loader.
8. Create Bootstrap compatibility resolver.
9. Create ADE detector abstraction.
10. Implement `gtt init` interactively.
11. Implement source-document selection.
12. Implement questionnaire fallback.
13. Implement multi-ADE installation.
14. Implement Primary ADE state.
15. Implement Bootstrap validation.
16. Implement ADE bootstrap contract handoff.
17. Implement status/inspect/validate.
18. Add integration fixtures.
```

Do not implement architecture reasoning, ADR reasoning, proposal
reasoning, evidence reasoning or GTT methodology inside the CLI.

Those capabilities belong to the Bootstrap.

------------------------------------------------------------------------

# 110. Final Rule

If a future developer asks:

> "Should this logic be added to the CLI?"

Apply this test:

``` text
Does the logic explain how GTT thinks,
governs, reasons, evaluates evidence,
designs architecture or makes decisions?

        YES
        ↓
    Bootstrap

Does the logic only determine how to install,
select, invoke, update, validate, export,
recover or operate the Bootstrap?

        YES
        ↓
       CLI
```

This boundary must survive the entire lifetime of GTT CLI v1.x.
