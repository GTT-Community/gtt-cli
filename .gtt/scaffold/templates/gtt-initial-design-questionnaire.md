# GTT Initial Design Questionnaire

> **GTT Governance Canonical:**
> https://github.com/GTT-Community/gtt-method/blob/main/GTT-CANONICAL-v2.1.md

## Purpose

This questionnaire is the **initial design elicitation instrument** used
by GTT when a project does not yet have sufficient design documentation.

It is part of the **GTT Bootstrap**.

It is not a governance engine, an architecture decision record, or a
final architecture specification.

Its purpose is to help the Primary ADE work with the human to
progressively establish a **Minimum Viable Governed Design**.

The preferred operating model is:

``` text
GTT CLI
   ↓
Bootstrap detects no sufficient design source
   ↓
Questionnaire is materialized
   ↓
Primary ADE opens the questionnaire
   ↓
ADE conducts an iterative interview with the human
   ↓
Human provides answers
   ↓
ADE writes the answers into this document
   ↓
ADE identifies missing information / conflicts / decisions
   ↓
Human clarifies
   ↓
Minimum viable design emerges
   ↓
Human reviews
   ↓
GTT governance process continues
```

The human should **not be required to understand or manually complete
every section before the ADE can help**.

The ADE should guide the human.

------------------------------------------------------------------------

# 1. Operating Contract for the ADE

When this questionnaire is used, the Primary ADE MUST:

1.  Read this questionnaire before beginning the interview.
2.  Read the applicable GTT Bootstrap instructions and `AGENTS.md`.
3.  Inspect the project before asking questions that can be answered
    from existing project evidence.
4.  Ask the human only for information that is not sufficiently
    established by available evidence.
5.  Work progressively rather than presenting a large questionnaire all
    at once.
6.  Populate this document as the conversation progresses.
7.  Clearly distinguish:
    -   information provided by the human;
    -   information found in project/source material;
    -   information that is unknown;
    -   conflicts;
    -   proposals;
    -   decisions requiring human confirmation.
8.  Never invent architectural decisions to complete a section.
9.  Never silently convert an ADE inference into a human decision.
10. Ask for explicit human confirmation when an architectural decision
    is required.
11. Keep unanswered questions visible.
12. Revisit earlier answers when new information creates a conflict.
13. Preserve the user's wording when it represents an explicit
    requirement or constraint.
14. Avoid unnecessary questions when the project already provides
    sufficient evidence.
15. Stop when a sufficient Minimum Viable Governed Design has been
    established or when the human explicitly decides to continue later.

The ADE is an **elicitation and documentation assistant**.

The human remains the decision authority.

------------------------------------------------------------------------

# 2. Important Rule: Do Not Interrogate the User With the Whole Form

The ADE SHOULD NOT begin by asking the user to answer all sections.

Instead, conduct an adaptive interview.

Recommended sequence:

``` text
Understand the project
        ↓
Understand the problem
        ↓
Understand users / consumers
        ↓
Understand scope
        ↓
Understand main capabilities
        ↓
Understand important constraints
        ↓
Understand known architecture
        ↓
Understand integrations and data
        ↓
Understand security / non-functional requirements
        ↓
Understand deployment / operations
        ↓
Identify unknowns
        ↓
Identify decisions requiring human confirmation
        ↓
Review the resulting Minimum Viable Design
```

The exact sequence may change according to the project.

The ADE should follow evidence and conversation rather than mechanically
following the numbered sections.

------------------------------------------------------------------------

# 3. Evidence and Attribution

Each important entry should preserve where the information came from.

Use the GTT provenance conventions when applicable:

``` text
[FUENTE: archivo:línea]
[VACÍO]
[CONFLICTO]
[PROPUESTA]
```

Examples:

``` text
Users:
[FUENTE: docs/product.md:42-48]
The system is used by internal operations teams.
```

``` text
Database:
[VACÍO]
The database technology has not yet been decided.
```

``` text
Architecture:
[PROPUESTA]
The ADE proposes a modular monolith as an option.
Human confirmation required.
```

