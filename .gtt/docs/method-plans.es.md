# GTT Bootstrap — Planes de Método

> **Canonical reference:** https://github.com/GTT-Community/gtt-method/blob/main/GTT-CANONICAL-v2.1.md

Un **Plan de Método** responde a una sola pregunta: *¿cuánto del trabajo operativo
delegas en GTT?* Eliges una intención — Light, Medium, Hard o Team — y GTT deriva de
ella las políticas técnicas. Nunca se te pide configurar una por una la política de
índice, de identidad o de validación.

Esta página es la versión legible de `.gtt/contract/profiles.json`. Ese archivo es la
única definición; si alguna vez difieren, el contrato tiene razón y la diferencia es
un defecto que hay que reportar.

## Qué es un plan y qué no es

- Un plan es un **perfil de operación y colaboración**. Fija con qué frecuencia GTT
  se detiene a preguntarte y qué debe tener montado un equipo.
- Un plan **no es un nivel de calidad**. Light no es "peor" ni Hard es "mejor":
  encajan en situaciones distintas.
- Un plan **nunca apaga el gobierno**. Las reglas de la sección siguiente valen en
  los cuatro.

## Lo que ningún plan cambia

Valen en todos los planes. Ninguna selección, ADE o nivel de automatización las
debilita, y la verificación del contrato falla si un plan lo intenta.

| Regla | Qué significa para ti |
|---|---|
| Autoridad de decisión humana | Los agentes preparan; solo una persona decide, ratifica y congela. |
| Las decisiones gobernadas siempre se confirman | Un cambio en `gtt-domain/context/`, `gtt-domain/adr/`, la estructura del backlog, un artefacto protegido por GTTGuard o el freeze siempre necesita tu decisión explícita. |
| Las operaciones destructivas siempre se confirman | Nada que no pueda regenerarse se elimina o sobrescribe sin preguntar. |
| Una propuesta nunca es una decisión | Un `[PROPUESTA]` nunca aparece dentro del contexto gobernado. |
| Los conflictos permanecen visibles | Un conflicto entre fuentes se expone, nunca se resuelve en silencio. |
| El freeze es una ratificación | No existe atajo para descongelar; el contexto congelado solo cambia por la vía gobernada. |
| Artefactos protegidos | El código protegido por GTTGuard y el contexto congelado nunca se modifican de forma autónoma. |

## Vocabulario

Los planes se describen con cuatro términos. Cada uno tiene un único significado:

| Término | Significado | Ejemplos |
|---|---|---|
| Operación determinista | El resultado se sigue por completo del repositorio y de la política declarada; no interviene ningún juicio. | Asignar el siguiente id libre, reconstruir el índice, regenerar el registro de GTTGuard, ejecutar la validación |
| Cambio relevante | Una operación automatizada que reescribe archivos que leen personas o ADEs, no archivos derivados. | Actualizar referencias entre documentos tras un movimiento, sincronizar los archivos de contexto de un ADE |
| Operación destructiva | Elimina o sobrescribe algo que no puede regenerarse. | `clean`, quitar un ADE, sobrescribir un archivo |
| Decisión gobernada | Cambia L0/L1, la estructura del backlog, un artefacto protegido o el freeze. | Cambiar el stack, aceptar un ADR, congelar |

## Elegir un plan

El Bootstrap pregunta. No elige por ti, y no infiere el plan a partir del proyecto,
del ADE ni de un proyecto anterior.

```text
Select GTT Method Plan

[1] Light Method
    Maximum safe automation and the fewest interruptions, for one developer or a governed prototype.

[2] Medium Method
    Automation with confirmation of relevant changes, for an individual project or a small team.

[3] Hard Method
    Explicit human authorization of every governed operation, for production and critical systems.

[4] Team Method
    Collaborative governance: the Hard gates plus CI, actor traceability and shared team policy.
```

Las descripciones de una línea son el `summary` de cada plan en el contrato, tal como
está escrito (en inglés).

Hasta que elijas, el proyecto reporta su plan como **no seleccionado**. La validación
sigue funcionando, usando las compuertas de Medium como respaldo — el comportamiento
que GTT tenía antes de que existieran los planes. Ese respaldo no es una selección:
el estado dice `not selected` con todas las letras, y el Bootstrap sigue obligado a
preguntar.

