# GTT — Developer Friction Minimization & Operational Automation

> **GTT Canonical Governance:** https://github.com/GTT-Community/gtt-method/blob/main/GTT-CANONICAL-v2.1.md

## 1. Purpose

GTT must minimize as much as possible the interruption and operational burden imposed on the developer while preserving the full level of governance.

> **GTT must govern the development without unnecessarily interrupting the developer.**

This is a transversal rule of GTT and applies to Methodology, Bootstrap, CLI and operational workflows.

## 2. Core Principle

> **The developer decides what only the developer can decide. GTT does everything else.**

Therefore:
- deterministic operations should be automated;
- already-defined policy must not be repeatedly requested from the developer;
- human intervention is required only when a genuine human decision or authorization is necessary;
- governance must not be weakened to achieve lower friction;
- successful automated operations should not create unnecessary stops.

### Light Method

```text
Less human intervention
+
More deterministic automation
+
Same governance
```

Light Method does **not** mean less validation, traceability, or governance.

## 3. Decision Rule Before Any Interruption

Before asking the developer anything, GTT must determine whether the operation can be resolved objectively from:
1. governed context;
2. active methodology plan;
3. project policies;
4. actual project state;
5. deterministic validation rules.

### Determinable operation

```text
ANALYZE
  ↓
VALIDATE
  ↓
DETERMINE
  ↓
EXECUTE
  ↓
VALIDATE RESULT
  ↓
CONTINUE
```

No unnecessary confirmation should be requested.

### Non-determinable operation

```text
ANALYZE
  ↓
IDENTIFY DECISION
  ↓
EXPLAIN BRIEFLY
  ↓
REQUEST HUMAN DECISION / AUTHORIZATION
  ↓
CONTINUE
```

## 4. Automatic Operations

When safe and deterministic, GTT should automatically handle:
- occupied artifact IDs;
- next valid free IDs;
- deterministic internal references;
- derived index rebuilding;
- provenance and structure validation;
- integrity checks;
- deterministic context synchronization;
- deterministic repairs;
- continuation to the next planned task;
- post-operation validation.

Example:

```text
ADR-002 requested
      ↓
ADR-002 occupied
      ↓
Find next valid ID
      ↓
ADR-005
      ↓
Update references
      ↓
Rebuild index
      ↓
Validate integrity
      ↓
Continue
```

The developer should not be interrupted for each mechanical step.

## 5. Protected Operations and `.sh` Scripts

When an operation affects a protected or governed artifact and requires human authorization, GTT must use the proposal/script mechanism.

This includes, when applicable:
- approve;
- promote;
- advance a governed stage;
- copy;
- move;
- delete;
- overwrite;
- modify;
- apply changes to protected artifacts.

When authorization is required, GTT must generate or provide the appropriate `.sh` script instead of asking the developer to reconstruct the command manually.

### Required flow

```text
ANALYZE
  ↓
PREPARE
  ↓
GENERATE / IDENTIFY .sh
  ↓
BRIEF AUTHORIZATION REQUEST
  ↓
EXECUTE
  ↓
VALIDATE RESULT
  ↓
CONTINUE
```

The script must:
- be directly executable;
- contain the complete operation;
- perform required precondition checks;
- fail explicitly when the expected state is not present;
- avoid ambiguous manual steps;
- be safe to execute from the documented project root.

Example:

```text
✓ ADR-009 is ready for promotion.
⚠ Requires authorization because it modifies governed context.

Run from project root:
bash gtt-domain/proposals/apply-ADR-009-base-evolution-alignment.sh

After execution, GTT validates the result and continues.
```

## 6. Do Not Turn Deterministic Work Into Conversation

Avoid asking:

```text
Do you want me to find another ID?
Do you want me to update the references?
Do you want me to rebuild the index?
Do you want me to validate?
Do you want me to continue?
```

When these operations are safe and deterministic, GTT performs them.

## 7. STOP Rule

`STOP` is not a general precaution mechanism.

