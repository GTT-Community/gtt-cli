# GTT --- Plan de Convergencia y Recuperación Funcional CDAD

> **Referencia canónica GTT:**
> https://github.com/GTT-Community/gtt-method/blob/main/GTT-CANONICAL-v2.1.md

**Estado:** Documento de trabajo\
**Propósito:** llevar GTT a una versión estable que conserve las
capacidades valiosas de CDAD, recupere las que se hayan diluido o
perdido y, al mismo tiempo, preserve las mejoras estructurales y
operativas introducidas por GTT.

------------------------------------------------------------------------

## 1. Objetivo

Este documento no propone regresar de GTT a CDAD.

El objetivo es realizar una **convergencia funcional**:

> **GTT final = capacidades esenciales de CDAD + mejoras de GTT +
> integración coherente entre ambas.**

La comparación realizada entre las versiones entregadas muestra que GTT
creció de manera importante y añadió capacidades reales que no deben
sacrificarse. Sin embargo, algunas ideas de CDAD que eran especialmente
fuertes quedaron menos explícitas, dispersas o no están todavía
materializadas con la misma precisión operacional.

El trabajo debe reparar eso antes de considerar una versión final.

### Resultado esperado

Al finalizar este trabajo, GTT debe:

1.  Mantener su separación actual entre **Engine**, **Governed Domain**
    y **ADE adapters**.
2.  Mantener Artifact Identity y Technical Index.
3.  Mantener GTTGuard y la frontera de promoción humana.
4.  Mantener Session Memory y los adaptadores ADE.
5.  Mantener query/retrieval, status, reconciliation y validation.
6.  Recuperar explícitamente la **taxonomía de procedencia** de CDAD.
7.  Recuperar explícitamente la semántica **OPEN vs BLOCKING**.
8.  Recuperar la noción formal de **fuentes autorizadas, autoridad y
    precedencia**.
9.  Mantener y formalizar los distintos **modos de entrada/bootstrap**.
10. Separar correctamente **Working Preferences / Working Agreements**
    de Session Memory y del contexto gobernado.
11. Mantener las dos confirmaciones humanas del bootstrap.
12. Preservar la frontera de evidencia: el razonamiento no debe recibir
    fuentes crudas cuando el modelo de gobierno exige dossier.
13. Evitar que los nuevos mecanismos dupliquen fuentes de verdad.
14. Integrar todo esto en una arquitectura GTT coherente, sin introducir
    una "capa CDAD" paralela.
15. Terminar con validación funcional reproducible en un repositorio Git
    real.

------------------------------------------------------------------------

# 2. Regla principal de evolución

No implementar una funcionalidad de CDAD simplemente porque existía en
CDAD.

Para cada capacidad se debe determinar una de estas categorías:

  -----------------------------------------------------------------------
  Categoría                           Tratamiento
  ----------------------------------- -----------------------------------
  `CONSERVADO`                        GTT ya lo hace correctamente; no
                                      duplicar

  `MEJORADO`                          GTT lo reemplazó por una solución
                                      superior; conservar GTT

  `REEMPLAZADO`                       Existe otra implementación con la
                                      misma intención y mejor diseño

  `DILUIDO`                           La capacidad sigue existiendo pero
                                      perdió claridad, fuerza o
                                      trazabilidad

  `PERDIDO`                           La capacidad existía en CDAD y ya
                                      no está suficientemente
                                      representada

  `NUEVO`                             Capacidad introducida por GTT y que
                                      CDAD no tenía

  `A RECUPERAR`                       Capacidad que debe reincorporarse
                                      explícitamente
  -----------------------------------------------------------------------

**Regla:** una capacidad `MEJORADA` o `REEMPLAZADA` nunca debe ser
reintroducida mediante una segunda implementación paralela.

------------------------------------------------------------------------

# 3. Lo que NO debe retroceder

Estas capacidades de GTT deben considerarse parte del núcleo moderno y
no deben sacrificarse para recuperar CDAD.

## 3.1 Separación Engine / Domain / ADE

Debe permanecer:

``` text
.gtt/          → Engine / machinery
gtt-domain/    → Governed Domain
ADE overlay    → integración con Claude/Kiro/Copilot/Codex
```

No volver a concentrar la maquinaria bajo `gtt-domain/`.

------------------------------------------------------------------------

## 3.2 Artifact Identity

Debe mantenerse como mecanismo de identidad estable de artefactos.

Objetivos:

-   detectar movimientos;
-   distinguir identidad lógica de path físico;
-   evitar pérdida de identidad por rename;
-   detectar artefactos retirados;
-   soportar reconciliación;
-   evitar que el path sea la única identidad.

No reemplazar Artifact Identity por mecanismos basados únicamente en
nombres de archivo.

------------------------------------------------------------------------

## 3.3 Technical Index

Debe permanecer como índice derivado.

Regla fundamental:

> El índice nunca es Source of Truth.

Debe poder:

-   reconstruirse;
-   actualizarse;
-   consultarse;
-   invalidarse;
-   compararse contra la identidad de artefactos.

------------------------------------------------------------------------

## 3.4 GTTGuard

GTTGuard representa una mejora importante respecto del modelo puramente
declarativo.

Debe permanecer como **control plane determinista**, especialmente para
proteger artefactos gobernados.

La recuperación de conceptos de CDAD no debe convertir GTTGuard
nuevamente en instrucciones textuales que dependan de que el agente
"recuerde" una regla.

------------------------------------------------------------------------

## 3.5 Human Promotion Boundary

Debe permanecer el principio:

``` text
AGENT
  ↓
ANALYSIS
  ↓
PROPOSAL
  ↓
PROMOTION PACKAGE
  ↓
HUMAN REVIEW
  ↓
EXPLICIT HUMAN EXECUTION
  ↓
GOVERNED CHANGE
```

El agente puede:

-   analizar;
-   proponer;
-   preparar ADR;
-   preparar archivos afectados;
-   preparar script de promoción;
-   explicar impacto.

El agente no debe convertir una propuesta en una modificación gobernada
por sí mismo.

------------------------------------------------------------------------

## 3.6 Session Memory

Session Memory es una evolución válida de GTT y no debe eliminarse para
recuperar el concepto de working preferences.

Debe distinguirse de:

-   governed context;
-   source evidence;
-   working preferences;
-   propuestas;
-   memoria operacional derivada.

Session Memory sirve para continuidad operacional.

No debe convertirse en una nueva Source of Truth arquitectónica.

------------------------------------------------------------------------

## 3.7 Status / Query / Reconcile / Validate

Los servicios actuales de GTT deben conservarse y evolucionar.

Especialmente:

``` text
gtt-status.sh
gtt-query.sh
gtt-index.sh
gtt-reconcile.sh
gtt-validate.sh
```

Estas capacidades representan una mejora operacional real respecto del
CDAD analizado.

------------------------------------------------------------------------

# 4. Capacidad CDAD que debe recuperarse: procedencia

Esta es una de las recuperaciones prioritarias.

CDAD tenía una definición muy fuerte:

``` text
[FUENTE]
[VACÍO]
[CONFLICTO]
[PROPUESTA]
```

La idea fundamental no es el texto exacto de las etiquetas.

La idea es:

> **Toda afirmación relevante debe poder distinguir entre evidencia
> recuperada, ausencia de evidencia, conflicto entre fuentes y contenido
> nuevo propuesto por el razonamiento.**

## 4.1 Modelo GTT propuesto

GTT debe incorporar una taxonomía equivalente y canónica.

Se puede conservar la nomenclatura CDAD si resulta conveniente:

``` text
[FUENTE]
[VACÍO]
[CONFLICTO]
[PROPUESTA]
```

o adoptar una nomenclatura equivalente si GTT tiene una razón
arquitectónica clara.

Lo importante es conservar la semántica.

### `[FUENTE]`

La afirmación proviene de evidencia recuperada.

Debe poder señalar:

-   fuente;
-   ubicación;
-   versión cuando corresponda;
-   referencia concreta.

### `[VACÍO]`

Las fuentes autorizadas no contienen la información requerida.

No significa que el agente pueda inventarla.

Significa explícitamente:

> la evidencia disponible no determina esta decisión.

### `[CONFLICTO]`

Dos o más fuentes autorizadas contienen afirmaciones incompatibles.

Debe preservarse la existencia del conflicto.

El grounding no debe ocultarlo simplemente escogiendo una versión.

### `[PROPUESTA]`

Contenido generado por razonamiento que no está respaldado directamente
por la evidencia.

Debe requerir la ruta de decisión correspondiente antes de convertirse
en contexto gobernado.

------------------------------------------------------------------------

# 5. Frontera de evidencia

La recuperación de procedencia debe mantener la idea más fuerte de
CDAD-THINK:

``` text
AUTHORIZED SOURCES
        ↓
GROUNDING
        ↓
EVIDENCE DOSSIER
        ↓
REASONING / WORKERS
        ↓
PROPOSALS
        ↓
HUMAN DECISION
```

No:

``` text
SOURCE → WORKER
```

cuando el régimen exige grounding.

## 5.1 Regla

El agente de grounding:

-   recupera;
-   cita;
-   declara ausencia;
-   expone conflictos;
-   respeta la autoridad y precedencia declaradas.

No debe decidir arquitectura por el worker.

El agente de razonamiento:

-   razona sobre el dossier;
-   identifica implicaciones;
-   produce propuestas;
-   puede proponer resolución de conflictos;
-   no convierte una propuesta en decisión.

La frontera debe quedar documentada en `AGENTS.md` y en la documentación
del Engine.

------------------------------------------------------------------------

# 6. OPEN vs BLOCKING

Esta es otra recuperación prioritaria.

CDAD tenía una distinción importante:

``` text
BLOCKING
OPEN
```

## 6.1 BLOCKING

Un vacío cuya resolución es necesaria antes de poder congelar
correctamente el diseño.

Características:

-   impide freeze;
-   representa una decisión necesaria;
-   debe ser resuelto por la ruta gobernada.

## 6.2 OPEN

Una decisión conocida como pendiente cuya resolución **no es necesaria
para el estado actual del diseño**.

Un `OPEN`:

-   puede atravesar el freeze;
-   queda visible;
-   no constituye autorización para modificar el diseño congelado;
-   puede resolverse posteriormente mediante la ruta ordinaria de
    cambio.

Regla crítica:

> `OPEN` significa "todavía no decidido", no "autorizado para cambiar".

------------------------------------------------------------------------

# 7. Alcance de un OPEN

El alcance debe quedar explícito.

Ejemplo:

``` text
Cache = OPEN
```

significa que la decisión sobre cache permanece abierta.

No significa:

``` text
Cache = OPEN
→ introducir libremente Redis
→ introducir un event bus
→ modificar la arquitectura
```

La resolución de un OPEN fuera de su perímetro debe seguir el proceso
normal:

``` text
change request
→ analysis
→ proposal
→ human decision
→ promotion
→ new freeze
```

Esto debe quedar formalizado en GTT.

------------------------------------------------------------------------

# 8. Visibilidad de OPEN

Los OPEN no deben existir solamente como texto perdido.

Deben ser visibles como estado gobernado.

Como mínimo:

-   contexto afectado;
-   `stack.md`;
-   status;
-   validation;
-   session/status derivado cuando corresponda.

El sistema debe poder responder:

``` text
¿Cuántos OPEN existen?
¿Cuáles?
¿Dónde están?
¿Cuáles están dentro de su alcance?
¿Cuáles siguen sin resolver?
```

------------------------------------------------------------------------

# 9. Resolución de OPEN

Cuando un OPEN se resuelve debe quedar trazabilidad:

``` text
OPEN original
      ↓
decisión humana
      ↓
ADR / cambio gobernado
      ↓
contexto actualizado
      ↓
nuevo freeze
```

No basta con reemplazar el texto `OPEN`.

Debe quedar evidencia de que:

-   originalmente estaba abierto;
-   quién/qué decisión lo resolvió;
-   cuándo;
-   mediante qué cambio;
-   qué artefactos resultaron afectados.

------------------------------------------------------------------------

# 10. Source Manifest / autoridad / precedencia

Esta capacidad debe recuperarse explícitamente.

Cuando existen múltiples fuentes, GTT debe poder representar:

``` text
SOURCE
VERSION
AUTHORITY
PRECEDENCE
STATUS
```

Ejemplo conceptual:

``` yaml
sources:
  - id: architecture-spec
    path: docs/architecture.md
    version: "2.1"
    authority: primary
    precedence: 1

  - id: requirements
    path: docs/requirements.md
    version: "1.7"
    authority: primary
    precedence: 2
```

La implementación final puede usar otro formato si el diseño GTT lo
requiere.

La intención es evitar que la precedencia quede determinada
accidentalmente por:

-   orden de lectura;
-   orden de archivos;
-   orden del prompt;
-   orden de recuperación.

------------------------------------------------------------------------

# 11. Conflictos entre fuentes

Si dos fuentes discrepan:

``` text
SOURCE A → X
SOURCE B → Y
```

el grounding no debe simplemente producir:

``` text
X
```

Debe poder producir algo equivalente a:

``` text
[CONFLICTO]

A afirma X.
B afirma Y.

Precedencia declarada:
A > B

Estado:
pendiente / resuelto según política
```

La precedencia puede ayudar a interpretar el conflicto, pero no debe
borrar la trazabilidad de que el conflicto existía.

------------------------------------------------------------------------

# 12. Modos de entrada / Bootstrap

CDAD tenía tres situaciones conceptualmente útiles:

``` text
Prompt-only
Documento existente / Bring Your Own
Metodología profesional / múltiples fuentes
```

GTT ya posee un bootstrap más maduro, por lo que no se debe copiar el
modelo literalmente.

Debe conservarse la intención.

## 12.1 Prompt-only

No existen fuentes de diseño suficientes.

La conversación no debe convertirse silenciosamente en contexto
gobernado.

Debe existir una etapa explícita de elicitación:

``` text
DIALOGUE
   ↓
SOURCE-BRIEF
   ↓
GROUNDING
   ↓
GOVERNED CONTEXT
```

La materialización de `SOURCE-BRIEF` convierte el conocimiento inicial
en una fuente explícita y auditable.

------------------------------------------------------------------------

## 12.2 Existing design / Bring Your Own

Si existe un documento de diseño:

1.  detectarlo;
2.  identificar candidatos;
3.  nunca asumir silenciosamente cuál es el autorizado;
4.  pedir confirmación;
5.  preservar la fuente original;
6.  mapearla al contexto gobernado;
7.  solicitar confirmación del contexto;
8.  freeze solo después de la confirmación humana.

GTT ya conserva buena parte de este flujo y debe reforzarlo con el
modelo de procedencia.

------------------------------------------------------------------------

## 12.3 Professional / Multi-source

Cuando existen múltiples documentos:

``` text
source discovery
→ manifest
→ authority
→ precedence
→ grounding
→ dossier
→ reasoning
```

Este flujo debe ser explícito.

------------------------------------------------------------------------

# 13. SOURCE-BRIEF

Debe preservarse el concepto de fuente original.

`SOURCE-BRIEF.*` no es una segunda Source of Truth del sistema.

Su función es histórica y probatoria:

``` text
original human-authored input
          ↓
SOURCE-BRIEF
          ↓
governed context
```

Después del freeze:

``` text
governed context = source of truth
SOURCE-BRIEF = evidence/history
```

Si ambos contradicen, el contexto gobernado gana.

No debe existir un grounding post-freeze que consulte casualmente
`SOURCE-BRIEF` como si fuera otra autoridad.

------------------------------------------------------------------------

# 14. Working Preferences / Working Agreements

Esta capacidad debe recuperarse, pero **no mezclarse con Session
Memory**.

## 14.1 Separación conceptual

GTT debe distinguir:

``` text
Governed Context
    ↓
verdad del sistema

Session Memory
    ↓
continuidad operacional

Working Preferences
    ↓
cómo prefiere trabajar la persona/equipo

Source Evidence
    ↓
de dónde proviene la información

Proposals
    ↓
cambios todavía no ratificados
```

## 14.2 Alcances

Se debe considerar al menos:

``` text
user-level preferences
team-level working agreements
```

Las preferencias personales pueden ser locales y no versionadas cuando
corresponda.

Los acuerdos de equipo pueden ser versionados y revisados.

## 14.3 Precedencia

Una preferencia de trabajo no puede contradecir el contexto gobernado.

Regla:

> Si una working preference contradice una decisión gobernada, gana el
> contexto gobernado.

No debe tratarse como conflicto arquitectónico.

------------------------------------------------------------------------

# 15. Dos confirmaciones humanas del bootstrap

Esta capacidad de CDAD está correctamente preservada en GTT y debe
permanecer explícita.

Las confirmaciones son distintas.

### Confirmación 1 --- Fuente / diseño

La persona confirma que el documento o material seleccionado representa
el diseño que debe usarse.

### Confirmación 2 --- Contexto generado

La persona confirma que el contexto generado representa correctamente el
diseño.

No deben fusionarse en una única confirmación ambigua.

Flujo:

``` text
source candidate
      ↓
HUMAN CONFIRMATION #1
      ↓
mapping / grounding
      ↓
generated governed context
      ↓
HUMAN CONFIRMATION #2
      ↓
freeze
```

------------------------------------------------------------------------

# 16. Backlog governance

La precedencia debe mantenerse clara:

``` text
L0 / Governed Context
        ↓
ADR
        ↓
Backlog
        ↓
Implementation
```

El backlog no debe convertirse en una segunda arquitectura.

Debe representar la línea de desarrollo, no reemplazar decisiones
gobernadas.

GTT ya conserva esta idea y debe mantenerla.

------------------------------------------------------------------------

# 17. Change Request

El Change Request debe continuar siendo el mecanismo para cambios que
exceden la autorización inmediata del trabajo.

Debe conectar:

``` text
request
→ analysis
→ proposal
→ decision
→ promotion
→ validation
```

No debe ser reemplazado por Session Memory, backlog ni propuestas
informales.

------------------------------------------------------------------------

# 18. Reconciliación con Artifact Identity

La recuperación de capacidades de CDAD debe integrarse con Artifact
Identity.

Ejemplo:

``` text
CDAD concept:
SOURCE-BRIEF.md

GTT identity:
artifact-id = stable identity
path = current physical location
```

No debe crearse un segundo mecanismo de identidad solo para las
funciones recuperadas.

La reconciliación debe permitir detectar:

-   rename;
-   move;
-   delete;
-   retired artifact;
-   artifact nuevo;
-   links obsoletos.

------------------------------------------------------------------------

# 19. Technical Index como capacidad derivada

El índice debe incorporar, cuando sea útil, información de:

-   procedencia;
-   OPEN/BLOCKING;
-   fuentes;
-   ADRs;
-   artifacts;
-   proposals;
-   estado.

Pero:

> El Technical Index nunca debe convertirse en la autoridad de esas
> cosas.

La fuente primaria sigue siendo el artefacto gobernado correspondiente.

------------------------------------------------------------------------

# 20. Query / Retrieval

`gtt-query.sh` debe evolucionar para permitir consultas que respondan
preguntas de gobierno además de navegación técnica.

Ejemplos conceptuales:

``` text
¿Qué artefactos definen la arquitectura?

¿Qué decisiones están OPEN?

¿Qué decisiones son BLOCKING?

¿Qué fuentes respaldan esta afirmación?

¿Qué ADR resolvió este OPEN?

¿Qué propuestas están pendientes?

¿Qué artefactos tienen identidad pero no registro consistente?

¿Qué fuentes tienen precedencia sobre otra?
```

El retrieval debe apuntar a los artefactos reales y no inventar
respuestas derivadas del índice.

------------------------------------------------------------------------

# 21. Status

`gtt-status.sh` debe conservarse como snapshot determinista.

Idealmente debe incorporar una sección resumida:

``` text
Evidence / Governance
- sources
- unresolved conflicts
- OPEN
- BLOCKING
- pending proposals
- pending change requests
```

Debe distinguir claramente:

``` text
DERIVED STATE
vs
AGENT NOTES
```

Las notas generadas por agentes nunca deben confundirse con estado
derivado.

------------------------------------------------------------------------

# 22. Validation Gates

La recuperación debe expresarse como gates deterministas cuando sea
posible.

Nuevos gates candidatos:

``` text
gtt-check-provenance.sh
gtt-check-gaps.sh
gtt-check-sources.sh
gtt-check-working-preferences.sh
```

No necesariamente deben convertirse todos en scripts independientes.

Puede integrarse la lógica en los gates existentes si eso produce una
arquitectura más simple.

### Deben poder detectar como mínimo

-   afirmaciones gobernadas sin procedencia cuando la política la exige;
-   `[VACÍO]` sin clasificación;
-   `[CONFLICTO]` sin fuentes participantes;
-   `[PROPUESTA]` tratada como decisión;
-   BLOCKING pendiente antes de freeze;
-   OPEN sin alcance;
-   OPEN utilizado como autorización;
-   fuentes sin autoridad cuando la política lo exige;
-   precedencia ambigua en un conflicto;
-   working preference que intenta sobreescribir gobernanza.

------------------------------------------------------------------------

# 23. Freeze

El freeze debe seguir siendo una operación humana/gobernada.

Antes del freeze deben verificarse, según el régimen:

``` text
context valid
sources valid
provenance valid
no unresolved BLOCKING
OPENs classified and scoped
proposals not treated as decisions
identity consistent
technical index consistent/derivable
protection active
```

Los `OPEN` válidos no deben bloquear artificialmente el freeze.

------------------------------------------------------------------------

# 24. Qué NO debe hacer la recuperación

No introducir:

-   una segunda carpeta `cdad/`;
-   un segundo index;
-   una segunda identidad de artefactos;
-   un segundo sistema de protección;
-   un segundo sistema de Session Memory;
-   una segunda Source of Truth;
-   agentes adicionales solo porque CDAD tenía determinados nombres;
-   duplicación de documentación con reglas contradictorias.

La recuperación debe ser **semántica y arquitectónica**, no una copia de
archivos.

------------------------------------------------------------------------

# 25. Arquitectura objetivo

La convergencia debe producir aproximadamente esta estructura
conceptual:

``` text
                         ┌─────────────────────┐
                         │   AUTHORIZED SOURCES │
                         └──────────┬──────────┘
                                    │
                              Manifest /
                           Authority / Precedence
                                    │
                                    ▼
                         ┌─────────────────────┐
                         │      GROUNDING      │
                         └──────────┬──────────┘
                                    │
                              Evidence Dossier
                              + Provenance
                                    │
                                    ▼
                         ┌─────────────────────┐
                         │     REASONING       │
                         │   / ADE / Agents    │
                         └──────────┬──────────┘
                                    │
                               Proposal
                                    │
                                    ▼
                         ┌─────────────────────┐
                         │   HUMAN DECISION    │
                         └──────────┬──────────┘
                                    │
                              Promotion
                                    │
                                    ▼
                    ┌──────────────────────────────┐
                    │       GOVERNED DOMAIN        │
                    │ context / ADR / backlog /    │
                    │ proposals / change-request  │
                    └──────────────┬───────────────┘
                                   │
                                  FREEZE
                                   │
                                   ▼
                         ┌─────────────────────┐
                         │      GTTGuard       │
                         └─────────────────────┘


     ┌─────────────────────────────────────────────────────────┐
     │                         GTT ENGINE                       │
     │                                                         │
     │ Artifact Identity │ Technical Index │ Query │ Status    │
     │ Reconcile         │ Validate        │ Guard │ Session   │
     │ ADE adapters      │ Scaffold        │ Scripts/docs     │
     └─────────────────────────────────────────────────────────┘


     Session Memory  ≠  Working Preferences  ≠  Governed Context
```

------------------------------------------------------------------------

# 26. Plan de implementación

## Fase 0 --- Baseline

Antes de modificar:

-   registrar estado actual de GTT;
-   preservar el ZIP como evidencia histórica;
-   ejecutar los checks disponibles;
-   documentar qué funciona actualmente;
-   confirmar el estado real desde el repositorio Git de desarrollo.

**No usar las inconsistencias observadas en el ZIP de trabajo como
evidencia de regresión funcional.**

### Entregable

`GTT-CONVERGENCE-BASELINE.md`

------------------------------------------------------------------------

## Fase 1 --- Provenance

Implementar:

-   taxonomía;
-   formato;
-   reglas;
-   documentación;
-   validación;
-   integración con grounding/reasoning;
-   integración con freeze.

### Criterio de aceptación

Debe ser imposible confundir una propuesta del agente con evidencia
gobernada cuando la política de procedencia esté activa.

------------------------------------------------------------------------

## Fase 2 --- OPEN / BLOCKING

Implementar:

-   modelo;
-   alcance;
-   visibilidad;
-   freeze behavior;
-   resolución;
-   trazabilidad.

### Criterio de aceptación

Un OPEN válido atraviesa freeze.

Un BLOCKING pendiente no atraviesa freeze.

Un OPEN nunca equivale a permiso para modificar el diseño.

------------------------------------------------------------------------

## Fase 3 --- Sources / Manifest / Precedence

Implementar:

-   source registry/manifest;
-   authority;
-   precedence;
-   version;
-   conflicto explícito;
-   integración con grounding.

### Criterio de aceptación

Dos fuentes contradictorias no pueden resolverse accidentalmente por
orden de lectura.

------------------------------------------------------------------------

## Fase 4 --- Bootstrap modes

Formalizar:

-   prompt-only;
-   existing document;
-   multi-source/professional;
-   source confirmation;
-   context confirmation;
-   SOURCE-BRIEF.

### Criterio de aceptación

Todos los caminos de bootstrap terminan en un mismo modelo gobernado y
no crean pipelines paralelos innecesarios.

------------------------------------------------------------------------

## Fase 5 --- Working Preferences

Implementar la separación entre:

``` text
user preferences
team working agreements
session memory
governed context
```

### Criterio de aceptación

Una preferencia nunca puede sobreescribir una decisión gobernada.

