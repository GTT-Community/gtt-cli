# Reporte de fallo — GTT Bootstrap 1.2.0 en Windows

**Los scripts del Bootstrap ejecutan el `bash.exe` de WSL en lugar de Git Bash**

| | |
|---|---|
| Componente | GTT Bootstrap: `gtt_contract.py`, `gtt_project.py`, `gtt_template.py`, `gtt_session_adapter.py` |
| Versión | 1.2.0, commit `0d187b4` |
| Detectado | 2026-10-03, CI de GTT CLI en `windows-latest` (run 37102186344) |
| Severidad | Alta: en Windows ninguna operación declarada del Bootstrap se puede ejecutar |
| Origen | Bootstrap. El CLI ya ejecuta el contrato con Git Bash |

## Síntoma

`gtt init` en Windows resuelve el Bootstrap, verifica integridad y compatibilidad (esas consultas pasan), pero la
primera operación declarada falla y la instalación se revierte:

```text
Installing GTT Core...
Installing ADE integrations...
GTT initialization failed and was rolled back.
Reason:
The Bootstrap refused ade.install.
Reason:
Windows Subsystem for Linux has no installed distributions.
You can resolve this by installing a distribution with the instructions below:
Use 'wsl.exe --list --online' to list available distributions
and 'wsl.exe --install <Distro>' to install.
```

Con una distribución WSL instalada el resultado sería igual de incorrecto: el script correría dentro de Linux con
una ruta de Windows que no existe ahí.

## Causa

El CLI lanza `bash .gtt/scripts/gtt-contract.sh ...` con Git Bash. Ese script ejecuta `gtt_contract.py` con el
Python de Windows, y Python lanza a su vez la implementación de cada operación llamando a `bash` por nombre:

```python
# .gtt/scripts/gtt_contract.py:300
command = ["bash", impl] + argv
proc = subprocess.run(command, ..., shell=False)
```

En Windows, `subprocess.run(["bash", ...])` usa `CreateProcess`, que busca el ejecutable en el directorio de
sistema (`C:\Windows\System32`) **antes** que en el PATH. Ahí está `bash.exe`, el lanzador de WSL, así que se ejecuta
ese aunque Git Bash esté primero en el PATH.

Mismo patrón en:

| Archivo | Línea | Llamada |
|---|---|---|
| `gtt_contract.py` | 300 | `["bash", impl] + argv` (todas las operaciones) |
| `gtt_project.py` | 75 | `["bash", *cmd]` |
| `gtt_template.py` | 111 | `["bash", index]` |
| `gtt_session_adapter.py` | 262 | `["bash", "-c", command]` |

Las consultas `release`, `negotiate`, `capabilities`, `operations` y `show` no fallan porque se resuelven dentro de
`gtt_contract.py` sin lanzar otro `bash`.

## Corrección propuesta

Un único resolutor de bash, usado en las cuatro llamadas:

```python
import os, shutil, sys

def bash_executable():
    """The bash to run Bootstrap scripts with; never the WSL launcher on Windows."""
    explicit = os.environ.get("GTT_BASH")
    if explicit and os.path.isfile(explicit):
        return explicit
    if sys.platform != "win32":
        return "bash"
    found = shutil.which("bash")          # respects PATH order, unlike CreateProcess
    if found and "\\windows\\system32\\" not in found.lower() and "\\windowsapps\\" not in found.lower():
        return found
    for base in (os.environ.get("ProgramFiles"), os.environ.get("ProgramW6432"), os.environ.get("ProgramFiles(x86)")):
        if base:
            candidate = os.path.join(base, "Git", "bin", "bash.exe")
            if os.path.isfile(candidate):
                return candidate
    raise RuntimeError("bash not found: install Git for Windows, or set GTT_BASH to bash.exe")
```

y en cada llamada: `[bash_executable(), impl] + argv` en lugar de `["bash", impl] + argv`.

**El CLI ya aporta su parte:** desde esta corrección pasa `GTT_BASH=<ruta de Git Bash>` en el entorno de cada
llamada al contrato, así que con el resolutor de arriba el Bootstrap usa exactamente el mismo bash que el CLI.

## Recomendación adicional: `.gitattributes`

El repositorio no tiene `.gitattributes`. Un usuario de Windows que lo clone con `core.autocrlf=true` (lo que trae
por defecto Git for Windows) obtiene los `.sh` con CRLF, que bash no ejecuta. El CLI ya clona con
`core.autocrlf=false`, pero un clon manual no. Propuesta:

```text
*.sh  text eol=lf
*.py  text eol=lf
```

## Pruebas sugeridas

- Windows con Git for Windows y WSL sin distribución: `gtt init` termina bien.
- Windows sin WSL: igual.
- `GTT_BASH` apuntando a un bash concreto: se usa ese.
- Linux y macOS: sin cambios de comportamiento.

## Impacto en el CLI

- GTT CLI 1.0.1 publica binarios de Windows; funcionan hasta donde llega el CLI, pero operar un proyecto en
  Windows requiere esta corrección del Bootstrap.
- En el CI del CLI, la integración en Windows queda informativa (`continue-on-error`) hasta que el Bootstrap
  publique la corrección. Después se quita esa línea.
- Backlog: STORY-047 queda `In Progress` por este motivo.
