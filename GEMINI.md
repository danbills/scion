# Scion: Multi-Agent Orchestration Testbed

Scion is an experimental multi-agent orchestration platform designed to run and manage "deep agents" (such as Gemini CLI, Claude Code, and others) in isolated, concurrent environments. Each agent operates within its own container, complete with a dedicated git worktree and isolated credentials, allowing for parallel collaboration on the same project without conflict.

## Project Overview

- **Core Mission:** Orchestrate isolated, harness-agnostic agents that coordinate through natural language and CLI tools.
- **Key Technologies:**
  - **Backend:** Go (Golang)
  - **Frontend:** React (TypeScript)
  - **Orchestration:** Docker, Podman, Kubernetes
  - **Session Management:** Tmux (inside containers)
  - **Communication:** WebSockets, Protobuf, OpenTelemetry
  - **Database:** SQLite (local) / PostgreSQL (Hub) via `ent`
- **Architecture:**
  - `cmd/`: CLI entry points (`scion`, `sciontool`).
  - `pkg/runtime`: Abstractions for container runtimes (Docker, Podman, K8s).
  - `pkg/agent`: Logic for agent lifecycle (provisioning, starting, messaging).
  - `pkg/harness`: Integration for specific agent harnesses (Gemini, Claude, etc.).
  - `pkg/broker`: Messaging and inter-agent communication.
  - `web/`: Web-based dashboard for monitoring and interaction.

## Building and Running

### Prerequisites
- Go 1.25.4 or later
- Node.js and npm (for the web frontend)
- Docker or Podman (for local agent execution)

### Key Commands (via Makefile)

- **Build all:** `make all` (Builds web assets and Go binary)
- **Build CLI:** `make build` (Compiles the `scion` binary to `./build/`)
- **Install CLI:** `make install` (Installs `scion` to `~/.local/bin/`)
- **Run Tests:** `make test` (Runs all Go tests)
- **Fast Tests:** `make test-fast` (Runs tests excluding SQLite-heavy ones)
- **Format Code:** `make fmt` (Runs `gofmt`)
- **Linting:** `make lint` or `make golangci-lint`
- **Build Frontend:** `make web` (Installs dependencies and builds the React app)

### Quick Start
1.  Initialize the machine: `scion init --machine`
2.  Initialize a project (Grove): `scion init`
3.  Start an agent: `scion start debug "Your prompt here" --attach`

## Development Conventions

- **Isolation:** Never modify files directly in the main repository from within an agent; use the provided git worktrees in `.scion/agents/`.
- **Testing:** New features should include tests in the corresponding `pkg/` directory. Use `-tags no_sqlite` for fast iteration if possible.
- **Coding Style:** Adhere to standard Go conventions. Use `make fmt` before committing.
- **Harnesses:** When adding a new agent type, implement the `Harness` interface in `pkg/harness/`.
- **Runtimes:** Use the `Runtime` interface in `pkg/runtime/` for any operations involving containers.
- **Tmux Interaction:** Scion communicates with agents via `tmux send-keys` and bracketed paste. Be mindful of this when modifying agent shell environments.
- **Git:** Add `.scion/agents` to your global or project-level `.gitignore` to avoid tracking nested worktrees.
