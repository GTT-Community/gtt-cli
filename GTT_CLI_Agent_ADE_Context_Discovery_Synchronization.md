# GTT CLI — Agent/ADE Context Discovery and Synchronization

> **GTT Canonical Governance:** https://github.com/GTT-Community/gtt-method/blob/main/GTT-CANONICAL-v2.1.md

## 1. Objetivo

El CLI de GTT debe asumir que el usuario puede trabajar inicialmente con un único ADE/agente de IA —por ejemplo Claude Code— mientras el proyecto puede declarar o incorporar otros agentes, como GitHub Copilot, Codex, Kiro u otros.

Esto crea un riesgo operativo:

- el usuario puede creer que solamente existe un agente;
- el proyecto puede requerir soporte para varios ADE/agentes;
- cada ADE puede mantener directorios, reglas, instrucciones o archivos de contexto propios;
- al cambiar de ADE/agente, parte del contexto requerido puede no estar disponible localmente;
- un agente puede trabajar con un contexto diferente al que el proyecto declara.

La tarea del CLI es **detectar, informar y sincronizar de forma segura el contexto específico de los ADE/agentes que el proyecto utiliza**, sin asumir que el usuario conoce todos los agentes presentes en el proyecto.

La sincronización debe ser automática cuando sea segura y determinista, pero **la incorporación de un nuevo ADE/agente al proyecto debe quedar explícitamente comunicada al usuario**.

---

# 2. Problema

Un proyecto GTT puede evolucionar de:

```text
Usuario
   ↓
Claude Code
```

a:

```text
Usuario
   ↓
Claude Code
GitHub Copilot
Codex
Kiro
...
```

El proyecto puede contener referencias, configuración o artefactos específicos para esos entornos.

Por ejemplo:

```text
.claude/
.github/
.copilot/
.codex/
.kiro/
...
```

No debe suponerse que todos estos directorios tienen el mismo significado ni que pueden copiarse indiscriminadamente entre sí.

El CLI debe distinguir entre:

1. **ADE/agente detectado en la máquina**
2. **ADE/agente declarado por el proyecto**
3. **ADE/agente actualmente utilizado**
4. **contexto específico requerido por el proyecto**
5. **contexto ya sincronizado**
6. **contexto pendiente de recuperación**

---

# 3. Principio fundamental

> **GTT gobierna el contexto del proyecto, no el contexto interno de cada proveedor de IA.**

GTT no debe intentar reemplazar:

- Claude Code;
- GitHub Copilot;
- Codex;
- Kiro;
- Gemini;
- otros ADE/agentes.

Cada herramienta conserva su propio mecanismo de instrucciones, memoria o configuración.

GTT debe proporcionar una capa de coordinación que permita que el proyecto sepa:

```text
¿Qué ADE/agentes reconoce este proyecto?
        ↓
¿Qué contexto específico necesita cada uno?
        ↓
¿Qué está disponible localmente?
        ↓
¿Qué falta?
        ↓
¿Qué puede sincronizar automáticamente GTT?
```

---

# 4. Regla de descubrimiento

Cada ejecución relevante del CLI debe comprobar el estado de los ADE/agentes.

Como mínimo:

```text
gtt init
gtt status
gtt sync
gtt doctor
```

deben poder ejecutar una comprobación de:

```text
Project Agent Registry
        +
Local ADE/Agent Detection
        +
Agent Context Synchronization State
```

La comprobación debe ser determinista siempre que sea posible.

No debe depender de que un agente de IA recuerde hacerla.

---

# 5. Registro de ADE/agentes del proyecto

El proyecto debe disponer de un registro declarativo.

Ejemplo conceptual:

```text
gtt/
└── agents/
    └── registry.yaml
```

Ejemplo:

```yaml
version: 1

agents:
  - id: claude-code
    provider: anthropic
    type: ade
    enabled: true

  - id: github-copilot
    provider: github
    type: agent
    enabled: true

  - id: codex
    provider: openai
    type: agent
    enabled: false

  - id: kiro
    provider: aws
    type: ade
    enabled: false
```