``` text
Authentication:
[CONFLICTO]
requirements.md describes OAuth2 while architecture.md describes SAML.
Human clarification required.
```

The ADE must not use a proposal marker to disguise a decision.

------------------------------------------------------------------------

# 4. Project Identity

## 4.1 Project name

**Question for the human:**

> What is the name of the project?

**Answer:**

<!-- ADE populates this section -->

## 4.2 Project purpose

**Question:**

> What are you trying to build, and why does it need to exist?

**Answer:**

<!-- ADE populates this section -->

## 4.3 Problem

**Question:**

> What problem does this project solve?

**Answer:**

<!-- ADE populates this section -->

## 4.4 Desired outcome

**Question:**

> What should be possible when this project is successfully delivered?

**Answer:**

<!-- ADE populates this section -->

------------------------------------------------------------------------

# 5. Scope

## 5.1 In scope

**Question:**

> What must this project include?

**Answer:**

<!-- ADE populates this section -->

## 5.2 Out of scope

**Question:**

> What should explicitly NOT be part of this project?

**Answer:**

<!-- ADE populates this section -->

## 5.3 Scope boundaries

**Question:**

> Are there organizational, technical, product, or operational
> boundaries that must be respected?

**Answer:**

<!-- ADE populates this section -->

------------------------------------------------------------------------

# 6. Users and Consumers

## 6.1 Primary users

**Question:**

> Who will use this system directly?

**Answer:**

<!-- ADE populates this section -->

## 6.2 External consumers

**Question:**

> Will other systems, customers, partners, or external organizations
> consume this system?

**Answer:**

<!-- ADE populates this section -->

## 6.3 Internal consumers

**Question:**

> Which internal teams or systems depend on it?

**Answer:**

<!-- ADE populates this section -->

------------------------------------------------------------------------

# 7. Main Capabilities

The ADE should identify the main capabilities without prematurely
designing the implementation.

**Questions:**

> What are the most important things the system must do?

> What are the critical user or business flows?

> Which capabilities are mandatory for the first version?

## Capabilities

<!-- ADE populates this section -->

------------------------------------------------------------------------

# 8. Functional Requirements

The ADE should progressively identify concrete functional requirements.

For each requirement:

``` text
ID:
Description:
Source:
Priority:
Notes:
```

## Requirements

<!-- ADE populates this section -->

------------------------------------------------------------------------

# 9. Non-Functional Requirements

The ADE should ask about the requirements that can materially affect
architecture.

## 9.1 Performance

Questions:

> Are there response-time expectations?

> Are there throughput or concurrency requirements?

**Answer:**

<!-- ADE populates this section -->

## 9.2 Availability

Questions:

> Does the system need a specific availability target?

> Are there critical operating windows?

**Answer:**

<!-- ADE populates this section -->

## 9.3 Scalability

Questions:

> How is expected usage likely to grow?

> Are there known scaling requirements?

**Answer:**

<!-- ADE populates this section -->

## 9.4 Reliability

Questions:

> What happens if a dependency fails?

> Are retries, recovery, or disaster recovery requirements known?

**Answer:**

<!-- ADE populates this section -->

## 9.5 Compliance

Questions:

> Are there regulatory, legal, contractual, or organizational
> requirements?

**Answer:**

<!-- ADE populates this section -->

------------------------------------------------------------------------

# 10. Security

The ADE should determine whether security requirements materially
influence the design.

## 10.1 Identity

> Who are the users or systems that must be identified?

**Answer:**

<!-- ADE populates this section -->

## 10.2 Authentication

> How are users or systems expected to authenticate, if known?

**Answer:**

<!-- ADE populates this section -->

## 10.3 Authorization

> Are there roles, permissions, or access boundaries?

**Answer:**

<!-- ADE populates this section -->

## 10.4 Sensitive data

> Does the system handle sensitive, confidential, personal, financial,
> or regulated data?

**Answer:**

<!-- ADE populates this section -->

## 10.5 Secrets

> Are there known requirements for credentials, keys, certificates, or
> secret management?

