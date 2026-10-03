# GTT Bootstrap — Guía de instalación

> **Canonical reference:** https://github.com/GTT-Community/gtt-method/blob/main/GTT-CANONICAL-v2.1.md

Este documento contiene los procedimientos detallados para instalar GTT Bootstrap.

El README del repositorio sigue siendo la puerta de entrada canónica. Esta guía amplía la instalación sin sacar el Quick Start del README.

## Modos de instalación

GTT admite dos modos de instalación inicial:

1. **Instalación manual** — una persona integra los archivos de GTT en un proyecto existente.
2. **Instalación asistida por agente** — un ADE/agente de programación con IA lee la documentación de GTT y realiza el bootstrap bajo reglas explícitas.

En ambos casos, el objetivo final es el mismo: establecer el contrato del workspace GTT, poblar el contexto gobernado, obtener confirmación humana y congelar el contexto antes del desarrollo gobernado normal.

---

## Requisitos previos

- Un proyecto/repositorio.
- Git.
- Bash para el CI gate y los scripts.
- Python 3 para el hook de protección.
- Claude Code, Kiro, Codex, GitHub Copilot u otro ADE capaz de seguir el procedimiento de bootstrap.
- Se recomienda un documento de diseño/origen terminado, aunque no es obligatorio.

El documento de diseño puede estar en Markdown, texto, Word, PDF u otro formato habitual.

---

## Instalación manual

### 1. Inspeccionar el proyecto anfitrión

Antes de copiar GTT, inspecciona la raíz del proyecto.

Identifica:

- `AGENTS.md` existente
- `readme-gtt.md` / `readme-gtt.es.md` existente
- `.gtt/` existente (el Motor)
- `gtt-domain/` existente (el dominio gobernado: `context/`, `adr/`, `proposals/`, `backlog.md`, `change-request.md`, `session.md`, `.frozen`)
- `.claude/` existente
- `.kiro/` existente
- `.copilot/copilot-instructions.md` existente
- `.gitignore` existente
- documentos de diseño/origen
- archivos o directorios cuyos nombres sean requeridos por GTT

**No sobrescribas archivos existentes silenciosamente.**

Si ya existe un nombre requerido por GTT, detente y resuelve el conflicto explícitamente.

### 2. Instalar el scaffolding de GTT

El workspace final debe contener:

```text
/
├── AGENTS.md                     # contrato portable del agente (archivo de descubrimiento del ADE)
├── readme-gtt.md
├── readme-gtt.es.md
├── SOURCE-BRIEF.*                # si existió documento fuente
│
├── .gtt/                         # MOTOR DE GTT-METHOD (maquinaria, estado derivado, documentación propia de GTT)
│   ├── README.md
│   ├── index/
│   ├── protection/
│   ├── ade.json                  #   estado de ADE del proyecto: participantes, primario, registro de instalación (proyectos instalados)
│   ├── scaffold/manifest.yaml    #   también el registro de ADE (overlays:) y las templates: que posee el Bootstrap
│   ├── scaffold/templates/       #   plantillas propiedad del Bootstrap (el Cuestionario Inicial de Diseño)
│   ├── scripts/
│   ├── session-adapters/
│   └── docs/                     #   documentación propia de GTT: index, installation, usage, docs, evidence,
│                                 #   gtt-completion, session-adapter-contract
│
├── gtt-domain/                   # EL DOMINIO GOBERNADO POR GTT-METHOD
│   ├── context/                  #   contexto gobernado L0
│   ├── adr/                      #   decisiones aceptadas L1
│   ├── proposals/                #   borradores gobernados pendientes de decisión humana
│   ├── backlog.md
│   ├── change-request.md
│   ├── session.md                #   estado operativo derivado (nunca autoridad)
│   └── .frozen                   #   marcador de freeze, escrito por gtt-freeze.sh
│
└── .claude/ | .kiro/ | .copilot/ # OVERLAYS DE ADE - uno por ADE participante; exactamente un ADE es Primario
```

El Motor vive en `.gtt/` (su documentación, incluyendo este archivo, en
`.gtt/docs/`) y el dominio gobernado en `gtt-domain/` — ver *Higiene del workspace*
en `readme-gtt.md`. Cada artefacto se genera directamente donde lo ubica el
scaffold (`.gtt/scaffold/manifest.yaml`), nunca en otro lugar para moverlo
después. Ambos nombres de directorio son propios de GTT y no colisionan con los `docs/`, `context/` o `adr/` del proyecto:
si el proyecto ya tiene un `.gtt/` o un `gtt-domain/`, detente e informa el conflicto en lugar de combinarlos.