El registro no significa que todas las herramientas deban estar instaladas.

Significa que el proyecto reconoce esos entornos como participantes potenciales o actuales.

---

# 6. No confundir detección con activación

GTT debe diferenciar:

```text
DETECTED
```

de:

```text
ENABLED
```

y de:

```text
ACTIVE
```

Ejemplo:

```text
Claude Code
  detected: true
  enabled: true
  active: true

GitHub Copilot
  detected: true
  enabled: true
  active: false

Codex
  detected: false
  enabled: false
  active: false

Kiro
  detected: true
  enabled: false
  active: false
```

La presencia de una herramienta instalada no implica que el proyecto la utilice.

---

# 7. Detección automática

El CLI debe implementar detectores/adaptadores independientes.

Conceptualmente:

```text
AgentDetector
    │
    ├── ClaudeCodeDetector
    ├── GitHubCopilotDetector
    ├── CodexDetector
    ├── KiroDetector
    └── FutureAgentDetector
```

Cada detector debe informar:

```text
agent_id
provider
installed/detected
version (si está disponible)
known project directories
known user directories
supported synchronization mechanisms
```

Los detectores deben ser extensibles.

No se debe diseñar el CLI para que los nombres actuales de herramientas sean permanentes.

---

# 8. Detección del contexto específico

Para cada agente reconocido:

```text
Project Agent Registry
        ↓
Agent Adapter
        ↓
Context Locations
        ↓
Local State
```

Ejemplo conceptual:

```text
Claude Code
  ├── project instructions
  ├── project rules
  └── project-specific configuration

GitHub Copilot
  ├── repository instructions
  ├── agent instructions
  └── project-specific configuration

Codex
  ├── project instructions
  └── project-specific configuration

Kiro
  ├── steering/configuration
  └── project-specific configuration
```

Los nombres y rutas reales deben pertenecer al adaptador de cada herramienta.

GTT no debe asumir que un directorio de un agente puede ser utilizado directamente por otro.

---

# 9. Regla de sincronización

Cuando el CLI detecte que un ADE/agente declarado por el proyecto requiere contexto que no está disponible localmente:

```text
GTT CLI
   ↓
detecta ausencia
   ↓
determina fuente autorizada
   ↓
informa al usuario
   ↓
descarga/sincroniza
   ↓
valida
   ↓
registra resultado
```

La sincronización debe ser idempotente.

Ejecutar:

```bash
gtt sync
```

dos o diez veces no debe producir duplicación ni corrupción.

---

# 10. Aviso obligatorio al usuario

La incorporación de un ADE/agente nuevo no debe ocurrir silenciosamente.

Ejemplo:

```text
GTT detected a project agent not currently synchronized:

  GitHub Copilot

Required project context:
  .github/...

Source:
  project-declared source

Action:
  GTT can synchronize the required context.

Continue? [Y/n]
```

El usuario debe saber:

- qué agente fue detectado;
- qué contexto se necesita;
- de dónde se obtiene;
- qué directorios/archivos serán creados o actualizados;
- si existen archivos locales que serán preservados.

---

# 11. Automatización segura

La regla es:

> **Detectar automáticamente; modificar de forma segura; informar siempre.**

GTT puede realizar automáticamente una sincronización cuando:

1. el agente ya está declarado por el proyecto;
2. la fuente está autorizada;
3. el destino está definido por el adaptador;
4. no se sobrescriben archivos locales no administrados;
5. la operación es reversible o reconstruible;
6. la operación queda registrada.

Si se detecta un agente que no está declarado por el proyecto:

```text
DETECTED BUT NOT DECLARED
```

GTT debe informar y solicitar confirmación antes de incorporarlo al registro del proyecto.

---

# 12. Fuentes autorizadas

La fuente del contexto debe estar explícitamente definida.

Posibles fuentes:

```text
Git repository
Remote project repository
GTT-managed artifact
Organization repository
Package/versioned artifact
Approved remote source
```

