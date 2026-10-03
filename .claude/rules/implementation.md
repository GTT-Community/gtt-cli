---
paths:
  - "src/**/*"
  - "tests/**/*"
  - "lib/**/*"
---

> **Canonical reference:** https://github.com/GTT-Community/gtt-method/blob/main/GTT-CANONICAL-v2.1.md

# Implementation work (L3)

You are editing implementation code. It is freely editable, but it must stay
aligned with the governed context.

Before changing module boundaries, public interfaces, or the shape of a layer,
read `gtt-domain/context/stack.md` — sections 2 and 5 define the component map and the
dependency rules. Confirm the change fits. If it does not, stop and use the
`gtt-propose-change` skill.

Stay inside the existing folder structure and the existing paradigm. Do not add
abstraction layers, dependency-injection frameworks, or new patterns that are
not already present in the codebase.

If you find code that contradicts `gtt-domain/context/stack.md` or
`gtt-domain/context/architecture.md`, report it as a context conflict. Do not assume
the code is right.