GTT should stop only for:
- a genuine architectural or semantic conflict;
- a decision only the developer can make;
- explicit authorization required for a protected operation;
- a safety/integrity condition preventing safe continuation;
- unresolved evidence required to continue correctly.

GTT must not stop merely because it reached an intermediate mechanical step.

## 8. Minimal Reporting

GTT reports must be **very brief by default**.

The normal report communicates only:
1. what was done;
2. result;
3. whether the developer must act;
4. next step when relevant.

Example:

```text
✓ T03 completed.
✓ Tests: 46 passed, 1 skipped.
⚠ ADR-009 requires authorization for promotion.
→ bash gtt-domain/proposals/apply-ADR-009-base-evolution-alignment.sh
```

Do not include long explanations, internal reasoning, or operational history unless requested.

### Detail on demand

Detailed information must remain available when requested.

Possible CLI mechanisms include:

```text
gtt status --verbose
gtt explain
gtt report --details
```

The exact commands are implementation decisions.

> **Minimal report by default. Detailed information on demand.**

## 9. Continuity

After a successful automatic operation, GTT should continue the planned workflow.

Avoid:

```text
✓ Index rebuilt.
Do you want me to continue?
```

Prefer:

```text
✓ Index rebuilt.
→ Continuing with T04.
```

## 10. Bootstrap Responsibility

Bootstrap must configure this behavior as part of the selected methodology plan.

Example:

```yaml
developer_experience:
  minimize_interruption: true
  automatic_deterministic_operations: true
  concise_reports: true
  details_on_demand: true
```

Bootstrap configures the policy.

CLI executes the policy.

The methodology defines the principle.

Bootstrap must not duplicate CLI execution logic.

## 11. Governance Boundary

Reducing developer interaction must never mean bypassing governance.

```text
                    GTT
                     │
          ┌──────────┴──────────┐
          │                     │
   Deterministic            Semantic
   operation                decision
          │                     │
          ▼                     ▼
       GTT acts             Human decides
          │                     │
          └──────────┬──────────┘
                     ▼
                 Validation
```

Examples of deterministic responsibility:
- file existence;
- line references;
- duplicate IDs;
- index consistency;
- structural validation;
- known reference updates;
- deterministic artifact movement;
- lifecycle transitions permitted by policy.

Examples of human responsibility:
- architectural choice;
- business requirement interpretation;
- unresolved semantic conflict;
- substantive contract change;
- decisions not defined by existing policy.

## 12. Implementation Task for Claude

Implement this rule across the current GTT Bootstrap and CLI implementation.

### First analyze

Before changing code:
1. inspect Bootstrap behavior;
2. inspect Method Plans;
3. inspect CLI behavior;
4. inspect proposal and `.sh` mechanisms;
5. identify unnecessary developer interruptions;
6. identify deterministic operations that can be automated;
7. preserve existing governance behavior.

### Then implement

Implement:
- automatic deterministic operations;
- minimum necessary human intervention;
- `.sh` generation/provision for protected operations requiring authorization;
- direct and executable commands;
- precondition validation;
- post-operation validation;
- automatic workflow continuation;
- genuine STOP conditions only;
- concise default reports;
- detailed reporting only on demand;
- compatibility with Light, Medium, Hard and Team plans.

### Critical requirement

Do **not** make every operation a question to the developer.

If the active directive and project state objectively determine the action, GTT must perform it.

### Testing

Before finishing:
- run relevant tests;
- verify generated scripts are executable;
- verify commands work from the documented project root;
- verify deterministic operations no longer produce unnecessary prompts;
- verify protected operations still require authorization;
- verify Light Method has lower operational friction without reducing governance;
- verify reports are concise.

Do not create a commit.

## 13. Final Principle

> **GTT should make governance almost invisible when it can safely handle it automatically, and clearly visible only when human judgment or authorization is genuinely required.**

> **Minimum developer friction. Maximum deterministic automation. Full governance.**