## Los cuatro planes

### Light Method

**Para:** un desarrollador individual, un proyecto personal, un prototipo gobernado,
una POC o un spike — donde se busca la mínima carga operativa.

**GTT hace sin preguntar:** asigna ids, actualiza referencias, reconstruye el índice,
valida y sincroniza el contexto de los ADE cuando no se sobrescribiría nada escrito
por ti.

**Se te pregunta por:** decisiones gobernadas y operaciones destructivas. Nada más.

**Qué relaja Light — dicho con exactitud, porque es el único plan que relaja algo:**

| Control | En Light |
|---|---|
| Profundidad documental | `solution-vision`, `stack` y `constraints` deben ser reales antes del freeze; `architecture`, `principles` y `glossary` pueden quedar mínimos, pero sin placeholders de plantilla. |
| Expectativa de ADR | Solo se espera un ADR para un cambio que altera contexto gobernado ya congelado. |
| Las dos confirmaciones | La Confirmación A (fuente) y la Confirmación B (contexto) pueden pedirse en un mismo intercambio; cada una sigue necesitando su propia respuesta explícita. |
| Cadencia de auditoría | `gtt-audit` no se programa; se ejecuta bajo demanda. |

**Compuertas verificadas por máquina:** procedencia en modo consultivo; las
advertencias no bloquean el freeze.

### Medium Method

**Para:** desarrollo normal — funcionalidades, integraciones, servicios internos — de
una persona o un equipo pequeño que quiere automatización pero prefiere confirmar los
cambios relevantes.

**GTT hace sin preguntar:** asigna ids, reconstruye el índice y valida.

**Se te pregunta por:** cambios relevantes (actualización de referencias,
sincronización del contexto de los ADE), decisiones gobernadas y operaciones
destructivas.

**Exige:** los seis archivos de contexto gobernado reales antes del freeze; un ADR
por cada decisión arquitectónica que cambie contexto gobernado; `gtt-audit` antes de
una release o tras un merge grande. Se recomienda CI.

**Compuertas verificadas por máquina:** procedencia en modo consultivo; las
advertencias no bloquean el freeze.

### Hard Method

**Para:** producción, sistemas críticos o sensibles en seguridad, arquitectura de alto
impacto, trabajo regulado.

Hard no significa trabajo manual. Significa **control explícito sobre las operaciones
gobernadas**: GTT sigue preparándolo todo; espera tu respuesta antes de aplicar.

**GTT hace sin preguntar:** reconstruye el índice y valida — solo estado derivado.

**Se te pregunta por:** ids (GTT propone uno y tú confirmas), cambios relevantes,
decisiones gobernadas y operaciones destructivas.

**Exige:** los seis archivos de contexto reales, cada fila de tecnología citada y un
manifiesto de fuentes; un ADR por cada cambio al contexto gobernado, incluso los
pequeños; rollback y radio de impacto en cada propuesta; `gtt-audit` antes de cada
promoción. Se recomienda CI.

**Compuertas verificadas por máquina:** procedencia obligatoria; manifiesto de
fuentes obligatorio para congelar; cada gap OPEN debe llevar `affects`; las
advertencias bloquean el freeze.

### Team Method

**Para:** equipos y proyectos colaborativos — varias personas, o varios ADE,
cambiando el mismo proyecto gobernado.

**GTT hace sin preguntar:** reconstruye el índice y valida. Para ids, actualización
de referencias y sincronización de ADE sigue lo que el equipo declare en
`gtt-domain/working-agreements.md`; donde el equipo no ha declarado nada, GTT propone
y espera confirmación, igual que en Hard.

**Se te pregunta por:** decisiones gobernadas y operaciones destructivas, siempre; el
resto, según la política del equipo.

**Exige:** todo lo que exige Hard y además: validación ejecutándose en CI en cada
cambio; una segunda persona revisando cada propuesta y cada script de promoción; la
persona que actúa registrada en confirmaciones, ADRs y evidencia; la revisión base
comprobada al promover, de modo que un cambio concurrente se detecta en vez de
sobrescribirse.

**Compuertas verificadas por máquina:** las mismas que Hard.

