# Requerimiento para GTT Bootstrap — estructura de tracking de MD iniciales

| | |
|---|---|
| Para | GTT Bootstrap (próxima versión menor tras 1.2.0) |
| Origen | `GTT_CLI_MD_Initial_Tracking_261001.md`, secciones 2 y 12; GAP-003 del CLI |
| Estado en el CLI | Listo: GTT CLI 1.0.0 ya usa la estructura en cuanto el Bootstrap la declare (2026-10-03) |
| Compatibilidad | Aditivo. Un Bootstrap que no lo declare sigue funcionando igual |

## Qué pide el documento fuente

> Bootstrap solo debe crear/inicializar la estructura de tracking cuando el proyecto se inicializa; establecer los
> valores iniciales necesarios; no duplicar la lógica de procesamiento del CLI.
>
> **Bootstrap prepara la infraestructura. CLI mantiene el estado.**

Hoy el Bootstrap no prepara nada, así que el CLI crea su propio registro en `.gtt/cli/tracking.json`.

## Qué tiene que hacer el Bootstrap

### 1. Declarar la ubicación en el manifiesto del scaffold

En `.gtt/scaffold/manifest.yaml`, una sección nueva de primer nivel:

```yaml
tracking:
  initial_sources: { path: .gtt/tracking/initial-sources.json, schema: 1, maintained_by: cli, role: initial MD tracking - read, applied, validated; derived state, never authority }
```

- `path` es relativa a la raíz del proyecto y debe quedar dentro de `.gtt/` (es estado del motor, no dominio
  gobernado). El CLI ignora una ruta que salga del proyecto.
- El CLI solo lee `path`. El resto de campos es documentación del Bootstrap.
- La ruta concreta la decide el Bootstrap; `.gtt/tracking/initial-sources.json` es una sugerencia.

### 2. Crear el archivo vacío al instalar

El paquete debe traer el archivo con su contenido inicial, para que la instalación del núcleo lo deje en el proyecto:

```json
{
  "schema": 1,
  "notice": "Initial MD tracking, maintained by the gtt CLI. Derived state, never an authority.",
  "entries": {}
}
```

En el manifiesto, la ruta debe figurar también en `engine:` como archivo no requerido, para que `gtt-validate.sh`
no lo marque como inesperado.

### 3. No escribir estado en él

El Bootstrap nunca marca un MD como leído, aplicado o validado. Solo el CLI escribe `entries`.

### 4. Tratarlo como estado del CLI en export y clean

- `export --clean`: excluirlo, igual que el resto de `.gtt/`.
- `clean`: listarlo entre lo que se elimina.
- `update`: preservarlo. Es estado del proyecto y no debe reemplazarse por el archivo vacío del paquete.

### 5. Documentarlo

Una línea en `.gtt/docs/usage.md` y en el README instalado (`.gtt/README.md`) diciendo qué es y quién lo mantiene.

## Formato de `entries` (lo escribe el CLI)

Para referencia, sin obligación de validarlo en el Bootstrap:

```json
"INITIAL-001.md": {
  "path": "INITIAL-001.md",
  "sha256": "…",
  "state": "VALIDATED",
  "discovered_at": "2026-10-03T12:00:00Z",
  "read_at": "…",
  "applied_at": "…",
  "validated_at": "…",
  "changed": false,
  "previous": [{ "sha256": "…", "state": "VALIDATED", "validated_at": "…" }]
}
```

Estados: `DISCOVERED`, `READ`, `APPLIED`, `VALIDATED`, `FAILED`. La identidad es el hash del contenido, no el
nombre del archivo.

## Lo que el CLI ya hace

- Lee `tracking.initial_sources.path` del manifiesto (`internal/bootstrap/manifest.go`).
- Si está declarado, guarda ahí el estado. Si no, usa `.gtt/cli/tracking.json`.
- Migra solo: si el registro declarado está vacío y existe el antiguo, conserva sus entradas y las escribe en
  el nuevo en la siguiente escritura. No crea un segundo registro.
- Prueba de integración: `TestSourceTrackingUsesTheRecordTheBootstrapDeclares`, con un catálogo que declara la
  sección y trae el archivo vacío.

## Pruebas sugeridas en el Bootstrap

- Instalación nueva: el archivo existe con `entries: {}` y `gtt-validate.sh` pasa.
- `update` sobre un proyecto con entradas: las entradas se conservan.
- `export --clean`: el archivo no aparece en la entrega.
- `clean`: el archivo se elimina y figura en el plan de limpieza.

## Cierre del GAP-003

Cuando el Bootstrap publique esto, el GAP-003 del CLI se resuelve sin cambios de código en el CLI.