**Answer:**

<!-- ADE populates this section -->

------------------------------------------------------------------------

# 11. Data

## 11.1 Main data

**Question:**

> What important information does the system create, read, update, or
> process?

**Answer:**

<!-- ADE populates this section -->

## 11.2 Data ownership

**Question:**

> Who owns the important data?

**Answer:**

<!-- ADE populates this section -->

## 11.3 Data lifecycle

**Question:**

> Are there retention, archival, deletion, or audit requirements?

**Answer:**

<!-- ADE populates this section -->

## 11.4 Known storage technology

**Question:**

> Is a database or storage technology already mandated or known?

**Answer:**

<!-- ADE populates this section -->

If not known:

``` text
[VACÍO]
Storage technology not established.
```

Do not invent one.

------------------------------------------------------------------------

# 12. Integrations and External Systems

## 12.1 Existing systems

**Question:**

> Which existing systems must this project integrate with?

**Answer:**

<!-- ADE populates this section -->

## 12.2 APIs

**Question:**

> Which APIs must be consumed or exposed?

**Answer:**

<!-- ADE populates this section -->

## 12.3 Messaging / events

**Question:**

> Does the system need messaging, asynchronous processing, events,
> queues, or streaming?

**Answer:**

<!-- ADE populates this section -->

Do not introduce Kafka, RabbitMQ, or another technology unless supported
by evidence or explicitly proposed and marked as such.

------------------------------------------------------------------------

# 13. Architecture

This section is intentionally divided into **known decisions** and
**unknown decisions**.

The ADE must not manufacture architecture.

## 13.1 Known architectural decisions

**Question:**

> Has any architecture already been decided?

**Answer:**

<!-- ADE populates this section -->

For each known decision:

``` text
Decision:
Reason:
Source:
Status:
```

## 13.2 Existing architecture

If an existing system or repository exists:

> What architecture already exists?

<!-- ADE inspects project and populates this section -->

## 13.3 Preferred architecture

**Question:**

> Do you already have an architectural direction in mind?

The user may describe an intention without it automatically becoming a
final decision.

<!-- ADE populates this section -->

## 13.4 Architecture unknowns

<!-- ADE populates this section -->

Use:

``` text
[VACÍO]
```

when architecture is genuinely unknown.

------------------------------------------------------------------------

# 14. Technology

## 14.1 Languages

> Are there mandated or preferred programming languages?

**Answer:**

<!-- ADE populates this section -->

## 14.2 Frameworks

> Are there mandated or preferred frameworks?

**Answer:**

<!-- ADE populates this section -->

## 14.3 Database / storage

> Is the storage technology known?

**Answer:**

<!-- ADE populates this section -->

## 14.4 Cloud

> Is a cloud provider required or preferred?

**Answer:**

<!-- ADE populates this section -->

## 14.5 Existing technology constraints

<!-- ADE populates this section -->

------------------------------------------------------------------------

# 15. Deployment and Infrastructure

## 15.1 Environments

> Which environments are required?

Examples:

``` text
development
QA
staging
production
```

**Answer:**

<!-- ADE populates this section -->

## 15.2 Containers

> Are containers required or preferred?

**Answer:**

<!-- ADE populates this section -->

## 15.3 Kubernetes

> Is Kubernetes required or already present?

**Answer:**

<!-- ADE populates this section -->

## 15.4 CI/CD

> What CI/CD platform or process is required?

**Answer:**

<!-- ADE populates this section -->

## 15.5 Infrastructure as Code

> Is Infrastructure as Code required or already present?

**Answer:**

<!-- ADE populates this section -->

------------------------------------------------------------------------

# 16. Observability

## 16.1 Logging

> What logging requirements exist?

**Answer:**

<!-- ADE populates this section -->

## 16.2 Metrics

> What metrics are important?

**Answer:**

<!-- ADE populates this section -->

## 16.3 Tracing

> Is distributed tracing required?

**Answer:**

<!-- ADE populates this section -->

## 16.4 Alerting

> What failures or conditions must generate alerts?

**Answer:**