No se debe descargar contenido arbitrario desde una URL simplemente porque un agente lo solicite.

La fuente debe estar gobernada.

---

# 13. Nunca sincronizar secretos

La sincronización de contexto de ADE/agentes no debe copiar automáticamente:

```text
API keys
tokens
credentials
cookies
private authentication state
local secrets
provider credentials
session databases
private chat history
```

Debe existir una separación explícita:

```text
Project Context
      ≠
User Credentials
      ≠
Provider Session State
```

GTT sincroniza contexto de proyecto, no credenciales ni sesiones privadas de proveedores.

---

# 14. No interferir con el contexto propio del ADE

GTT no debe modificar arbitrariamente:

```text
Claude internal state
Copilot internal state
Codex internal state
Kiro internal state
```

Debe trabajar únicamente mediante los puntos de integración documentados por cada adaptador.

La regla es:

```text
GTT
 ↓
Project integration boundary
 ↓
ADE/Agent
```

No:

```text
GTT
 ↓
Internal provider state
```

---

# 15. Registro de sincronización

El CLI debe mantener un estado mínimo de sincronización.

Ejemplo:

```text
.gtt/
└── agent-sync-state.json
```

Contenido conceptual:

```json
{
  "version": 1,
  "agents": {
    "claude-code": {
      "status": "synchronized",
      "last_sync": "2026-09-30T12:00:00Z"
    },
    "github-copilot": {
      "status": "synchronized",
      "last_sync": "2026-09-30T12:00:00Z"
    }
  }
}
```

Este estado debe ser reconstruible.

Nunca debe convertirse en la única fuente de verdad.

---

# 16. Manifest de contexto

Para permitir validación determinista, GTT debería mantener un manifest de los artefactos administrados.

Ejemplo:

```text
.gtt/
└── agent-context-manifest.yaml
```

Conceptualmente:

```yaml
version: 1

artifacts:
  - agent: claude-code
    path: ...
    source: ...
    checksum: ...

  - agent: github-copilot
    path: ...
    source: ...
    checksum: ...
```

Esto permite detectar:

```text
missing
changed
outdated
unexpected
synchronized
```

sin depender de un LLM.

---

# 17. Protección contra sobrescritura

GTT nunca debe sobrescribir silenciosamente un archivo que:

- exista localmente;
- no sea administrado por GTT;
- tenga cambios locales;
- no coincida con el manifest esperado.

Ejemplo:

```text
WARNING

The target file already exists and differs from the
GTT-managed version.

File:
  .github/...

Options:
  [P] Preserve local file
  [M] Show merge information
  [R] Replace after confirmation
  [C] Cancel
```

La operación destructiva requiere confirmación explícita.

---

# 18. Revisión automática en cada ejecución

El CLI debe ejecutar una comprobación liviana en las operaciones normales.

Conceptualmente:

```text
gtt <command>
    ↓
GTT bootstrap/check
    ↓
Detect project agents
    ↓
Compare local context
    ↓
Synchronize safe missing context
    ↓
Continue command
```

No debe convertir cada ejecución del CLI en una operación costosa.

La comprobación debe ser:

```text
fast
deterministic
cacheable
incremental
```

---

# 19. Comando explícito

Debe existir un comando dedicado:

```bash
gtt agents
```

Ejemplo:

```text
GTT Agent Environment

Project agents
────────────────────────────────

Claude Code
  enabled: yes
  detected: yes
  context: synchronized

GitHub Copilot
  enabled: yes
  detected: yes
  context: synchronized

Codex
  enabled: no
  detected: yes
  context: not required

Kiro
  enabled: no
  detected: no
  context: not required
```

También:

```bash
gtt agents check
gtt agents sync
gtt agents list
```

---

# 20. Comportamiento de `gtt agents check`

Debe ser solo lectura.

Ejemplo:

```text
$ gtt agents check

Project configuration: OK

Declared agents:
  Claude Code
  GitHub Copilot

Detected:
  Claude Code
  GitHub Copilot
  Kiro

Synchronization:
  Claude Code      OK
  GitHub Copilot   OK

Undeclared detected agent:
  Kiro

No changes were made.
```