> **Estado: línea base.** Hoy Team es las compuertas de Hard más los requisitos de
> colaboración anteriores. Están previstos planes específicos para equipos que
> refinarán esta definición. Hasta que se publiquen, no se promete nada más allá de
> lo escrito aquí.

## Comparación

| | Light | Medium | Hard | Team |
|---|---|---|---|---|
| Estructura GTT | sí | sí | sí | sí |
| Invariantes | todas | todas | todas | todas |
| Controles relajados | cuatro, listados arriba | ninguno | ninguno | ninguno |
| Automatización | alta | media-alta | controlada | alta, según política del equipo |
| Resolución de ids | automática | automática | propuesta y luego confirmada | política del equipo |
| Actualización de referencias | automática | confirmada | confirmada | política del equipo |
| Sincronización de contexto ADE | automática cuando es segura | confirmada | confirmada | política del equipo |
| Índice y validación | automáticos | automáticos | automáticos | automáticos |
| Decisiones gobernadas | confirmadas | confirmadas | confirmadas | confirmadas |
| Operaciones destructivas | confirmadas | confirmadas | confirmadas | confirmadas |
| CI | opcional | recomendado | recomendado | requerido |
| Multiusuario | opcional | opcional | posible | central |
| Trazabilidad | estándar | estándar | estándar | reforzada (actor registrado) |
| Compuerta de procedencia | consultiva | consultiva | obligatoria | obligatoria |
| Las advertencias bloquean el freeze | no | no | sí | sí |

## Qué se hace cumplir y quién lo hace

Un plan es honesto sobre dónde se hace cumplir cada una de sus reglas:

| Parte del plan | La hace cumplir | ¿La verifica el Bootstrap? |
|---|---|---|
| Compuertas (procedencia, manifiesto de fuentes, `affects`, advertencias) | `gtt-check-provenance.sh`, dentro de `gtt-validate.sh` y `gtt-freeze.sh` | Sí — de forma determinista |
| Invariantes | Hooks, scripts y compuertas, según declara cada una en el contrato | Sí, donde existe una compuerta o un hook |
| Semántica (rigor de la propuesta, profundidad documental, expectativas de ADR y auditoría) | Instrucciones a los agentes (`AGENTS.md`, el skill de bootstrap) | No — nada verifica que un ADE las siga |
| Política operativa (qué es automático, qué se confirma, CI, trazabilidad de actores) | El CLI de GTT, que la lee del contrato | No — el Bootstrap la declara; el CLI la ejecuta |

El Bootstrap y el CLI no implementan dos veces la misma regla: el Bootstrap define
qué significa un plan; el CLI lee esa definición y actúa en consecuencia.

## Cómo evita GTT interrumpirte

> El desarrollador decide lo que solo el desarrollador puede decidir. GTT hace todo lo demás.

La regla es una sola; **su efecto depende de tu plan**. El plan fija *qué* operaciones
hace GTT sin preguntar; la regla fija *cómo* se comporta GTT alrededor de ellas. Menos fricción nunca
significa menos gobierno: ninguna confirmación, compuerta o invariante anterior se
debilita por ello.

**Antes de preguntarte algo**, GTT comprueba si la respuesta ya se desprende del
contexto gobernado, de tu plan, de las políticas del proyecto, del estado real del
proyecto o de una regla determinista. Si es así, GTT actúa, valida el resultado y
continúa. Si no, nombra la decisión, la explica brevemente y pregunta una sola vez.

**En ningún plan se te pregunta** si reconstruir el índice, validar o continuar con el
siguiente paso planificado. Lo demás depende del plan:

| | Light | Medium | Hard | Team |
|---|---|---|---|---|
| Un id está ocupado | el siguiente libre, usado | el siguiente libre, usado | el siguiente libre, **propuesto** | el siguiente libre, **propuesto** salvo que la política del equipo diga otra cosa |
| Se movió un artefacto | referencias reescritas, reportado | **se detiene**, da el comando | **se detiene**, da el comando | **se detiene**, da el comando salvo que la política del equipo diga otra cosa |
| Hay que sincronizar el contexto ADE | se hace cuando es seguro | **confirmado** | **confirmado** | política del equipo |
| Se te pregunta por | decisiones gobernadas, operaciones destructivas | + cambios relevantes | + ids, cambios relevantes | + lo que la política del equipo reserve |