<!-- ADE populates this section -->

------------------------------------------------------------------------

# 17. Constraints

The ADE should actively look for constraints because constraints often
influence architecture more strongly than preferences.

Possible categories:

``` text
Budget
Timeline
Team
Skills
Technology
Cloud
Security
Compliance
Existing systems
Vendor
Licensing
Infrastructure
Organizational
```

## Constraints

<!-- ADE populates this section -->

------------------------------------------------------------------------

# 18. Existing Documentation

The CLI may have provided initial source documents.

The ADE should record them here.

``` text
Document:
Path:
Purpose:
Version/date if known:
```

## Documents

<!-- ADE populates this section -->

Important:

The existence of a document does not automatically mean that every
statement in it is a governed decision.

------------------------------------------------------------------------

# 19. Unknowns

The ADE must maintain a visible list of relevant unknowns.

Do not hide unanswered questions.

For each unknown:

``` text
ID:
Topic:
What is unknown:
Why it matters:
Blocking:
```

## Unknowns

<!-- ADE populates this section -->

Use:

``` text
[VACÍO]
```

where appropriate.

------------------------------------------------------------------------

# 20. Conflicts

When sources or answers disagree, the ADE must expose the conflict.

It must not silently choose a winner.

For each conflict:

``` text
ID:
Topic:
Source A:
Source B:
Conflict:
Human clarification required:
```

## Conflicts

<!-- ADE populates this section -->

Use:

``` text
[CONFLICTO]
```

------------------------------------------------------------------------

# 21. Proposals

The ADE may propose options when the information is not established.

A proposal is not a decision.

For each proposal:

``` text
ID:
Topic:
Proposal:
Reasoning:
Evidence:
Alternatives:
Human decision required:
```

## Proposals

<!-- ADE populates this section -->

Use:

``` text
[PROPUESTA]
```

Never write a proposal as if it were an approved architectural decision.

------------------------------------------------------------------------

# 22. Decisions Requiring Human Confirmation

At the end of the interview, the ADE should identify decisions that
require explicit human confirmation.

Examples:

``` text
Architecture style
Database
Authentication mechanism
Cloud provider
Deployment model
Messaging technology
External integration strategy
Security model
```

## Decisions

<!-- ADE populates this section -->

For each:

``` text
Decision ID:
Topic:
Options:
Evidence:
Proposal, if any:
Human decision:
Confirmed:
Date:
```

The ADE must not fill `Human decision` on behalf of the human.

------------------------------------------------------------------------

# 23. Minimum Viable Governed Design

The objective is not to complete every field.

The objective is to establish enough reliable information to begin the
governed design process.

The ADE should determine whether the project has enough information to
establish:

``` text
Project purpose
Problem
Scope
Main users/consumers
Main capabilities
Important constraints
Relevant non-functional requirements
Known integrations
Known technology constraints
Known architecture decisions
Important unknowns
Important conflicts
Human decisions still required
```

## Readiness

``` text
Status:
```

Possible values:

``` text
READY
READY_WITH_OPEN_ITEMS
NOT_READY
```

The ADE must explain why.

------------------------------------------------------------------------

# 24. Human Review

Before considering the questionnaire complete, the ADE should summarize
the current understanding for the human.

The review should answer:

``` text
What are we building?
Why?
For whom?
What is in scope?
What is out of scope?
What are the important requirements?
What architecture is already known?
What remains unknown?
What conflicts exist?
What decisions require confirmation?
```

The human must have an opportunity to correct the document.

------------------------------------------------------------------------

# 25. Completion Rule

The questionnaire is complete when:

1.  the human has reviewed the resulting understanding;
2.  important unknowns are visible;
3.  important conflicts are visible;
4.  proposals are clearly marked;
5.  decisions are not fabricated;
6.  the project has enough information to continue through the GTT
    governed design process.

Completion does **not** mean that every question has an answer.

An explicit unknown is better than an invented answer.

------------------------------------------------------------------------

# 26. What Happens Next

After human review, the ADE should continue according to the installed
GTT Bootstrap process.