---

# 21. Comportamiento de `gtt agents sync`

Debe sincronizar únicamente lo permitido.

Ejemplo:

```text
$ gtt agents sync

Checking project agents...

Claude Code
  context: OK

GitHub Copilot
  context: missing

GTT can retrieve:
  <managed source>

Target:
  <managed target>

Proceed? [Y/n]
```

Después:

```text
Synchronization completed.

GitHub Copilot
  status: synchronized
  files: 4
  changed: 0
```

---

# 22. Incorporación de un nuevo agente

Si el proyecto pasa de:

```text
Claude Code
```

a:

```text
Claude Code
GitHub Copilot
```

el cambio debe quedar registrado.

Conceptualmente:

```text
Project Agent Registry
        ↓
CHANGE
        ↓
Git diff
        ↓
User-visible notice
```

GTT no debe introducir silenciosamente un nuevo participante del proyecto.

---

# 23. El agente no debe ser el responsable del descubrimiento

Esta es una regla importante.

No debe depender de:

```text
Claude → recordar ejecutar gtt agents
Copilot → recordar ejecutar gtt agents
Codex → recordar ejecutar gtt agents
```

Debe funcionar al revés:

```text
GTT CLI
   ↓
detecta
   ↓
valida
   ↓
sincroniza
   ↓
informa
   ↓
ADE/agente trabaja
```

La responsabilidad pertenece al plano de control de GTT.

---

# 24. Compatibilidad con múltiples ADE/agentes

GTT debe permitir:

```text
Claude Code
+
GitHub Copilot
+
Codex
+
Kiro
```

sin asumir que uno de ellos es el agente principal.

El proyecto puede tener:

```text
Primary Agent: Claude Code
```

pero también:

```text
Supported Agents:
- GitHub Copilot
- Codex
- Kiro
```

La metodología debe permanecer neutral respecto del proveedor.

---

# 25. Cambio de ADE/agente

El usuario puede cambiar:

```text
Claude Code
       ↓
Kiro
```

o:

```text
Kiro
       ↓
Codex
```

o trabajar alternativamente con varios.

Al iniciar el CLI:

```text
GTT checks the project agent environment.
```

El sistema debe comprobar nuevamente:

```text
agent availability
agent registration
agent context
context synchronization
```

No debe asumir que el agente utilizado ayer será el agente utilizado hoy.

---

# 26. Detección del agente actualmente activo

Cuando sea técnicamente posible, GTT puede detectar el contexto de ejecución.

Ejemplo conceptual:

```text
GTT running from:
  Claude Code

Project supports:
  Claude Code
  Copilot
  Codex
```

Pero esta información es informativa.

No debe utilizarse para cambiar automáticamente la arquitectura ni el contexto gobernado.

---

# 27. Relación con el contexto gobernado

El contexto específico del ADE/agente es una capa de integración.

Modelo:

```text
                    GTT
                     │
             Governed Project Context
                     │
          ┌──────────┼──────────┐
          │          │          │
          ▼          ▼          ▼
       Claude      Copilot     Codex
       adapter     adapter    adapter
          │          │          │
          ▼          ▼          ▼
       Agent       Agent       Agent
       Context     Context     Context
```

El contexto de integración no debe convertirse en una segunda autoridad arquitectónica.

La autoridad continúa siendo el contexto gobernado por GTT.

---

# 28. Regla de precedencia

Cuando exista conflicto entre:

```text
GTT Governed Context
```

y:

```text
ADE-specific local instructions
```

el contexto gobernado de GTT mantiene la precedencia definida por el proyecto.

Una instrucción específica de un ADE no puede modificar silenciosamente:

- arquitectura congelada;
- constraints;
- decisiones ratificadas;
- reglas de gobierno;
- artefactos protegidos.

---

# 29. Integración con el modelo de evidencia

La detección y sincronización de ADE/agentes no debe alterar la frontera de evidencia definida por GTT.