------------------------------------------------------------------------

## Fase 6 --- Integración con GTT Engine

Integrar las capacidades recuperadas con:

-   Artifact Identity;
-   Technical Index;
-   Query;
-   Status;
-   Reconcile;
-   Validate;
-   GTTGuard;
-   Session Memory;
-   ADE adapters.

### Criterio de aceptación

No existen mecanismos duplicados.

------------------------------------------------------------------------

## Fase 7 --- End-to-End

Crear un proyecto de prueba Git real y ejecutar:

``` text
init
→ source discovery
→ confirmation #1
→ grounding
→ context proposal
→ confirmation #2
→ freeze
→ query
→ work
→ OPEN resolution
→ change request
→ proposal
→ human promotion
→ artifact move
→ reconcile
→ index
→ status
→ validation
→ freeze
```

Debe probarse al menos:

1.  fuente única;
2.  múltiples fuentes;
3.  conflicto;
4.  vacío BLOCKING;
5.  vacío OPEN;
6.  propuesta no ratificada;
7.  rename de artefacto;
8.  cambio arquitectónico;
9.  working preference;
10. recuperación de sesión;
11. protección de artefactos;
12. integración con un ADE.

------------------------------------------------------------------------

# 27. Matriz de regresión funcional

Antes de declarar final GTT debe existir una matriz:

  Capacidad CDAD                Estado GTT            Acción
  ----------------------------- --------------------- --------------------
  Context is Source of Truth    Conservado            Mantener
  Human promotion boundary      Conservado/mejorado   Mantener
  Proposals                     Conservado/mejorado   Mantener
  ADR governance                Conservado            Mantener
  Change Request                Conservado            Mantener
  Two bootstrap confirmations   Conservado            Formalizar
  SOURCE-BRIEF                  Conservado            Reforzar semántica
  Provenance taxonomy           Diludido/perdido      Recuperar
  OPEN                          Diludido/perdido      Recuperar
  BLOCKING                      Diludido/perdido      Recuperar
  Source authority              Diludido              Recuperar
  Source precedence             Diludido              Recuperar
  Multi-source manifest         Diludido/perdido      Recuperar
  Prompt-only elicitation       Diludido              Recuperar
  Working Preferences           Diludido/perdido      Recuperar
  Artifact Identity             Nuevo GTT             Preservar
  Technical Index               Nuevo GTT             Preservar
  GTTGuard                      Nuevo/mejorado GTT    Preservar
  Session Memory                Nuevo GTT             Preservar
  Query                         Nuevo/mejorado GTT    Preservar
  Status                        Nuevo/mejorado GTT    Preservar
  Reconcile                     Nuevo GTT             Preservar
  ADE session adapters          Nuevo GTT             Preservar
  Deterministic validation      Mejorado GTT          Preservar

Esta tabla debe convertirse en una matriz verificable durante la
implementación y no quedarse como documentación.

------------------------------------------------------------------------

# 28. Definición de "final"

GTT no debe declararse final porque:

-   tenga más archivos;
-   tenga más scripts;
-   pase algunos checks;
-   la documentación diga que está completo.

Debe considerarse listo cuando:

### Gobierno

-   [ ] Context es Source of Truth.
-   [ ] Human promotion boundary funciona.
-   [ ] Proposals no son decisiones.
-   [ ] ADR governance funciona.
-   [ ] Change Request funciona.
-   [ ] Freeze es gobernado.

### Evidencia

-   [ ] Existe taxonomía de procedencia.
-   [ ] `[FUENTE]` es trazable.
-   [ ] `[VACÍO]` es explícito.
-   [ ] `[CONFLICTO]` conserva las fuentes participantes.
-   [ ] `[PROPUESTA]` no puede convertirse silenciosamente en decisión.

### Gaps

-   [ ] BLOCKING bloquea freeze.
-   [ ] OPEN no bloquea freeze cuando está correctamente clasificado.
-   [ ] OPEN tiene alcance.
-   [ ] Resolver OPEN genera trazabilidad.
-   [ ] OPEN no constituye autorización implícita.

### Sources

-   [ ] Fuentes autorizadas pueden declararse.
-   [ ] Authority está definida.
-   [ ] Precedence está definida cuando corresponde.
-   [ ] Las contradicciones son detectables.
-   [ ] SOURCE-BRIEF conserva la fuente original cuando corresponde.

### Bootstrap

