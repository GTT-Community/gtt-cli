# GTT CLI — Method Plans and Automation Policy

> **GTT Canonical Governance:** https://github.com/GTT-Community/gtt-method/blob/main/GTT-CANONICAL-v2.1.md

## 1. Propósito

Definir cómo el **GTT CLI** implementa los planes metodológicos de GTT y cómo cada plan determina el nivel de automatización, intervención humana y gobierno.

Principio rector:

> **Light Method = menos decisiones y menos intervención humana, no menos gobierno.**

La reducción del plan Light ocurre en la **carga cognitiva y operativa del usuario**, no en la protección del proyecto.

---

## 2. Planes metodológicos

GTT debe utilizar una taxonomía común y explícita:

| Plan | Enfoque |
|---|---|
| **Light Method** | Automatización máxima para un desarrollador/proyecto individual |
| **Medium Method** | Automatización con confirmación humana en cambios relevantes |
| **Hard Method** | Gobierno estricto y confirmación explícita de cambios gobernados |
| **Team Method** | Gobierno colaborativo, trazabilidad y coordinación entre personas/agentes |

Los planes no representan calidad ni madurez.

Representan **cuánta responsabilidad operacional delega el usuario a GTT**.

---

# 3. Principio transversal

Los cuatro planes mantienen:

- gobernanza;
- trazabilidad;
- identidad de artefactos;
- integridad;
- contexto gobernado;
- protección de L0;
- validación;
- procedencia;
- reglas de cambio.

Lo que cambia es:

```text
AUTOMATIZACIÓN
INTERVENCIÓN HUMANA
CONFIRMACIONES
COORDINACIÓN
```

No cambia la autoridad del contexto gobernado.

---

# 4. Light Method

### Objetivo

Reducir al mínimo el trabajo manual del desarrollador individual.

El usuario expresa intención:

```text
"promover este ADR"
"validar"
"sincronizar"
"continuar"
```

GTT resuelve automáticamente la mecánica determinista.

### Comportamiento

GTT debe:

- detectar conflictos de identidad;
- asignar IDs libres;
- actualizar referencias;
- reconstruir índices;
- ejecutar validaciones;
- sincronizar contexto de ADE/agentes;
- detectar inconsistencias;
- corregir inconsistencias deterministas;
- generar scripts cuando una operación deba quedar explícitamente materializada.

### Ejemplo

Si el usuario intenta crear:

```text
ADR-002
```

y `ADR-002` ya existe en el índice:

```text
GTT detecta:

ADR-002 → ocupado
ADR-003 → ocupado
ADR-004 → ocupado
ADR-005 → libre

Resolución:
ADR-005
```

GTT puede continuar automáticamente si la operación:

1. es determinista;
2. no destruye información;
3. no cambia una decisión arquitectónica;
4. no modifica arbitrariamente L0;
5. queda registrada.

El usuario no debería investigar manualmente el índice.

---

# 5. Medium Method

GTT automatiza las operaciones deterministas, pero solicita confirmación antes de cambios relevantes.

Ejemplo:

```text
ADR-002 is already assigned.

Suggested resolution:
ADR-005

Affected references:
6

Proceed? [Y/n]
```

El usuario confirma.

---

# 6. Hard Method

La automatización existe, pero las modificaciones sobre artefactos gobernados requieren confirmación explícita.

Ejemplo:

```text
Identity conflict detected.

Proposed:
ADR-005

Affected L0:
6 files

Action:
REQUIRES HUMAN CONFIRMATION
```

Hard Method no significa que el usuario haga manualmente el trabajo técnico.

Significa que GTT no ejecuta automáticamente una operación gobernada sin autorización explícita.

---

# 7. Team Method

Diseñado para proyectos donde participan:

- varios desarrolladores;
- varios equipos;
- varios ADE/agentes;
- CI/CD;
- repositorios compartidos;
- procesos de revisión.

Team Method añade:

- coordinación;
- trazabilidad de actor;
- locking cuando sea necesario;
- validación CI;
- políticas compartidas;
- historial de operaciones;
- detección de cambios concurrentes;
- revisión de cambios gobernados.

El objetivo no es aumentar el trabajo manual.

El objetivo es evitar que la automatización individual produzca conflictos entre participantes.

---

# 8. Matriz de comportamiento

