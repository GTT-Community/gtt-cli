# GTT Bootstrap

> **Canonical reference:** https://github.com/GTT-Community/gtt-method/blob/main/GTT-CANONICAL-v2.1.md

> **Cuando el contexto no gobierna a la IA, la IA gobierna la solución.**

Kit de inicio oficial de **Governance Throw Think (GTT)** — un flujo práctico para el desarrollo de software asistido por IA.

**El contexto es la Fuente de Verdad.**

Compatible con Claude Code, Kiro, Codex y GitHub Copilot · CC BY 4.0

🌐 **Idiomas**
- 🇺🇸 [English (canónico)](readme-gtt.md)
- 🇪🇸 Español (actual)

---

## Navegación rápida

- [Flujo de uso](#flujo-de-uso)
- [Adaptadores de ADE](#adaptadores-de-ade)
- [Instalación manual](#instalación-manual)
- [Instalación asistida por agente](#instalación-asistida-por-agente)
- [Scaffolding obligatorio del workspace GTT](#scaffolding-obligatorio-del-workspace-gtt)
- [Higiene del workspace](#higiene-del-workspace)
- [El problema](#el-problema)
- [Los dos archivos que siempre tocarás](#los-dos-archivos-que-siempre-tocarás)
- [El mapa](#el-mapa)
- [Cambiar algo](#cambiar-algo)
- [Límite Humano de Promoción](#límite-humano-de-promoción)
- [Backlog](#backlog)
- [Artefactos protegidos (GTTGuard)](#artefactos-protegidos-gttguard)
- [Mantener el mapa honesto](#mantener-el-mapa-honesto)
- [Principio de diseño](#principio-de-diseño)
- [Estructura](#estructura)
- [Capas de contexto](#capas-de-contexto)
- [Primeros pasos](#primeros-pasos)
- [Compatibilidad con herramientas](#compatibilidad-con-herramientas)
- [Lo que mantienes](#lo-que-mantienes)
- [Requisitos](#requisitos)
- [Evolución](#evolución)
- [Licencia](#licencia)

---

## Flujo de uso

### 1. Inicializar el contexto gobernado

**Paso 1 — Comienza con tu diseño, si ya tienes uno.**

Deja tu documento de diseño en la raíz del proyecto. Puede tener cualquier nombre y cualquier formato habitual: `.md`, `.txt`, Word, PDF o equivalente.

No existe una convención de nombre obligatoria. El documento debería estar terminado y no ser un borrador. Cuando corresponda, debe describir:

- idea y objetivo
- visión
- requisitos
- arquitectura propuesta
- stack tecnológico
- restricciones
- reglas de desarrollo

Idealmente, revisa el diseño con un LLM antes de inicializar GTT para detectar inconsistencias.

Si todavía no tienes un documento de diseño, omite este paso. El agente puede definir el contexto contigo mediante conversación.

**Paso 2 — Indica a tu ADE/agente de programación con IA que inicialice GTT.**

Por ejemplo:

> `clone GTT Bootstrap and bootstrap the project`

El agente puede ser Claude Code, Kiro, Codex, Cursor, Copilot u otro ADE capaz de seguir el procedimiento de bootstrap de GTT.

El proceso:

1. Descarga/clona GTT Bootstrap en el proyecto — la distribución fuente es un catálogo de todos los adaptadores, no algo que se instale completo.
2. Detecta los ADE candidatos (una observación, nunca una elección), te los muestra y pregunta cuáles participan en el proyecto y cuál es el Primario. Si no puede establecerlo, pregunta en lugar de adivinar — ver [Adaptadores de ADE](#adaptadores-de-ade).
3. Instala el núcleo portable de GTT más únicamente los overlays de los ADE participantes, excluyendo explícitamente los demás.
4. Comprueba si `gtt-domain/context/` todavía contiene placeholders de plantilla.
5. Busca en la raíz el documento de diseño/origen.
6. Si existe más de un candidato, pregunta en lugar de adivinar. Si no tienes ningún documento de diseño, ofrece el Cuestionario Inicial de Diseño — ver la sección *Empezar sin documento de diseño*.
7. Si existe uno, solicita confirmar que está terminado y no es un borrador.
8. Si no está terminado, se detiene y espera.
9. Lee el documento confirmado y lo mapea a los seis archivos de contexto gobernado.
10. Pregunta directamente aquello que el documento todavía no responde.
11. Resume el contexto resultante y solicita una segunda confirmación explícita: que los seis archivos realmente representan el diseño.
12. Solo después de esa confirmación escribe los archivos de contexto completos.
13. Conserva el documento fuente como `SOURCE-BRIEF.*` en la raíz cuando se haya proporcionado.
14. Indica que debe revisarse el resultado y ejecutarse `.gtt/scripts/gtt-freeze.sh` para ratificarlo.

Antes del freeze todavía no existe nada ratificado que proteger. Por eso el agente puede escribir `gtt-domain/context/` durante este bootstrap inicial.

El freeze es una **acción humana**. Valida que el contexto ya no contenga placeholders y crea el marcador `gtt-domain/.frozen`, que cambia el proyecto al régimen gobernado y protege las rutas gobernadas contra escrituras directas del agente.

Consulta `.claude/skills/gtt-bootstrap/SKILL.md` para el procedimiento detallado.

A partir de ese momento, el agente lee primero el contexto gobernado antes de tomar decisiones de implementación.

La idea es simple:

> Tú y el agente definen qué quieren construir y cómo debe construirse; tú lo confirmas; GTT convierte ese diseño acordado en contexto gobernado; después la IA desarrolla bajo ese contexto.

**Plan de Método.** Durante el bootstrap se te pregunta cuánto trabajo operativo quieres delegar en GTT — una sola elección entre **Light**, **Medium**, **Hard** y **Team**. Es un perfil de operación, no un nivel de calidad; GTT deriva de él las políticas técnicas y nunca elige uno por ti. Ningún plan apaga el gobierno. Qué hace cada plan sin preguntar, qué te pregunta y qué exige está detallado en [.gtt/docs/method-plans.es.md](.gtt/docs/method-plans.es.md).

Para los procedimientos detallados, consulta [.gtt/docs/installation.es.md](.gtt/docs/installation.es.md) y [.gtt/docs/usage.es.md](.gtt/docs/usage.es.md).

### 2. Instalación manual

GTT también puede instalarse manualmente.

El proyecto debe recibir, como mínimo, el scaffolding definido a continuación. Copia los archivos/directorios distribuidos por GTT al proyecto, conserva las ubicaciones requeridas, combina el `.gitignore` con el existente en lugar de sobrescribirlo y completa el contexto gobernado antes de congelarlo.

Consulta [.gtt/docs/installation.es.md](.gtt/docs/installation.es.md#instalación-manual).

### 3. Instalación asistida por agente

Un ADE puede instalar GTT cuando el usuario le proporciona la URL del repositorio o le solicita inicializar GTT.

El agente debe:

1. Leer primero este README.
2. Identificar el contrato de bootstrap y la estructura obligatoria.
3. Inspeccionar el proyecto anfitrión antes de modificarlo.
4. Detectar documentos de diseño sin adivinar.
5. Informar conflictos en lugar de sobrescribir.
6. Crear el scaffolding requerido.
7. Poblar el contexto mediante el flujo de bootstrap.
8. Obtener confirmación explícita antes de ratificarlo.
9. Ejecutar el freeze cuando corresponda.
10. Informar exactamente qué creó, preservó, omitió o requiere acción humana.

Consulta [AGENTS.md](AGENTS.md) para el contrato orientado a agentes.

---

## Adaptadores de ADE

GTT distribuye un núcleo portable más un adaptador por cada ADE soportado.
Un proyecto destino recibe el núcleo portable más el adaptador de cada ADE
que elegiste que participe — nunca el catálogo completo, nunca un adaptador que
nadie eligió. La distribución fuente de GTT Bootstrap contiene todos los
adaptadores porque es un catálogo; instalarlos todos en un proyecto no es el
flujo previsto.

**Un solo modelo de gobernanza, varias superficies de integración, un ADE Primario.**
Un proyecto puede usar un ADE o varios, incluso a la vez (tres terminales en un
mismo editor, por ejemplo). La gobernanza, los artefactos gobernados y la
Frontera de Promoción Humana son los mismos para todos, y cada ADE participante
recibe su overlay — la gobernanza nunca depende de que el Primario sea el activo.
Exactamente un ADE participante es el **ADE Primario**: el entorno principal del
flujo de trabajo del proyecto. Es un identificador de flujo, no una autoridad: no
supera, aprueba ni arbitra a ningún otro ADE, y ningún ADE ratifica nada.

| Término | Significado |
| --- | --- |
| Detectado | Observado en esta máquina o en el repositorio. Solo un candidato — nunca instalado, autorizado, gobernado ni participante por sí mismo. |
| Participante | Elegido por ti para que GTT lo gobierne; recibe su overlay. |
| Primario | Un ADE participante, elegido por ti. |

La elección queda registrada en `.gtt/ade.json` y el registro de ADE es la sección
`overlays:` de `.gtt/scaffold/manifest.yaml`; ambos se leen y escriben solo a través
de `.gtt/scripts/gtt-ade.sh` (`list`, `detect`, `state`, `validate`, `owned`, `install`,
`adopt`, `set-primary`, `record`, `remove`, `update`). Todo comando que escribe es un
ensayo hasta `--apply`, nunca sobrescribe un archivo que GTT no instaló y revierte si
algo falla. Los archivos de instrucciones y los overlays (`AGENTS.md`, `.claude/`,
`.kiro/`, `.copilot/`, `.codex/`, la memoria o el historial de sesiones de un ADE) son
superficies de integración: permiten que un ADE participe en GTT y nunca se convierten
en una autoridad propia.

**Núcleo portable:** `AGENTS.md`, `readme-gtt.md`, `readme-gtt.es.md`, `SOURCE-BRIEF.*` (si existió documento fuente) en la raíz del proyecto, más el dominio gobernado (`gtt-domain/`: `context/`, `adr/`, `proposals/`, `backlog.md`, `change-request.md`, `session.md`) y el Motor de GTT (`.gtt/`, incluida la documentación propia de GTT en `.gtt/docs/`, declarado por `.gtt/scaffold/manifest.yaml`).

Qué contiene el adaptador de cada ADE — un proyecto con varios ADE participantes tiene la unión de sus filas:

| ADE anfitrión | Adaptador | `.claude/` | `.kiro/` | `AGENTS.md` | `.copilot/copilot-instructions.md` |
| --- | --- | :---: | :---: | :---: | :---: |
| Claude Code | Claude | SÍ | NO | SÍ | NO |
| Kiro | Kiro | NO | SÍ | SÍ | NO |
| Codex | Portable/AGENTS | NO | NO | SÍ | NO |
| GitHub Copilot | Copilot | NO | NO | SÍ | SÍ |
| Otro ADE soportado | Solo el adaptador explícito | solo si está mapeado | solo si está mapeado | según soporte | según soporte |
| ADE desconocido | Portable/desconocido | NO | NO | no adivinar | NO |

> **Limitación conocida:** la ruta real que GitHub Copilot lee para
> instrucciones personalizadas a nivel de repositorio es
> `.github/copilot-instructions.md`, según la documentación actual de
> GitHub. GTT mantiene el archivo deliberadamente en
> `.copilot/copilot-instructions.md`, por consistencia de nombres con
> `.claude/` y `.kiro/` — lo que significa que Copilot no lo carga
> automáticamente en esa ruta. Duplicalo en `.github/copilot-instructions.md`
> también si necesitás que Copilot lo cargue por sí solo.

La detección se basa en el entorno realmente presente, nunca en el modelo
subyacente. Un modelo Claude no es Claude Code; un modelo GPT no es Codex; la API
de Anthropic u OpenAI por sí sola tampoco lo es. Y detectar no es participar:
archivos o un binario encontrados para un ADE (`gtt-ade.sh detect`) lo convierten en
un *candidato*. Si el proyecto destino ya muestra archivos de más de un ADE (por
ejemplo, una configuración parcial previa dejó tanto `.claude/` como `.kiro/`), el
agente no adopta uno solo porque sus archivos existan — te muestra los candidatos y
pregunta:

> Se detectaron estos ADE posibles.
>
> - Claude Code
> - Kiro
>
> ¿Cuáles de ellos deben participar en este proyecto (GTT instala y gobierna la
> integración de cada uno) y cuál es el ADE Primario?

Solo Claude Code tiene un bloqueo de escritura en tiempo real; cualquier otro ADE
participante se gobierna por sus instrucciones más el CI gate
(`gtt-check-protection.sh`) — gobernado no es lo mismo que bloqueado, y a ningún ADE
se le atribuye una garantía que no tiene (`enforcement:` en el registro lo declara
por ADE).

Para un ADE sin adaptador nativo, GTT instala solo el núcleo portable e
informa claramente que no existe un adaptador nativo — nunca inventa uno.

Volver a ejecutar el bootstrap nunca elimina un ADE participante, nunca
reintroduce un ADE que una ejecución previa excluyó salvo que tú lo indiques, y
nunca cambia el ADE Primario salvo que tú lo indiques. Un proyecto anterior a
`.gtt/ade.json` sigue funcionando sin cambios; `gtt-ade.sh adopt` registra su ADE
existente como Primario y participante — solo cuando eso coincide con el estado real
(donde hay varios overlays, nunca se adivina).

Algoritmo completo: `.claude/skills/gtt-bootstrap/SKILL.md` (paso 0). Valida un
proyecto instalado con `.gtt/scripts/gtt-check-adapter.sh` (cada ADE participante
contra `.gtt/ade.json`; una integración ausente o inconsistente FALLA, un ADE
detectado que no participa es una ADVERTENCIA). La matriz de un solo ADE anterior
sigue disponible como `.gtt/scripts/gtt-check-adapter.sh <claude|kiro|codex|copilot|unknown>`.

---

## Empezar sin documento de diseño

Si el proyecto no tiene un documento de diseño suficiente, el Bootstrap ofrece su
**Cuestionario Inicial de Diseño** (`.gtt/scaffold/templates/gtt-initial-design-questionnaire.md`,
declarado en `templates:` del manifest). Es un instrumento de elicitación para el ADE
Primario, no lógica del CLI ni arquitectura gobernada:

1. Se detecta la necesidad (tú, el ADE o un CLI) y se solicita la plantilla al
   Bootstrap — `.gtt/scripts/gtt-template.sh materialize initial-design-questionnaire --apply`
   crea la copia de trabajo en `gtt-domain/proposals/bootstrap/initial-design-questionnaire.md`.
   Nadie más lleva una copia del cuestionario ni de su metodología.
2. El ADE Primario conduce una entrevista adaptativa: primero inspecciona el proyecto,
   te pregunta solo lo que falta y va completando el documento. No se espera que
   completes tú solo todo el formulario.
3. Lo desconocido queda como `[VACÍO]` (mejor que una respuesta inventada); los
   desacuerdos como `[CONFLICTO]`; las sugerencias del ADE como `[PROPUESTA]` hasta que
   decidas. Ninguno es una decisión confirmada.
4. Tras tu revisión, con Readiness `READY` o `READY_WITH_OPEN_ITEMS`, el cuestionario
   completo es tu documento fuente (Confirmación A): se preserva textualmente como
   `SOURCE-BRIEF.md` y el contexto gobernado se deriva de él mediante el bootstrap
   normal y la Confirmación B. Completar el cuestionario nunca hace que algo quede
   gobernado por sí mismo.

---

## Scaffolding obligatorio del workspace GTT

Al realizar el bootstrap de GTT en un proyecto, **el agente de programación con IA/ADE DEBE crear y preservar exactamente la siguiente estructura**:

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
│   ├── scaffold/manifest.yaml
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

### Reglas del scaffolding

- `AGENTS.md`, `readme-gtt.md`, `readme-gtt.es.md` y `SOURCE-BRIEF.*` (cuando existe) DEBEN permanecer en la raíz.
- El dominio gobernado — `gtt-domain/context/`, `gtt-domain/adr/`, `gtt-domain/proposals/`, `gtt-domain/backlog.md`, `gtt-domain/change-request.md`, `gtt-domain/session.md`, `gtt-domain/.frozen` — DEBE generarse directamente bajo `gtt-domain/`, nunca en otro lugar para moverlo después. `gtt-domain/.frozen` solo lo escribe `.gtt/scripts/gtt-freeze.sh`, tras confirmación humana; `gtt-domain/session.md` es derivado y lo regenera `.gtt/scripts/gtt-status.sh`.
- El Motor de GTT (`scaffold/`, `scripts/`, `index/`, `protection/`, `session-adapters/`) y la documentación propia de GTT (`docs/`: `index.md`, `installation.md`, `usage.md`, `method-plans.md` y sus pares `.es.md`, `gtt-completion.md`, `evidence.md`, `docs.md`, `session-adapter-contract.md`) DEBEN vivir bajo `.gtt/`, y el README de GTT Bootstrap DEBE instalarse como `.gtt/README.md`. `.gtt/scaffold/manifest.yaml` es la definición canónica y declarativa de este scaffold.

- El agente NO DEBE mover, renombrar, duplicar ni redistribuir artefactos de GTT fuera de esta estructura.
- El agente DEBE preservar la estructura existente del proyecto anfitrión y NO DEBE sobrescribir silenciosamente un archivo existente con el mismo nombre. Los conflictos DEBEN informarse y resolverse explícitamente.
- Los archivos específicos del ADE — `.claude/`, `.kiro/` o `.copilot/copilot-instructions.md` — permanecen en sus ubicaciones requeridas y no modifican el contrato de workspace de GTT. Solo se instalan los adaptadores de los ADE participantes; ver [Adaptadores de ADE](#adaptadores-de-ade).

Esta estructura es un **contrato de bootstrap de GTT**, no solamente una convención documental.

---

## Higiene del workspace

**GTT mantiene separados dos hogares: el Motor (`.gtt/`) y el dominio gobernado
(`gtt-domain/`), más el overlay del ADE.**

La raíz del proyecto pertenece al proyecto. GTT ocupa exactamente dos de sus
directorios, ambos con nombre propio para que no puedan colisionar con los
`docs/`, `context/`, `adr/` o `proposals/` del propio proyecto: `.gtt/` contiene la
maquinaria propia de GTT (scripts, identidad de artefactos e índice técnico,
registro de protección, declaraciones de adaptadores de sesión, el manifest del
scaffold) y la documentación propia de GTT (`.gtt/docs/`); `gtt-domain/` contiene
el dominio gobernado por GTT-Method — `context/`, `adr/`, `proposals/`,
`backlog.md`, `change-request.md`, `session.md`, `.frozen`. Junto a ellos la raíz
solo conserva los puntos de entrada humanos (`readme-gtt.md`, `readme-gtt.es.md`,
`SOURCE-BRIEF.*`), `AGENTS.md` y el overlay del ADE — instrucciones a agentes, en
ninguno de los dos grupos.

El invariante: la autoridad arquitectónica (L0/L1) existe solo en
`gtt-domain/context/` y `gtt-domain/adr/`; la lógica ejecutable de GTT existe solo
en `.gtt/`. `gtt-domain/` no contiene ninguna en operación: `gtt-domain/proposals/`
contiene borradores gobernados, incluidos los scripts de promoción que solo un
humano ejecuta.

La prueba: *¿puedo saber, solo por el nombre del directorio, si un archivo es
maquinaria o documentación sobre GTT (`.gtt/`), o estado gobernado del proyecto
(`gtt-domain/`)?*

| Parte | Dónde | Pertenece a |
| --- | --- | --- |
| Motor | `.gtt/` (incluido `.gtt/docs/`) | GTT |
| Dominio gobernado | `gtt-domain/`: `context/`, `adr/`, `proposals/`, `backlog.md`, `change-request.md`, `session.md`, `.frozen` | el Solution Designer (los agentes solo redactan en `gtt-domain/proposals/`) |
| Overlay del ADE | `.claude/`, `.kiro/`, `.copilot/` | las integraciones de ADE — una por ADE participante, exactamente una Primaria (un identificador de flujo, nunca una autoridad) |


`.gtt/scaffold/manifest.yaml` declara este scaffold. Hay un único scaffold; los ADE
solo le añaden un overlay. El bootstrap genera cada artefacto directamente donde
corresponde — nunca lo escribe en otro lugar pidiéndote que ordenes después, como
versiones anteriores pedían borrar adaptadores de ADE no usados. Ver *Adaptadores de
ADE* arriba para el mismo principio de generación selectiva aplicado a las
integraciones de herramientas. `.gtt/` y `gtt-domain/` tienen nombre propio y no colisionan con los `docs/`, `context/` o `adr/`
del proyecto anfitrión: si el proyecto anfitrión ya tiene alguno de los dos, el bootstrap se
detiene e informa el conflicto en lugar de combinarlos.

---

## El problema

La IA acelera la implementación. Los humanos gobiernan el contexto y la arquitectura.

El modo de fallo no es necesariamente el código defectuoso: los agentes pueden escribir código razonable de manera individual. El problema más profundo es la **deriva arquitectónica**: una secuencia de cambios individualmente defendibles que, en conjunto, desplaza la solución hacia un lugar que nadie decidió alcanzar.

La deriva suele ser invisible a nivel de commit y solo resulta evidente a nivel de arquitectura, precisamente el nivel que menos se revisa de manera continua.

GTT convierte la arquitectura y su contexto en activos explícitos, protegidos y legibles por máquinas. Cambiar decisiones gobernadas pasa a ser un acto deliberado y no un efecto secundario de la implementación.

---

## Los dos archivos que siempre tocarás

| Archivo | Qué es | Cuándo lo tocas |
| --- | --- | --- |
| **`SOURCE-BRIEF.*`** (raíz del proyecto) | Tu diseño original: visión, arquitectura, stack, restricciones, en tus propias palabras | Una vez, antes o durante la configuración |
| **`gtt-domain/change-request.md`** | La puerta de entrada para solicitar un cambio | Cuando deba cambiar una decisión gobernada o la línea de desarrollo |

`SOURCE-BRIEF.*` permanece en la raíz del proyecto — el único artefacto de GTT que es puramente tuyo para encontrar rápido, nunca maquinaria de gobernanza. `gtt-domain/change-request.md` está en `gtt-domain/` junto con el resto del estado gobernado, y es el único punto de entrada que usas constantemente.

`gtt-domain/context/stack.md` es el archivo que leerás con mayor frecuencia —el mapa de una pantalla de lo que es el sistema—, pero es un resultado y no un archivo que normalmente debas editar manualmente. Los cambios aprobados llegan mediante `gtt-domain/change-request.md`.

---

## El mapa

`gtt-domain/context/stack.md` responde **“¿qué es este sistema?”** sin abrir el código.

Proporciona siete vistas:

| # | Vista | Responde |
| ---: | --- | --- |
| 1 | Stack de un vistazo | ¿Sobre qué está construido y qué ADR lo bloqueó? |
| 2 | Mapa de componentes | ¿Qué se comunica con qué y mediante qué protocolo? |
| 3 | Topología de despliegue | ¿Dónde se ejecuta cada pieza? |
| 4 | Observabilidad | Si falla a las 3 a. m., ¿dónde miro? |
| 5 | Reglas de dependencias | ¿Qué módulo puede llamar a cuál? |
| 6 | Historial de cambios del mapa | Una fila por cada ADR aceptado |
| 7 | Señales de drift | ¿Qué rutas fuera de los directorios propios de GTT cargan peso arquitectónico, y qué protegen? |

El mapa usa Markdown más Mermaid, por lo que se renderiza en GitHub y en los IDE. No hay una imagen que regenerar ni una herramienta de diagramación que mantener. Además, se puede revisar como código: un pull request muestra exactamente qué cambió en la arquitectura.

Una fila de la tabla de stack sin un ADR en la columna **Locked by** es, por sí misma, un hallazgo: una decisión entró al sistema sin pasar por gobernanza.

---

## Cambiar algo

Existe una única puerta de entrada. No necesitas buscar qué archivo gobernado modificar.

```text
gtt-domain/change-request.md  ->  gtt-domain/proposals/  ->  revisas y ejecutas un script  ->  gtt-domain/adr/ + gtt-domain/context/stack.md
      declaras intención           el agente propone      el Límite Humano de Promoción     gobernado/protegido
      siempre escribible           escribible por agente
```

Completa el bloque de solicitud en `gtt-domain/change-request.md`: qué debe cambiar, por qué, qué lo desencadenó, alcance, impacto, riesgo y prioridad.

Después solicita al agente que procese la solicitud.

El agente devuelve una propuesta completa con:

- decisión actual
- cambio sugerido
- impacto
- riesgo
- alternativas
- filas exactas del mapa que cambian

Tú apruebas la propuesta. El agente entonces prepara un **paquete de promoción** en `gtt-domain/proposals/`: el borrador del ADR, el texto completo de cada archivo afectado bajo `gtt-domain/context/`, y un script ejecutable:

```bash
bash gtt-domain/proposals/apply-ADR-NNN-<slug>.sh
```

Revisa la propuesta, el ADR y el script, y ejecuta tú mismo ese único comando desde la raíz del proyecto. El script te deja `view` (ver) el texto del ADR y el diff exacto contra el contexto actual, y solo aplica todos los archivos afectados juntos cuando respondés `yes`; nunca lo ejecuta el agente — ver [Límite Humano de Promoción](#límite-humano-de-promoción).

**`gtt-domain/proposals/` es el único directorio gobernado donde el agente puede escribir como parte del flujo de cambios gobernados.**

El trabajo rutinario de implementación no necesita entrar en este flujo. Si las tareas normales requieren solicitudes de cambio repetidamente, probablemente las restricciones están escritas de forma demasiado amplia.

---

## Límite Humano de Promoción

Preparar un cambio gobernado y promoverlo son actos distintos, y GTT los
mantiene así:

```text
PROPUESTA -> PAQUETE DE PROMOCIÓN -> REVISIÓN HUMANA -> EJECUCIÓN HUMANA EXPLÍCITA -> CAMBIO GOBERNADO
```

> **La IA puede preparar el cambio. La IA no puede promover el cambio de forma autónoma.**

El agente puede analizar el impacto, redactar la propuesta, redactar el ADR,
preparar los archivos afectados bajo `gtt-domain/context/`, y generar el script de
promoción. No puede ejecutar ese script, editar `gtt-domain/adr/` o `gtt-domain/context/`
directamente, ni tratar una propuesta, un ADR o un script redactados como una
aprobación — cada promoción gobernada necesita su propia decisión humana
explícita, y aprobar un cambio nunca se traslada al siguiente.

El script de promoción es en sí mismo un artefacto GTT, no un envoltorio de
conveniencia. Vive en `gtt-domain/proposals/`, nombra en su cabecera la propuesta y
el ADR a los que pertenece, pide una confirmación final antes de escribir
nada, aplica todos los archivos que el cambio toca en una sola ejecución, y
falla con claridad en lugar de dejar el mapa a medio actualizar. Regla
completa: `AGENTS.md` → *Human Promotion Boundary*.

---

## Backlog

`gtt-domain/backlog.md` es la línea de desarrollo: Epics, Stories, y el trabajo
que actualmente se espera construir. Responde "qué existe, qué sigue, qué
está bloqueado" — es un artefacto de planificación, no arquitectura, y
nunca una segunda fuente de verdad junto a `gtt-domain/context/`.

```text
Contexto Gobernado / L0  ->  ADR  ->  gtt-domain/backlog.md  ->  Implementación
```

Una Story que contradice el contexto gobernado o un ADR aceptado es un
hallazgo, no una resolución — nunca sobrescribe silenciosamente la
arquitectura.

**Gobernado igual que todo lo demás, con una excepción rutinaria:**

| Cambio | Vía |
| --- | --- |
| Epic o Story nueva/eliminada, o cambio material de alcance/criterios de aceptación | `gtt-domain/change-request.md` → `gtt-domain/proposals/` → decisión del Solution Designer |
| Actualización de estado de Story, *Current Focus*, *Next Work*, *Blocked* durante trabajo ya aprobado | Edición directa — implementación rutinaria, no una decisión gobernada |

Antes de trabajar en desarrollo, el agente establece la Epic/Story
aplicable desde `gtt-domain/backlog.md`. Si hay Epics/Stories definidas en otro lugar
pero ausentes del backlog, esa brecha se reconcilia mediante el proceso de
cambio normal — el agente no las ignora silenciosamente, ni reescribe el
backlog para que coincida sin más. Si no hay ninguna definida, el agente lo
dice explícitamente en vez de inventar requisitos de negocio.

`.gtt/scripts/gtt-check-backlog.sh` verifica determinísticamente la
integridad estructural — IDs únicos de Epic/Story, valores de estado
válidos. Si una Epic/Story es real, vigente y realmente refleja el trabajo
en curso es un juicio que hace la skill `gtt-audit`, no algo que un script
pueda verificar.

---

## Artefactos protegidos (GTTGuard)

`gtt-domain/backlog.md` gobierna *qué* se construye. GTTGuard gobierna *qué
partes del código ya existente un agente nunca puede tocar por su cuenta*
— un archivo, una clase o un método. Es un mecanismo hermano de L0/L1, no
una copia: protege código L3 que vos decidís proteger, y su modelo de
promoción es deliberadamente más liviano que el Límite Humano de Promoción
de arriba.

Marcá una declaración y GTT hace el resto:

```java
@GTTGuard(reason = "Cálculo financiero", source = "ADR-021")
public PaymentResponse calculatePayment(...) { ... }
```

```text
El desarrollador agrega @GTTGuard
        ↓
.gtt/scripts/gtt-guard-sync.sh lo detecta, resuelve el símbolo de forma determinista
        ↓
.gtt/protection/registry.yaml se regenera (derivado — nunca editado a mano)
        ↓
.gtt/scripts/gtt-check-protection.sh lo valida en CI
        ↓
.claude/hooks/protect-guard.py bloquea una edición autónoma en tiempo real
```

`getPayment()` junto a un `calculatePayment()` protegido en el mismo
archivo permanece libremente editable — solo se bloquea el tramo resuelto
del símbolo marcado, y el hook falla de forma segura bloqueando el archivo
completo si esa resolución alguna vez es ambigua. Solicitar un cambio a un
artefacto protegido pasa por `gtt-propose-change` (formulario 5): una vez
que lo aprobás **en la conversación**, el agente lo implementa directamente
y vuelve a sincronizar el registro — sin ADR, sin script, a propósito.

El bloqueo en tiempo real existe hoy en Claude Code. Kiro, Codex y GitHub
Copilot dependen del CI gate (`gtt-check-protection.sh`) más una nota en el
plano de instrucciones, el mismo respaldo honesto que ya se usa para la
condición de dos regímenes de `gtt-domain/context/`/`gtt-domain/adr/`.

Mecanismo completo: `AGENTS.md` → *Protected artifacts (GTTGuard)*.
Procedimientos: `.claude/skills/gtt-guard/SKILL.md` y
`.claude/skills/gtt-propose-change/SKILL.md`.

---

## Mantener el mapa honesto

Cuatro mecanismos, del más débil al más fuerte:

| Mecanismo | Qué hace |
| --- | --- |
| `AGENTS.md` | Establece que un ADR que no declara su efecto sobre el mapa está incompleto |
| Skill `gtt-adr` | Exige un delta del stack antes/después y una fila de historial |
| Skill `gtt-audit` | Verifica las vistas contra manifests, grafo real de imports y reglas de alertas |
| `.gtt/scripts/gtt-check-stack.sh` | **Hace fallar el build** cuando cambia un ADR y el mapa no cambia |

Los tres primeros son instrucciones o procedimientos y dependen parcialmente del comportamiento del modelo. El cuarto es enforcement determinista.

---

## Principio de diseño

Coloca cada preocupación en el plano que puede hacerla cumplir.

| Plano | Mecanismo | Garantía | Coste de contexto |
| --- | --- | --- | --- |
| Control | `permissions.deny` + hook PreToolUse | Determinista | Cero |
| Build | CI gate en `.gtt/scripts/` | Determinista, al hacer merge | Cero |
| Instrucción | `AGENTS.md`, `.claude/rules/` | Probabilística | Tokens |
| Procedimental | `.claude/skills/` | Bajo demanda | Cero hasta invocarse |

**Todo lo que pueda hacerse cumplir en el plano de control no debería expresarse solamente como una instrucción.**

Por ejemplo, escribir “la IA no debe modificar los archivos de arquitectura” en el contexto consume tokens en cada sesión y solo ofrece una garantía probabilística. Bloquear la escritura en el plano de control la hace determinista sin consumir contexto.

Las instrucciones siguen siendo necesarias para el trabajo que requiere juicio: decidir si un cambio es arquitectónico, si la implementación contradice el contexto o si una abstracción está justificada.

El segundo principio se deriva de esto: **la capa determina tanto quién puede editar como cuándo se carga**. Solo las reglas y restricciones críticas deben cargarse al inicio; el conocimiento más amplio queda disponible bajo demanda.

---

## Estructura

```text
AGENTS.md                       # reglas centrales portables
readme-gtt.md                   # este archivo — configuración, compatibilidad
readme-gtt.es.md                # espejo en español
SOURCE-BRIEF.*                  # diseño original, preservado tras el bootstrap
.gitignore                      # combinar con el del proyecto anfitrión
│
.gtt/                           # MOTOR DE GTT — maquinaria, estado derivado, documentación propia de GTT
├── README.md                   # README operativo orientado al proyecto
├── scaffold/manifest.yaml      # la definición canónica y declarativa del scaffold (layout versión 2)
├── index/                      # identidad de artefactos (artifacts.json) + índice técnico derivado
├── protection/                 # registry.yaml de GTTGuard — derivado, nunca editado a mano
├── session-adapters/           # declaraciones de Session Memory por ADE (solo datos)
├── docs/                       # documentación propia de GTT
│   ├── index.md                # mapa de todos los archivos — comienza aquí
│   ├── installation.md         # procedimientos detallados de instalación (+ installation.es.md)
│   ├── usage.md                # el flujo normal de desarrollo (+ usage.es.md)
│   ├── method-plans.md         # Light / Medium / Hard / Team en palabras claras (+ method-plans.es.md)
│   ├── gtt-completion.md       # registro durable de finalización del bootstrap
│   ├── evidence.md
│   ├── docs.md                 # metodología, portabilidad, migración
│   └── session-adapter-contract.md
└── scripts/
    ├── gtt-check-stack.sh       # CI gate
    ├── gtt-check-adapter.sh     # valida que el adaptador instalado coincide con la matriz
    ├── gtt-check-backlog.sh     # valida la integridad estructural de gtt-domain/backlog.md
    ├── gtt-check-protection.sh  # valida el registro de GTTGuard
    ├── gtt-guard-sync.sh        # regenera el registro de GTTGuard desde los marcadores en el código
    └── gtt_guard.py             # motor compartido de GTTGuard (detección, resolución, registro)
│
gtt-domain/                     # EL DOMINIO GOBERNADO POR GTT-METHOD
├── context/                    # L0 — contexto gobernado
│   ├── stack.md                # mapa de arquitectura con siete vistas
│   ├── architecture.md
│   ├── solution-vision.md
│   ├── principles.md
│   ├── constraints.md          # restricciones siempre disponibles
│   └── glossary.md
├── adr/                        # L1 — decisiones aceptadas
├── proposals/                  # borradores gobernados pendientes de decisión humana
├── backlog.md                  # línea de desarrollo — Epics, Stories, foco actual
├── change-request.md           # puerta de entrada para cambios
├── session.md                  # estado operativo derivado — nunca autoridad
└── .frozen                     # marcador de freeze, escrito por gtt-freeze.sh
│

# abajo: el catálogo de adaptadores que distribuye esta fuente — un proyecto
# instalado recibe los que necesitan sus ADE participantes, elegidos al momento del bootstrap
│
.claude/                        # adaptador de Claude Code
├── CLAUDE.md
├── settings.json
├── hooks/protect-l0.py
├── hooks/protect-guard.py      # bloqueo en tiempo real de GTTGuard
├── rules/
└── skills/
    ├── gtt-bootstrap
    ├── gtt-propose-change
    ├── gtt-adr
    ├── gtt-audit
    └── gtt-guard
│
.kiro/steering/                 # adaptador de Kiro
│
.copilot/copilot-instructions.md # adaptador de GitHub Copilot
```

### Por qué algunos archivos permanecen en la raíz

`.claude/`, `.kiro/` y `.copilot/copilot-instructions.md` permanecen en la raíz porque estas herramientas descubren su configuración en ubicaciones determinadas. Moverlos dentro de `.gtt/` puede hacer que dejen de cargar silenciosamente las reglas y skills previstas. Solo se instala el que corresponde a tu adaptador resuelto — ver [Adaptadores de ADE](#adaptadores-de-ade).

`AGENTS.md` permanece en la raíz porque Kiro, Codex y Copilot lo leen por convención.

`readme-gtt.md`/`.es.md` permanecen en la raíz porque son los puntos de entrada humanos — lo primero que cualquiera que abra el proyecto debería poder encontrar, no algo enterrado bajo `.gtt/docs/`.

`SOURCE-BRIEF.*` permanece en la raíz por la misma razón: es el diseño original del Solution Designer, en sus propias palabras, y debe ser tan descubrible como los READMEs.

El dominio gobernado está en `gtt-domain/`, aparte del Motor, porque es el estado gobernado del propio proyecto, no maquinaria de GTT — ver [Higiene del workspace](#higiene-del-workspace). El Motor y la documentación propia de GTT permanecen en `.gtt/`.

---

## Capas de contexto

| Capa | Contenido | Política | Carga |
| --- | --- | --- | --- |
| L0 | `gtt-domain/context/` | Solo propuesta | Bajo demanda, excepto `constraints.md` |
| L1 | `gtt-domain/adr/` | Propuesta con revisión | Bajo demanda |
| L2 | `.gtt/docs/` | Editable con revisión | Nunca automáticamente |
| L3 | `src/`, `tests/`, pipelines, IaC | Editable | Según necesidad |

---

## Primeros pasos

1. Copia el núcleo portable — `AGENTS.md`, `readme-gtt.md`, `readme-gtt.es.md`, `.gtt/` (el Motor, incluyendo `.gtt/scripts/`, `.gtt/docs/` y `.gtt/scaffold/manifest.yaml`) y el esqueleto del dominio gobernado (`gtt-domain/context/`, `gtt-domain/adr/`, `gtt-domain/proposals/`, `gtt-domain/backlog.md`, `gtt-domain/change-request.md`) — en la raíz del proyecto, más únicamente el adaptador correspondiente a tu ADE: `.claude/` (incluyendo `.claude/CLAUDE.md`) para Claude Code, `.kiro/` para Kiro, `.copilot/copilot-instructions.md` para GitHub Copilot, o nada adicional para Codex. Ver [Adaptadores de ADE](#adaptadores-de-ade); no copies los demás adaptadores "por las dudas".
2. Combina el `.gitignore` de GTT con el existente; no sobrescribas el archivo del proyecto.
3. Ejecuta la skill `gtt-bootstrap` (por ejemplo, “bootstrap GTT” o “set up GTT”) en lugar de completar `gtt-domain/context/` manualmente — realiza el paso 1 anterior por vos, de forma determinista.
4. Si prefieres crear el contexto manualmente, comienza con `gtt-domain/context/stack.md`. Deja una celda vacía en vez de adivinar; un dato desconocido explícito es mejor que una decisión inventada.
5. Ajusta los globs `paths:` de `.claude/rules/` al layout del proyecto anfitrión (solo Claude Code).
6. Conecta `.gtt/scripts/gtt-check-stack.sh`, `.gtt/scripts/gtt-check-backlog.sh` y `.gtt/scripts/gtt-check-protection.sh` al CI contra la rama por defecto.
7. Ejecuta una sesión e inspecciona `/context`. Solo deberían cargarse automáticamente las reglas centrales y las restricciones esperadas.
8. Verifica el guardrail: pide al agente editar un archivo protegido, como `gtt-domain/context/stack.md`. La escritura debe ser bloqueada por el mecanismo de enforcement correspondiente y no simplemente desaconsejada.
9. Verifica los adaptadores: `.gtt/scripts/gtt-check-adapter.sh` confirma que cada ADE participante tiene una integración intacta (contra `.gtt/ade.json`).
10. Revisa el contexto terminado y ejecuta `.gtt/scripts/gtt-freeze.sh` para ratificarlo.

### Actualización al modelo de dos regímenes

Si actualizas un proyecto creado antes de que existiera el modelo de dos regímenes, ejecuta:

```bash
./.gtt/scripts/gtt-freeze.sh
```

inmediatamente después de la actualización cuando `gtt-domain/context/` ya contenga contenido real. Hasta que exista el marcador de freeze, ese contexto puede seguir siendo escribible por el agente.

Mapa completo de archivos: [`.gtt/docs/index.md`](.gtt/docs/index.md) · Migración: [`.gtt/docs/docs.md#migrating-from-gtt-v1`](.gtt/docs/docs.md#migrating-from-gtt-v1)

---

## Compatibilidad con herramientas

| Capacidad | Claude Code | Kiro | Codex | GitHub Copilot |
| --- | --- | --- | --- | --- |
| Reglas centrales portables | mediante import | nativo | nativo | nativo (`AGENTS.md`) + puntero `.github/copilot-instructions.md` |
| Carga condicional | `paths:` | `inclusion: fileMatch` | `AGENTS.md` anidados | ninguna — solo a nivel repo |
| Procedimientos bajo demanda | Skills | `inclusion: manual` | prompt | prompt |
| Bloqueo determinista de escritura | sí | `permissions.yaml` (1.0+) | globs de configuración | no — solo CI gate |
| Contexto gobernado + CI gate | sí | sí | sí | sí |
| Bloqueo en tiempo real de GTTGuard | sí — `protect-guard.py` | no — solo CI gate | no — solo CI gate | no — solo CI gate |

Claude Code soporta el conjunto completo de adaptadores. `permissions.yaml` de Kiro cubre declarativamente las rutas de infraestructura incondicionales; las rutas dependientes del régimen utilizan el hook compartido y el CI gate cuando corresponde. Codex mantiene el modelo de protección de escritura, pero dispone de menos controles de carga condicional. GitHub Copilot lee instrucciones a nivel de repositorio desde `.github/copilot-instructions.md` e instrucciones de agente desde `AGENTS.md`, según la documentación actual de GitHub — el archivo adaptador de GTT vive en `.copilot/copilot-instructions.md` en cambio, así que no se carga automáticamente en la ruta real de Copilot; ver la nota arriba. No tiene carga condicional por rutas ni bloqueo determinista de escritura más allá del CI gate — el adaptador de Copilot es deliberadamente delgado y no reclama capacidades que GTT no haya implementado realmente para él.

Detalles y notas de portabilidad: [`.gtt/docs/docs.md`](.gtt/docs/docs.md#portability-claude-code-kiro-codex-copilot)

### Cambiar de ADE más adelante

El bootstrap instala solo los adaptadores de los ADE que elegiste — ver
[Adaptadores de ADE](#adaptadores-de-ade). No hay nada que podar el primer día.
Sumar, quitar o cambiar la prioridad de un ADE más adelante se hace con
`gtt-ade.sh`, nunca borrando directorios a mano: sabe exactamente qué archivos
instaló GTT (`.gtt/ade.json` guarda un registro con hashes) y no toca tu propia
configuración de ADE. Todo comando es un ensayo hasta `--apply`:

```bash
# Sumar Copilot junto a los ADE que ya usas
bash .gtt/scripts/gtt-ade.sh install --from <catálogo del bootstrap> --participating claude,copilot --primary claude
# Hacer Primario a otro ADE participante (un identificador de flujo; no se mueve ninguna autoridad)
bash .gtt/scripts/gtt-ade.sh set-primary copilot --apply
# Dejar de usar Kiro: elimina solo los archivos que GTT instaló para él y conserva los tuyos
bash .gtt/scripts/gtt-ade.sh remove kiro --apply
# Actualizar todos los overlays participantes tras actualizar el Bootstrap (nunca pisa ediciones locales)
bash .gtt/scripts/gtt-ade.sh update --from <nuevo catálogo del bootstrap> --apply
```

Para un proyecto instalado antes del soporte multi-ADE, ejecuta una vez
`gtt-ade.sh adopt` para registrar el ADE existente; nada más cambia.

**Nunca elimines `AGENTS.md`.** Contiene las reglas centrales portables. Claude Code las importa; Kiro, Codex y Copilot las leen de forma nativa.

Eliminar `.claude/` elimina su capa local de enforcement. En Kiro, `permissions.yaml` proporciona protección incondicional donde es compatible; las rutas dependientes del régimen pueden depender del hook compartido y del CI gate. En Codex o Copilot, utiliza `AGENTS.md` anidados cuando necesites reglas específicas por ámbito:

```text
AGENTS.md
src/AGENTS.md
infra/AGENTS.md
```

---

## Lo que mantienes

- `gtt-domain/context/` y `gtt-domain/adr/`: se aplican mediante el proceso gobernado y, una vez congelados, no deben ser escritos directamente por un agente.
- `gtt-domain/backlog.md`: las Epics/Stories cambian vía `gtt-domain/change-request.md` como una decisión arquitectónica; las actualizaciones de estado y foco durante implementación rutinaria son ediciones directas.
- `gtt-domain/change-request.md`: tu puerta de entrada cuando deba cambiar una decisión gobernada o la línea de desarrollo comprometida.
- `SOURCE-BRIEF.*`: se escribe una vez durante el bootstrap y se conserva como fuente original, en la raíz del proyecto.
- Los adaptadores de tus ADE participantes (`.claude/`, `.kiro/`, `.copilot/copilot-instructions.md`), `.gtt/ade.json` (escrito solo por `gtt-ade.sh`) y `.gtt/scripts/`: activos de runtime/integración de GTT que normalmente requieren pocos cambios, aparte de la configuración de rutas.
- `.gtt/protection/registry.yaml`: nunca se mantiene a mano — se regenera desde los marcadores `@GTTGuard` en el código mediante `.gtt/scripts/gtt-guard-sync.sh`. Tu parte es poner/quitar el marcador; el registro sigue.

---

## Requisitos

Claude Code, Kiro, Codex o GitHub Copilot.

El hook de protección (Claude Code) necesita `python3`, presente por defecto en Linux y macOS. El CI gate necesita `git` y `bash`.

---

## Evolución

GTT es una metodología en evolución centrada en la gobernanza del contexto en el desarrollo asistido por IA. El trabajo futuro puede extenderla a soluciones de software, nube e infraestructura, sistemas agénticos, documentación y gobernanza del conocimiento, preservando el principio central:

> **El contexto es la Fuente de Verdad.**

Relacionado: [GTT Framework](https://github.com/mgriott/context-driven-ai-development) — metodología, whitepapers, principios y modelo de gobernanza.

Comunidad: [GTT Community (ES)](https://gtt-community.github.io/es/) · [gtt-docs](https://github.com/GTT-Community/gtt-docs)

---

## Licencia

Creative Commons Attribution 4.0 International (CC BY 4.0).

Eres libre de compartir, adaptar y construir sobre este trabajo, incluso comercialmente, siempre que se otorgue la atribución correspondiente.

**Atribución:** Copyright © 2026 Moisés Griott. Mantenido por **GTT Community**.

[CC BY 4.0](https://creativecommons.org/licenses/by/4.0/)

---

**GTT Community** · Governance Throw Think