-   [ ] Prompt-only.
-   [ ] Existing source/document.
-   [ ] Multi-source/professional.
-   [ ] Confirmación de fuente.
-   [ ] Confirmación de contexto.
-   [ ] Freeze posterior a confirmación.

### Working model

-   [ ] Working Preferences separadas de Session Memory.
-   [ ] User preferences separadas de team agreements.
-   [ ] Ninguna preference puede sobreescribir gobernanza.

### Engine

-   [ ] Artifact Identity.
-   [ ] Technical Index.
-   [ ] Query.
-   [ ] Status.
-   [ ] Reconcile.
-   [ ] Validate.
-   [ ] GTTGuard.
-   [ ] Session adapters.
-   [ ] ADE integration.

### Integridad

-   [ ] No hay mecanismos duplicados.
-   [ ] No hay una segunda Source of Truth.
-   [ ] El índice sigue siendo derivado.
-   [ ] La identidad sigue siendo estable.
-   [ ] La protección es determinista donde corresponda.
-   [ ] Los cambios pueden verificarse end-to-end.

------------------------------------------------------------------------

# 29. Regla para Claude durante esta implementación

Claude debe tratar este documento como **plan de trabajo**, no como
autorización para modificar todo de una vez.

Para cada fase:

1.  inspeccionar el estado real;
2.  identificar qué ya existe;
3.  identificar qué falta;
4.  reutilizar mecanismos existentes;
5.  evitar duplicación;
6.  implementar el mínimo cambio coherente;
7.  actualizar documentación afectada;
8.  ejecutar las validaciones;
9.  registrar resultados;
10. detenerse ante contradicciones arquitectónicas;
11. preparar propuesta para cambios gobernados;
12. no convertir automáticamente una propuesta en decisión.

### Regla adicional

Si GTT ya posee una solución técnicamente superior a la de CDAD, **no
debe implementarse la solución antigua**.

Debe recuperarse solamente la capacidad semántica que se perdió.

Ejemplo:

``` text
CDAD: protección declarativa / hooks
GTT: GTTGuard determinista

→ conservar GTTGuard
→ recuperar solamente las reglas o semánticas que falten
```

Otro ejemplo:

``` text
CDAD: mecanismos basados en paths
GTT: Artifact Identity

→ conservar Artifact Identity
→ no recrear identidad basada en path
```

------------------------------------------------------------------------

# 30. Principio final

La evolución buscada no es:

``` text
CDAD → volver atrás → GTT
```

Es:

``` text
             ┌────────────── CDAD ──────────────┐
             │ governance                       │
             │ provenance                       │
             │ evidence boundary                │
             │ OPEN / BLOCKING                  │
             │ source authority                 │
             │ bootstrap semantics              │
             │ working preferences              │
             └──────────────┬───────────────────┘
                            │ recuperar
                            ▼
                      ┌─────────────┐
                      │     GTT     │
                      │             │
                      │ + Engine    │
                      │ + Identity  │
                      │ + Index     │
                      │ + Guard     │
                      │ + Session   │
                      │ + ADE       │
                      │ + Query     │
                      │ + Reconcile │
                      │ + Validate  │
                      └──────┬──────┘
                             │
                             ▼
                   GTT CONVERGENTE / ESTABLE
```

**La meta no es que GTT se parezca nuevamente a CDAD.**

La meta es que **ninguna capacidad esencial de CDAD haya sido perdida
por la evolución**, mientras que las mejoras arquitectónicas y
operativas introducidas por GTT permanezcan intactas.

------------------------------------------------------------------------

## 31. Primera tarea para Claude

Antes de implementar cualquier cambio, Claude debe producir una
auditoría de este plan contra el repositorio actual:

``` text
A. EXISTE Y ESTÁ CORRECTAMENTE IMPLEMENTADO
B. EXISTE PERO ESTÁ INCOMPLETO
C. EXISTE PERO ESTÁ DUPLICADO / REDUNDANTE
D. EXISTÍA EN CDAD Y ESTÁ PERDIDO
E. EXISTÍA EN CDAD Y ESTÁ DILUIDO
F. GTT LO REEMPLAZÓ POR ALGO SUPERIOR
G. REQUIERE DECISIÓN ARQUITECTÓNICA
```

La auditoría debe citar:

-   archivos;
-   scripts;
-   funciones;
-   reglas;
-   documentación;
-   tests/checks existentes.

**No comenzar implementando.**

Primero producir la matriz de gap real.

Después de esa matriz, implementar por fases y validar cada fase.