| Operación | Light | Medium | Hard | Team |
|---|---|---|---|---|
| Detectar ID ocupado | automático | automático | automático | automático |
| Buscar ID libre | automático | automático | automático | automático |
| Actualizar referencias | automático | automático | confirmar | política/revisión |
| Reconstruir índice | automático | automático | confirmar | CI/política |
| Validar | automático | automático | automático | automático |
| Corregir inconsistencia determinista | automático | confirmar | confirmar | política |
| Promover ADR | automático si seguro | confirmar | confirmar | revisión |
| Modificar L0 | según operación segura | confirmar | confirmar | revisión |
| Sincronizar ADE | automático | confirmar si afecta proyecto | confirmar | política |
| Operaciones destructivas | nunca silenciosas | confirmar | confirmar | revisión |
| CI/CD | opcional | recomendado | recomendado | obligatorio |

---

# 9. Regla de seguridad

Ningún plan permite:

```text
silently destructive automation
```

Automatización no significa:

```text
hacer cualquier cosa
```

Significa:

```text
automatizar lo que GTT puede determinar y justificar
```

---

# 10. Resolución automática de identidad

El CLI debe disponer de un servicio conceptual:

```text
ArtifactIdentityAllocator
```

Responsabilidad:

```text
requested ID
      ↓
lookup
      ↓
collision?
      ↓
yes
      ↓
find next valid ID
      ↓
build impact set
      ↓
update references
      ↓
validate
      ↓
record operation
```

Debe funcionar independientemente del agente utilizado.

---

# 11. No depender del agente

Claude Code, Copilot, Codex, Kiro u otro ADE pueden solicitar:

```text
create ADR
promote proposal
validate
```

Pero el agente no debe ser responsable de:

- encontrar IDs libres;
- editar manualmente el índice;
- descubrir todas las referencias;
- decidir cómo resolver una colisión técnica;
- recordar qué archivos deben actualizarse.

Eso pertenece al CLI.

---

# 12. Regla para Light Method

El usuario no debería convertirse en el "orquestador humano" de operaciones deterministas.

Mala experiencia:

```text
Usuario:
  revisa artifacts.json
  busca ID
  renombra
  busca referencias
  edita seis archivos
  ejecuta index
  ejecuta integrity
  vuelve a validar
```

Experiencia esperada:

```text
Usuario:
  gtt promote

GTT:
  collision detected
  ADR-005 allocated
  6 references updated
  index rebuilt
  integrity OK
  validation OK

  Promotion completed.
```

---

# 13. Bootstrap y CLI

El plan seleccionado por Bootstrap debe quedar disponible para el CLI.

Ejemplo:

```text
.gtt/config.yaml
```

```yaml
method:
  plan: light
```

El CLI debe leer esta configuración.

Bootstrap no debe duplicar las reglas del CLI.

Bootstrap **configura**.

CLI **ejecuta**.

---

# 14. Cambio de plan

Debe existir:

```bash
gtt method
```

y:

```bash
gtt method show
gtt method set light
gtt method set medium
gtt method set hard
gtt method set team
```

El cambio de plan debe quedar registrado.

---

# 15. `gtt method show`

Ejemplo:

```text
GTT Method Plan

Plan:
  Light Method

Automation:
  High

Human intervention:
  Low

Governance:
  Full

Project:
  individual

Agent synchronization:
  automatic when safe
```

---

# 16. Validación

El CLI debe poder comprobar:

```bash
gtt method check
```

Debe validar:

- plan válido;
- configuración coherente;
- reglas de automatización;
- políticas compatibles;
- artefactos requeridos;
- integridad.

---

# 17. Regla de precedencia

Si una política del plan entra en conflicto con una constraint gobernada:

```text
Governed Context
        >
Method Plan
        >
ADE preferences
        >
Agent behavior
```

El plan no puede degradar una constraint.

---

# 18. Criterio de diseño

La pregunta para decidir si una operación puede automatizarse en Light debe ser:

> **¿Puede GTT determinar esta operación de forma objetiva, reproducible y segura a partir de los artefactos del proyecto?**

Si sí:

```text
automate
```

Si no:

```text
ask / propose
```

Nunca:

```text
guess
```

---

# 19. Resultado esperado

Con Light Method, un desarrollador debería poder trabajar casi exclusivamente con intención:

```text
create
promote
validate
sync
status
```

mientras GTT realiza la mecánica repetitiva.

El gobierno permanece completo.

La intervención humana se reserva principalmente para decisiones que realmente son decisiones.

---

# 20. Regla final

> **Los planes de GTT no reducen el gobierno; reducen o aumentan la cantidad de trabajo operativo que GTT exige al humano.**

Light Method debe ser el ejemplo más claro:

```text
Less human operation
        ≠
Less governance
```

sino:

```text
More deterministic automation
        +
Same governance
        =
Light Method
```
