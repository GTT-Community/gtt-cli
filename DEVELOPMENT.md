# gtt-cli-dev — development repository

Private. This is where gtt-cli is developed, governed with GTT. The public repository
[GTT-Community/gtt-cli](https://github.com/GTT-Community/gtt-cli) only receives the clean product: never edit it by
hand. This file is not published (see `.publishignore`).

## Two repositories

| Repository | Holds | How it changes |
|---|---|---|
| `gtt-cli-dev` (private, this one) | Everything: code, tests, GTT governance (`.gtt/`, `gtt-domain/`, `AGENTS.md`, ADE overlay), source documents, reports, scripts | `git commit` and `git push`, as usual |
| `gtt-cli` (public) | The product: code, tests, README, LICENSE, installers, packaging, CI | Only through `scripts/release.sh` |

## Daily work

```bash
make build        # ./gtt
make test-unit    # unit + architecture tests, seconds
make test         # everything, including integration against ../gtt-bootstrap (about 10 minutes)
bash .gtt/scripts/gtt-maintain.sh   # GTT index, protection registry and validation
git commit ... && git push
```

Go lives in `~/.local/go/bin` on this machine. Start every change from a Story in `gtt-domain/backlog.md`.

## Publishing

Everything must be committed first. From the root of this repository:

```bash
bash scripts/release.sh 1.0.2   # new version: code + tag v1.0.2 -> binaries, PyPI and npm
bash scripts/release.sh         # code only, no new version
```

It pushes this repository, mirrors `gtt export --clean` (minus `.publishignore`) into `../gtt-cli-public`, asks once,
then commits, tags and pushes the public repository. The tag starts `.github/workflows/release.yml` there:

1. GoReleaser builds `gtt` for Linux, macOS and Windows (amd64, arm64) and publishes the archives and
   `checksums.txt` to GitHub Releases.
2. `packaging/pypi/build_wheels.py` turns the same binaries into PyPI wheels, published with `uv publish`
   (repository secret `PYPI_TOKEN`; skipped with a warning when it is missing).
3. The `npm/` wrapper is published as `gtt-cli` (repository secret `NPM_TOKEN`; skipped with a warning when it is
   missing).

The version is set by the tag; nothing else needs editing.

## How users install it

| Method | Command |
|---|---|
| curl (Linux, macOS) | `curl -fsSL https://raw.githubusercontent.com/GTT-Community/gtt-cli/main/install.sh \| sh` |
| PowerShell (Windows) | `irm https://raw.githubusercontent.com/GTT-Community/gtt-cli/main/install.ps1 \| iex` |
| uv | `uv tool install gtt-cli` (one-off: `uvx --from gtt-cli gtt version`) |
| npm | `npm install -g gtt-cli` |
| Go | `go install github.com/GTT-Community/gtt-cli/cmd/gtt@latest` |

All of them install the same binary from the latest release, verified against `checksums.txt`.

## Secrets of the public repository

Set in GitHub: `GTT-Community/gtt-cli` → Settings → Secrets and variables → Actions.

| Secret | Used for | Where to get it |
|---|---|---|
| `PYPI_TOKEN` | `uv tool install gtt-cli` | pypi.org → Account settings → API tokens |
| `NPM_TOKEN` | `npm install -g gtt-cli` | npmjs.com → Access Tokens → Automation |

## CI

The same workflow runs in both repositories: unit and architecture tests on Linux, macOS and Windows; integration
against `GTT-Community/gtt-bootstrap` on Linux and Windows; the GTT governance gate only where GTT is installed (here).
Test failures are published as annotations on the run page, readable without log access.
