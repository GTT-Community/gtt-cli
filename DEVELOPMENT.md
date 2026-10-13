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
2. `packaging/pypi/build_wheels.py` turns the same binaries into PyPI wheels, published with `uv publish`.
3. The `npm/` wrapper is published as `gtt-cli`.

PyPI and npm are separate jobs. A channel that cannot be published fails its job, so a release is green only when
every channel carries it. Retry a failed channel with "Re-run failed jobs" on the run page: both skip a version that
is already published.

The version is set by the tag; nothing else needs editing. (`baseVersion` in `internal/core/identity.go` only
labels plain source builds; bump it now and then.)

## How users install it

| Method | Command |
|---|---|
| curl (Linux, macOS) | `curl -fsSL https://raw.githubusercontent.com/GTT-Community/gtt-cli/main/install.sh \| sh` |
| PowerShell (Windows) | `irm https://raw.githubusercontent.com/GTT-Community/gtt-cli/main/install.ps1 \| iex` |
| uv | `uv tool install gtt-cli` (one-off: `uvx --from gtt-cli gtt version`) |
| npm | `npm install -g gtt-cli` |
| Go | `go install github.com/GTT-Community/gtt-cli/cmd/gtt@latest` |

All of them install the same binary from the latest release, verified against `checksums.txt`.

## Publishing credentials of the public repository

Each channel authenticates as a Trusted Publisher (OIDC: the registry trusts the workflow `release.yml` of
`GTT-Community/gtt-cli`, no secret to keep or renew), or with a repository secret when one is set. The secret wins.

| Channel | Trusted Publisher | Secret instead |
|---|---|---|
| PyPI | pypi.org → Your account → Publishing → add a GitHub publisher for project `gtt-cli` (a pending publisher before the first release) | `PYPI_TOKEN`: pypi.org → Account settings → API tokens |
| npm | npmjs.com → package `gtt-cli` → Settings → Trusted Publisher → GitHub Actions. The package must exist, so the first release needs `NPM_TOKEN` | `NPM_TOKEN`: npmjs.com → Access Tokens → granular token with read and write on packages |

Secrets are set in GitHub: `GTT-Community/gtt-cli` → Settings → Secrets and variables → Actions.

## CI

The same workflow runs in both repositories: unit and architecture tests on Linux, macOS and Windows; integration
against `GTT-Community/gtt-bootstrap` on Linux and Windows; the GTT governance gate only where GTT is installed (here).
Test failures are published as annotations on the run page, readable without log access.