Instala el adaptador de cada ADE que realmente usas en el proyecto — `.claude/` para Claude Code, `.kiro/` para Kiro, `.copilot/copilot-instructions.md` para GitHub Copilot (Codex no necesita ningún archivo adicional) — y declara exactamente uno como ADE Primario (el entorno principal de tu flujo de trabajo; no tiene autoridad). No copies los demás "por las dudas"; el repositorio fuente distribuye todos los adaptadores como catálogo, no como paquete para instalar completo. La vía soportada es `gtt-ade.sh`, que copia los overlays, registra lo que instaló en `.gtt/ade.json` y valida el resultado; ejecútalo primero sin `--apply`:

```bash
bash .gtt/scripts/gtt-ade.sh detect                       # solo candidatos - detectar no es participar
bash .gtt/scripts/gtt-ade.sh install --from <catálogo del bootstrap> --participating claude,copilot --primary claude
bash .gtt/scripts/gtt-ade.sh install --from <catálogo del bootstrap> --participating claude,copilot --primary claude --apply
```

Ver la matriz de adaptadores de ADE en el README.

### 3. Preservar el diseño original

Si el proyecto contiene un documento de diseño y se utilizará como fuente del bootstrap, consérvalo como:

```text
SOURCE-BRIEF.*
```

No modifiques silenciosamente el documento original.

### 4. Combinar `.gitignore`

Si el proyecto ya posee `.gitignore`, combina las entradas necesarias de GTT con él.

No reemplaces el `.gitignore` del proyecto.

### 5. Poblar el contexto

Método recomendado:

```text
bootstrap GTT
```

o:

```text
set up GTT
```

No comiences inventando manualmente los seis archivos de contexto si el procedimiento de bootstrap está disponible.

Si decides crearlos manualmente, comienza por:

```text
gtt-domain/context/stack.md
```

y marca explícitamente las decisiones desconocidas en lugar de adivinarlas.

### 6. Revisar

El responsable humano debe revisar:

- arquitectura
- stack tecnológico
- requisitos
- restricciones
- principios
- visión de solución
- glosario
- mapa de stack

La generación de archivos no significa que el contexto esté ratificado.

### 7. Freeze

Cuando el contexto esté completo y confirmado:

```bash
./.gtt/scripts/gtt-freeze.sh
```

Esto crea:

```text
gtt-domain/.frozen
```

y establece el régimen gobernado.

### 8. Verificar el guardrail

Solicita al agente modificar:

```text
gtt-domain/context/stack.md
```

El mecanismo de enforcement correspondiente debe bloquear la escritura.

Que el modelo diga “no debería hacerlo” no equivale a enforcement determinista.

### 9. Instalar el CI gate

Conecta:

```text
.gtt/scripts/gtt-check-stack.sh
.gtt/scripts/gtt-check-backlog.sh
.gtt/scripts/gtt-check-protection.sh
```

al CI contra la rama por defecto. Esto asegura que los cambios arquitectónicos gobernados y el mapa permanezcan sincronizados, y que `.gtt/protection/registry.yaml` (GTTGuard) siempre coincida con los marcadores `@GTTGuard` realmente presentes en el código.

---

## Instalación asistida por agente

El agente debe tratar este repositorio como un contrato documental ejecutable, no como una colección de archivos que puede copiar sin analizar.

### Procedimiento del agente

1. Leer `readme-gtt.md`.
2. Leer `AGENTS.md`.
3. Detectar los ADE candidatos (`gtt-ade.sh detect`: una observación, nunca una elección) y mostrarlos al humano. El humano elige los ADE participantes y exactamente un Primario; si no se puede establecer con confianza, detenerse y preguntar — nunca adivinar, ni instalar un adaptador que nadie eligió.
4. Inspeccionar el proyecto anfitrión.
5. Identificar el documento de diseño/origen.
6. Si no existe, trabajar mediante conversación.
7. Si existen varios candidatos, preguntar.
8. Nunca adivinar cuál es la fuente autorizada.
9. Nunca sobrescribir silenciosamente un archivo existente con el mismo nombre.
10. Crear el contrato de workspace GTT: el núcleo portable más únicamente los overlays de los ADE participantes, excluyendo explícitamente los demás.
11. Mapear la fuente confirmada al contexto gobernado.
12. Buscar Epics/Stories definidas (documento de requisitos, issue tracker, o conversación previa). Si existen, reconciliarlas en `gtt-domain/backlog.md`; si no existe ninguna, decirlo explícitamente en vez de inventarlas.
13. Solicitar confirmación del contexto generado.
14. Preservar la fuente como `SOURCE-BRIEF.*`.
15. Ejecutar freeze solo después de confirmación humana explícita.
16. Verificar la protección.
17. Informar el estado final.

### Informe obligatorio del agente

Debe informar:

- ADE detectados, ADE participantes y ADE Primario
- adaptadores excluidos explícitamente
- si existe soporte nativo para ese ADE
- archivos creados
- archivos preservados
- conflictos encontrados
- archivos omitidos deliberadamente
- documento fuente utilizado
- si el contexto fue confirmado
- si `gtt-domain/backlog.md` está definido y reconciliado con las Epics/Stories conocidas
- si se ejecutó freeze
- si se verificó la protección
- si se conectó el CI gate
- acciones humanas pendientes

---

## Checklist

- [ ] Proyecto anfitrión inspeccionado.
- [ ] ADE candidatos detectados; ADE participantes y exactamente un Primario elegidos por el humano (preguntado, no adivinado, si no estaba claro).
- [ ] Solo se instalaron los adaptadores de los ADE participantes y quedaron registrados en `.gtt/ade.json`; los demás quedaron explícitamente excluidos.
- [ ] `.gtt/scripts/gtt-check-adapter.sh` pasa para cada ADE participante.
- [ ] Archivos requeridos identificados.
- [ ] Protegidos los archivos existentes contra sobrescritura silenciosa.
- [ ] Scaffolding GTT creado.
- [ ] `SOURCE-BRIEF.*` preservado cuando corresponde.
- [ ] `.gitignore` combinado.
- [ ] Contexto poblado.
- [ ] `gtt-domain/backlog.md` presente; Epics/Stories conocidas reconciliadas o explícitamente ausentes.
- [ ] Revisión humana completada.
- [ ] Contexto confirmado explícitamente.
- [ ] `gtt-domain/.frozen` creado.
- [ ] Escritura protegida verificada.
- [ ] CI gate conectado.
- [ ] Instalación informada.

---

## Siguiente paso

Continúa con [usage.es.md](usage.es.md).

### Destino del despliegue: repositorio Bootstrap vs. proyecto anfitrión

`readme-gtt.md`, `readme-gtt.es.md` y `AGENTS.md` permanecen en la **raíz
del proyecto** — los puntos de entrada canónicos, leídos antes que
cualquier otra cosa. Todo lo demás sigue el mismo scaffold en el repositorio Bootstrap y en cualquier
proyecto anfitrión donde se instale: el Motor (con la Documentación, incluyendo
este archivo) en `.gtt/`, y el dominio gobernado en `gtt-domain/`.


Cuando un agente despliega GTT dentro de un proyecto anfitrión, DEBE reorganizar el workspace instalado para que coincida con el scaffold de GTT:

```text
/
├── AGENTS.md                     # contrato portable del agente (archivo de descubrimiento del ADE)
├── readme-gtt.md
├── readme-gtt.es.md
├── SOURCE-BRIEF.*                # si existió documento fuente
│
├── .gtt/                         # MOTOR DE GTT-METHOD (maquinaria, estado derivado, documentación propia de GTT)
│   ├── README.md
│   ├── index/
│   ├── protection/
│   ├── ade.json                  #   estado de ADE del proyecto: participantes, primario, registro de instalación (proyectos instalados)
│   ├── scaffold/manifest.yaml    #   también el registro de ADE (overlays:) y las templates: que posee el Bootstrap
│   ├── scaffold/templates/       #   plantillas propiedad del Bootstrap (el Cuestionario Inicial de Diseño)
│   ├── scripts/
│   ├── session-adapters/
│   └── docs/                     #   documentación propia de GTT: index, installation, usage, docs, evidence,
│                                 #   gtt-completion, session-adapter-contract
│
├── gtt-domain/                   # EL DOMINIO GOBERNADO POR GTT-METHOD
│   ├── context/                  #   contexto gobernado L0
│   ├── adr/                      #   decisiones aceptadas L1
│   ├── proposals/                #   borradores gobernados pendientes de decisión humana
│   ├── backlog.md
│   ├── change-request.md
│   ├── session.md                #   estado operativo derivado (nunca autoridad)
│   └── .frozen                   #   marcador de freeze, escrito por gtt-freeze.sh
│
└── .claude/ | .kiro/ | .copilot/ # OVERLAYS DE ADE - uno por ADE participante; exactamente un ADE es Primario
```

El adaptador de cada ADE participante — `.claude/` (Claude Code), `.kiro/` (Kiro), o `.copilot/copilot-instructions.md` (GitHub Copilot) — permanece en la raíz del proyecto anfitrión. Solo se instalan los adaptadores de los ADE participantes; la elección queda en `.gtt/ade.json`, escrito solo por `gtt-ade.sh`.

El agente debe preservar los archivos existentes del proyecto, no sobrescribir conflictos silenciosamente y no ejecutar automáticamente el freeze. La revisión y confirmación humana deben ocurrir antes del freeze.