`bash .gtt/scripts/gtt-project.sh interaction` muestra esto para el plan seleccionado,
derivado de su política — los mismos datos que recibe el CLI.

| Situación | Qué hace GTT |
|---|---|
| El id que pediste está ocupado | Busca el siguiente libre (`gtt-project.sh next-id --kind adr`); un id retirado nunca se reutiliza. Light y Medium lo usan; en Hard y Team se propone y lo confirmas al revisar el paquete — no en una pregunta aparte. |
| Terminó cualquier operación | Ejecuta `gtt-maintain.sh`: registro de protección, índice y validación en una sola pasada, reportados en pocas líneas. |
| Un artefacto movido o renombrado | Reescribir referencias es un cambio relevante. En Light `gtt-maintain.sh` reconcilia los movimientos inequívocos y los reporta; en Medium, Hard y Team — o sin plan seleccionado — se detiene y da el comando exacto de `gtt-reconcile.sh`. Un movimiento ambiguo o un artefacto faltante es una decisión en todos los planes. |
| Un cambio gobernado o protegido | Te entrega **un único script ejecutable** con la operación completa, sus comprobaciones previas y un fallo explícito cuando el estado no es el esperado. Lo ejecutas tú; GTT nunca. Después GTT valida y continúa. |

**GTT solo se detiene por** un conflicto arquitectónico o semántico real; una decisión
que solo tú puedes tomar; una autorización que exige una operación protegida o
gobernada; una condición de seguridad o integridad que impide continuar; o evidencia
faltante que necesita para continuar correctamente. Llegar a un paso mecánico
intermedio no es motivo para detenerse.

**Los reportes son breves por defecto** — qué se hizo, el resultado, si debes actuar y
el siguiente paso:

```text
✓ Protection registry in sync.
✓ Index rebuilt (51 artifact(s), 513 section(s)).
✓ Validation: 11 passed, 1 skipped.
```

El detalle nunca se oculta, solo no se ofrece de entrada: `gtt-maintain.sh --verbose`,
`gtt-status.sh`, `gtt-query.sh` o la salida completa de cualquier verificación.

Dónde se hace cumplir: los dos servicios anteriores son scripts deterministas. El
comportamiento alrededor de ellos — no preguntar, detenerse solo por los motivos
listados, reportar con brevedad — es una instrucción a los agentes y una política que
ejecuta el CLI; el Bootstrap la declara (`profiles.json` → `developer_experience`) y
no puede verificar que un ADE la siga.

## Seleccionar y cambiar de plan

Un plan puede seleccionarse o cambiarse en cualquier momento sin recrear el proyecto.
Solo se escribe `.gtt/methodology.json`; el contexto gobernado, los ADRs, las
propuestas, el backlog, el índice y el historial no se tocan.

```bash
bash .gtt/scripts/gtt-project.sh profile get                          # qué está seleccionado
bash .gtt/scripts/gtt-project.sh profile set --profile light          # simulación: muestra el cambio
bash .gtt/scripts/gtt-project.sh profile set --profile light --apply  # lo escribe
```

Un CLI hace lo mismo mediante las operaciones declaradas `methodology.profile.get` y
`methodology.profile.set`. Las operaciones conservan el nombre *profile* por
compatibilidad con el contrato 1.0; *profile* y *plan* son lo mismo.

**En un proyecto congelado**, pasar a un plan *menos* estricto se rechaza: es un
cambio gobernado y sigue la vía normal (change request → propuesta → decisión
humana). Pasar a uno más estricto está permitido. El orden es
Light < Medium < Hard < Team.

## Lo que el Bootstrap deliberadamente no añade

- **Ningún segundo archivo de configuración.** La selección vive en
  `.gtt/methodology.json` y el significado en `.gtt/contract/profiles.json`. No
  existe `.gtt/config.yaml`.
- **Ningún registro de agentes aparte.** Qué ADEs participan ya queda registrado en
  `.gtt/ade.json` y declarado en `.gtt/scaffold/manifest.yaml`; un ADE nunca se
  incorpora solo por estar instalado en la máquina.
