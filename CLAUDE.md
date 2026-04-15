# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build, test, lint

All development is driven through the `Makefile` (run `make help` for the full list). The CI job on GitHub mirrors `make ci-full`, so passing that locally is the strongest pre-PR signal.

- `make build` — compile `scion` into `./build/scion`.
- `make install` — build + install to `~/.local/bin/scion`. Required after any change that affects container launch args (the broker embeds the CLI's version/flags at build time).
- `make all` — `web` + `install`. Use this when you've touched anything the dashboard loads from `web/dist`; otherwise the embedded web assets 404.
- `make test-fast` — run the whole Go test suite with the `no_sqlite` build tag. This is what CI runs. Plain `make test` pulls in SQLite via cgo and uses more memory.
- `make lint` — `go vet -tags no_sqlite ./...`.
- `make golangci-lint` — runs golangci-lint only on diff vs. `main` (`--new-from-rev=main`). Lint findings on unchanged code are ignored by design.
- `make fmt` / `make fmt-check` — gofmt.
- `make ci` (fast) / `make ci-full` (full, includes web + golangci-lint).
- Single test: `go test -run TestName ./pkg/hub/...` (add `-tags no_sqlite` if the package pulls in ent/sqlite).
- `make container-binaries` cross-compiles `scion` and `sciontool` for `linux/amd64|arm64` into `./.build/container/`, then `export SCION_DEV_BINARIES=./.build/container` to make the broker mount those binaries into agent containers instead of the ones baked into images. Essential when iterating on `cmd/sciontool` or `pkg/runtimebroker`.

## Architecture: the three-binary, three-plane model

Scion is a multi-agent orchestration testbed. Its biggest conceptual move is: **the host-side CLI, the in-container agent supervisor, and the optional central control plane are three separate programs speaking well-defined APIs**. Understanding which plane a piece of code lives on is usually the first step to navigating it.

### Binaries (`cmd/`)

- **`scion`** (`cmd/scion/main.go` → dispatch in `cmd/*.go`) — the user-facing CLI. Starts/stops/lists agents, manages groves, talks to a Hub when configured. Commands are thin; real logic sits in `pkg/`.
- **`sciontool`** (`cmd/sciontool/`) — PID 1 inside every agent container. Clones the git workspace, resolves auth/env, launches the harness CLI (Claude/Gemini/Codex/OpenCode) inside tmux, reaps zombies, and streams telemetry back. When hub-mode clones fail, this is where the error is emitted.

### Core abstractions (`pkg/`)

- **`pkg/api`** — the interface types (`api.Harness`, `api.Runtime`, agent/grove records) that glue everything together. If you're adding a harness or runtime, this is the contract.
- **`pkg/harness`** — per-CLI adapters. Each file (`claude_code.go`, `gemini_cli.go`, `codex.go`, `opencode.go`, `generic.go`) implements `api.Harness`: `ResolveAuth`, `InjectSystemPrompt`, `GetCommand`. `auth.go:RequiredAuthEnvKeys` is the source of truth for which env vars hub-mode env-gather will demand for each harness.
- **`pkg/runtime`** — container runtimes. `factory.go` selects Docker/Podman/Apple Container/K8s based on OS + profile. `podman.go` handles rootless quirks (the `--userns=keep-id` flag is set here).
- **`pkg/runtimebroker`** (listens on a Unix socket by default) — local-mode and hub-mode both dispatch agent-create requests to the broker. `handlers.go` is the entry point; `start_context.go` builds the `StartOptions` that include workspace, env, git-clone config. Env-gather (which keys the hub must supply for the agent to run) lives in `extractRequiredEnvKeys` + `resolveHarnessConfigForEnvGather`.
- **`pkg/hub`** — optional HTTP control plane backed by `ent` (SQLite at `~/.scion/hub.db`). Tracks groves, agents, secrets, brokers. `httpdispatcher.go` translates hub-side agent records into broker RPC calls; `handlers.go` hosts the REST API. `clone_url.go`'s `normalizeCloneURLLabel` decides whether a grove's git remote is cloneable as-is or needs an `https://` prefix.
- **`pkg/ent/schema`** — the persistence model. Changes here require `go generate ./pkg/ent/...`.
- **`pkg/plugin`** — dynamic harness/runtime plugins loaded at startup.

### Data model

- **Grove** — a project directory with a `.scion/` folder. Holds `settings.yaml`, `templates/`, `harness-configs/`, and agent state under `.scion/agents/`. In hub mode, a grove is a DB row keyed on its git remote; `scion hub link` registers it.
- **Template** (`.scion/templates/<name>/`) — agent blueprint. Three files: `scion-agent.yaml` (harness-config, runtime, services), `system-prompt.md` (role identity; opencode downgrades this to `AGENTS.md`), and `agents.md` (behavioral notes).
- **Harness config** (`.scion/harness-configs/<name>/config.yaml`) — container image, model, env, auth method. Resolution order: template path → grove `.scion/harness-configs` → `~/.scion/harness-configs` (see `pkg/config.FindHarnessConfigDir`).
- **Agent** — a container provisioned from (template × harness config). State tracked under `.scion/agents/<name>/` locally; in hub mode, mirrored to the Hub DB.

### Container images (`image-build/`)

- `core-base` → `scion-base` → per-harness images (`claude/`, `gemini/`, `codex/`, `opencode/`). Each harness Dockerfile installs that CLI on top of `scion-base`. Harness configs reference these images by tag (e.g. `scion-opencode:latest`).

## Writing tests

- Table-driven tests using `testing.T` are the norm; parallel subtests (`t.Parallel()`) are common.
- Hub and broker tests use in-memory ent/SQLite — they won't run under the `no_sqlite` build tag, so `make test-fast` (and therefore CI) skips them. Use `go test ./pkg/hub/...` (no tag) locally when working on hub handlers.
- Integration-style tests that need a real container runtime key off `podman`/`docker` being on PATH and skip otherwise.

## Gotchas

- After editing `cmd/` or anything that affects `scion`'s behavior on the host, `make install` before retesting — many Scion invocations shell out to the installed binary (e.g. the broker re-exec's `scion` internally).
- After editing `cmd/sciontool` or runtime-side code, either rebuild the relevant harness image **or** use `make container-binaries` + `SCION_DEV_BINARIES` to hot-swap without rebuilding images.
- Hub mode + env-gather: a harness's `RequiredAuthEnvKeys` can block agent dispatch if no matching secret is set on the hub. For local-only demos, a dummy value (e.g. `scion hub secret set OPENAI_API_KEY local`) is enough to satisfy the group.