El agente sigue recibiendo el contexto que le corresponde según el régimen del proyecto.

La sincronización de directorios del ADE es infraestructura de integración, no una nueva fuente de verdad.

Esto es coherente con la separación existente entre grounding, workers y decisión humana. En el modelo de evidencia, el grounding recupera y cita; los workers razonan sobre el dossier; la decisión permanece fuera del agente. fileciteturn0file0L39-L51

---

# 30. Relación con session handoff

La sincronización de ADE/agentes no debe confundirse con el estado de una sesión.

Son dos problemas distintos:

```text
Agent Context
    ↓
¿Cómo debe trabajar el ADE?

Session Handoff
    ↓
¿Dónde quedó el trabajo?
```

El análisis existente identifica precisamente esta separación: el handoff describe el estado del trabajo, mientras que las preferencias/directivas describen cómo debe comportarse el agente. fileciteturn0file1L55-L65

Por lo tanto:

```text
Agent Synchronization
       ≠
Session Handoff
       ≠
Working Preferences
```

---

# 31. Reutilización con `gtt status`

`gtt status` debería convertirse en el punto de entrada común para conocer:

```text
Project state
Agent state
Synchronization state
Freeze state
Open items
Pending proposals
```

La idea es coherente con el diseño previo donde `cdad status` se plantea como una operación determinista que reconstruye el estado desde artefactos reales, en lugar de depender de la memoria del agente. fileciteturn0file1L33-L41

---

# 32. Diseño extensible mediante adapters

La implementación debería utilizar un contrato similar a:

```text
AgentAdapter
```

con operaciones conceptuales:

```text
detect()
get_project_context_locations()
get_user_context_locations()
get_managed_artifacts()
validate()
sync()
status()
```

No todos los adaptadores necesitan implementar exactamente las mismas operaciones.

El CLI debe poder agregar un nuevo ADE sin modificar el núcleo de GTT.

---

# 33. Política para agentes nuevos

Cuando aparezca un ADE/agente desconocido:

```text
UNKNOWN AGENT
```

GTT debe:

1. detectarlo si puede;
2. informar al usuario;
3. no descargar contenido automáticamente;
4. no modificar el proyecto;
5. permitir agregar un adapter posteriormente.

Ejemplo:

```text
GTT detected an unsupported AI development environment.

Agent:
  <unknown>

GTT will not modify or synchronize this environment.

Create an integration proposal?
```

---

# 34. Trazabilidad

Cada sincronización debe dejar un registro mínimo:

```text
timestamp
agent
adapter version
source
destination
operation
result
```

Ejemplo:

```text
2026-09-30T15:20:00Z
agent: github-copilot
operation: sync
result: success
files: 4
```

No deben almacenarse secretos ni contenido privado innecesario.

---

# 35. CI/CD

La misma comprobación debe poder ejecutarse en CI.

Ejemplo:

```bash
gtt agents check --ci
```

Debe fallar cuando exista una inconsistencia gobernada:

```text
declared agent without required context
modified managed artifact
invalid manifest
unknown synchronization state
```

No debe fallar simplemente porque un agente opcional no está instalado en el runner.

---

# 36. Principio de reconstruibilidad

El estado de sincronización debe ser reconstruible desde:

```text
Project Registry
+
Agent Adapters
+
Managed Context Manifest
+
Filesystem
```

No depender exclusivamente de:

```text
agent memory
chat history
local session database
```

Esto mantiene la portabilidad entre ADE/agentes.

---

# 37. Seguridad

La implementación debe contemplar:

- allowlist de fuentes;
- validación de rutas;
- protección contra path traversal;
- validación de checksums;
- protección contra sobrescritura;
- exclusión de secretos;
- logs sin credenciales;
- confirmación para cambios destructivos;
- posibilidad de rollback cuando corresponda.

---

# 38. Requisitos funcionales

### RF-01

El CLI debe detectar ADE/agentes soportados.

### RF-02

El CLI debe leer el registro de agentes del proyecto.