The questionnaire remains **source material**.

It does not automatically become governed architecture.

The transition is conceptually:

``` text
Questionnaire
     ↓
Source material
     ↓
Grounding / evidence
     ↓
GTT design process
     ↓
Proposals / decisions
     ↓
Human ratification
     ↓
Governed context
     ↓
Freeze
```

The exact transition mechanism is defined by the GTT Bootstrap.

------------------------------------------------------------------------

# 27. ADE Behavior Summary

The preferred behavior is:

``` text
Human:
"I want to build an API for processing customer orders."

ADE:
"Understood. Before we design the architecture, I want to establish
the basic project context. I found README.md and docs/orders.md.
I'll use those as initial source material. I'll ask only for
information that is still missing."

        ↓

ADE reads evidence.

        ↓

ADE asks:
"Who are the consumers of the API?"

Human answers.

        ↓

ADE updates this document.

        ↓

ADE asks:
"I found that orders are already stored in PostgreSQL in the
existing project. Is PostgreSQL still the intended persistence
technology?"

Human confirms or corrects.

        ↓

ADE updates this document.

        ↓

ADE identifies:
"[VACÍO] Authentication mechanism is not established."

        ↓

ADE asks the human.

        ↓

If the human asks:
"What would you recommend?"

ADE may provide a marked proposal:

"[PROPUESTA] ..."

        ↓

Human decides.

        ↓

ADE records the human decision.

        ↓

Continue until Minimum Viable Governed Design is established.
```

The important property is:

``` text
ADE conducts the interview.
Human supplies and confirms information.
ADE documents the result.
ADE does not invent authority.
```

------------------------------------------------------------------------

# 28. Anti-Patterns

The ADE must NOT:

### 28.1 Ask everything at once

Bad:

``` text
Please answer these 80 questions.
```

### 28.2 Invent missing architecture

Bad:

``` text
The project will use microservices and PostgreSQL.
```

when no source or human decision supports it.

### 28.3 Treat inference as decision

Bad:

``` text
Since this is an API, REST is the chosen architecture.
```

### 28.4 Hide uncertainty

Bad:

``` text
Database: PostgreSQL
```

when the database is unknown.

Correct:

``` text
[VACÍO]
Database technology not established.
```

### 28.5 Resolve conflicts silently

Bad:

``` text
Document A says OAuth2.
Document B says SAML.
The ADE chooses OAuth2.
```

Correct:

``` text
[CONFLICTO]
Authentication mechanism differs between sources.
Human clarification required.
```

### 28.6 Turn proposals into decisions

Bad:

``` text
Database: PostgreSQL
```

after the ADE merely recommended it.

Correct:

``` text
[PROPUESTA]
PostgreSQL proposed.

Human decision:
[PENDING]
```

------------------------------------------------------------------------

# 29. Versioning

This questionnaire is a Bootstrap artifact.

Its version must be associated with the Bootstrap version that provides
it.

The CLI must not maintain an independent questionnaire version.

Example:

``` text
Bootstrap: 2.1.x
Questionnaire contract: bootstrap-defined
```

When the questionnaire changes materially, update the Bootstrap.

------------------------------------------------------------------------

# 30. Ownership

``` text
Owner:
GTT Bootstrap

Consumed by:
GTT CLI

Executed by:
Primary ADE

Filled through:
Human + ADE interaction

Authority:
Human

Final governed architecture:
GTT governed design process
```

------------------------------------------------------------------------

# 31. Core Principle

The questionnaire exists to solve one problem:

> **Start a GTT project without requiring the human to already possess a
> complete architecture document.**

The preferred interaction is therefore:

``` text
No design document
        ↓
GTT Bootstrap
        ↓
Initial Design Questionnaire
        ↓
Primary ADE
        ↓
Guided conversation with human
        ↓
Progressive document population
        ↓
Minimum Viable Governed Design
        ↓
Human review
        ↓
GTT governed design
```

The ADE should make the process feel like a professional architecture
discovery session, while preserving GTT's evidence and governance
boundaries.