### RF-03

El CLI debe comparar agentes declarados y detectados.

### RF-04

El CLI debe detectar contexto faltante.

### RF-05

El CLI debe sincronizar contexto administrado de forma segura.

### RF-06

El CLI debe informar al usuario cuando se incorpore un agente nuevo.

### RF-07

El CLI no debe sobrescribir silenciosamente archivos locales.

### RF-08

El CLI no debe sincronizar credenciales ni sesiones privadas.

### RF-09

La operación debe ser idempotente.

### RF-10

El mecanismo debe ser extensible mediante adapters.

### RF-11

El estado debe poder validarse sin utilizar un LLM.

### RF-12

La operación debe funcionar aunque el usuario cambie de ADE/agente.

---

# 39. Criterios de aceptación

La implementación se considera válida cuando:

```text
Caso 1
Claude Code solamente
→ GTT detecta Claude Code
→ contexto OK
```

```text
Caso 2
Proyecto declara Copilot
pero el contexto local falta
→ GTT detecta ausencia
→ informa
→ sincroniza después de confirmación
```

```text
Caso 3
Kiro está instalado
pero no está declarado
→ GTT informa
→ no modifica el proyecto
```

```text
Caso 4
Contexto administrado fue modificado localmente
→ GTT detecta diferencia
→ no sobrescribe silenciosamente
```

```text
Caso 5
Usuario cambia de Claude Code a Codex
→ GTT vuelve a detectar
→ sincroniza el contexto requerido
→ no altera el contexto gobernado
```

```text
Caso 6
Se ejecuta `gtt agents sync` varias veces
→ resultado estable
→ sin duplicaciones
```

---

# 40. Tarea específica para el CLI

## Implementar Agent/ADE Context Discovery & Synchronization

El equipo de implementación debe:

1. definir `AgentAdapter`;
2. definir `AgentRegistry`;
3. implementar detección inicial de ADE/agentes;
4. implementar manifest de contexto;
5. implementar `gtt agents`;
6. implementar `gtt agents check`;
7. implementar `gtt agents sync`;
8. integrar la comprobación con `gtt status`;
9. implementar protección contra sobrescritura;
10. implementar exclusión de secretos;
11. implementar trazabilidad;
12. preparar adapters extensibles para nuevos ADE/agentes;
13. documentar qué rutas administra cada adapter;
14. incorporar pruebas de idempotencia;
15. incorporar pruebas de detección y sincronización;
16. incorporar validación para CI/CD.

---

# 41. Regla final

> **El usuario puede pensar que trabaja con un solo ADE o agente. GTT no debe hacer esa suposición.**

GTT debe conocer el estado real del proyecto mediante un mecanismo determinista:

```text
DISCOVER
   ↓
COMPARE
   ↓
INFORM
   ↓
CONFIRM WHEN REQUIRED
   ↓
SYNC
   ↓
VALIDATE
   ↓
RECORD
```

El objetivo no es controlar qué agente utiliza el usuario.

El objetivo es garantizar que, independientemente de si el usuario trabaja con Claude Code, GitHub Copilot, Codex, Kiro u otro ADE/agente soportado, **el entorno de desarrollo pueda reconstruir y utilizar el contexto de integración requerido por el proyecto sin crear una autoridad paralela ni perder trazabilidad**.

---

## 42. Relación con la arquitectura GTT

Esta capacidad debe considerarse una función del **CLI/plano de control de GTT**, no una responsabilidad del agente.

La separación recomendada es:

```text
                 GTT CLI
                    │
        ┌───────────┴───────────┐
        │                       │
 Agent Discovery          Context Sync
        │                       │
        └───────────┬───────────┘
                    │
             Governed Context
                    │
        ┌───────────┼───────────┐
        ▼           ▼           ▼
      Claude      Copilot      Codex
        │           │           │
        └───────────┼───────────┘
                    ▼
              Development
```

GTT no gobierna al proveedor.

GTT gobierna la **integración del contexto del proyecto con los entornos que participan en su desarrollo**.
