# Settings Refactor Design

## Motivation

Current configuration management suffers from coupled concerns and "tangled" logic.
- **Runtime vs Container Intersection**: A specific container image might be needed for a specific runtime.
- **Feature Flags**: Settings like `use_tmux` might apply to some runtimes but not others, or impact container selection.
- **User Assumptions**: Containers are often built with a specific user (e.g., `node`, `root`) in mind, but this is currently hardcoded or loosely coupled.
- **Ambiguity**: The distinction between "local" (e.g., Docker Desktop) and "remote" (e.g., K8s cluster) runtimes needs clear separation and "aliasing" capabilities.

## Proposed Structure

We propose refactoring the `Settings` struct (and the corresponding JSON representation) to use top-level `local` and `remote` blocks. Each block effectively acts as a named configuration profile or "alias".

### JSON Schema Draft

```json
{
  "defaults": {
    "runtime_alias": "local",  // Points to which top-level block to use by default
    "template": "gemini"
  },
  "local": {
    "runtime": {
      "type": "docker",
      "host": "unix:///var/run/docker.sock",
      "use_tmux": true
    },
    "providers": [
      {
        "name": "gemini",
        "image": "us-docker.pkg.dev/scion/gemini-cli:latest",
        "username": "root"
      },
      {
        "name": "claude",
        "image": "claude-code:latest",
        "username": "node"
      }
    ]
  },
  "remote": {
    "runtime": {
      "type": "kubernetes",
      "context": "gke_my-project_us-central1_my-cluster",
      "namespace": "scion-agents",
      "use_tmux": false
    },
    "providers": [
       {
        "name": "gemini",
        "image": "us-docker.pkg.dev/scion/gemini-cli:prod",
        "username": "root"
      }
    ]
  }
}
```

## Key Concepts

### 1. Top-Level Aliases (`local`, `remote`)
Instead of a flat structure, we group configurations into logical "environments" or aliases.
- `local`: Typically for Docker or local process execution.
- `remote`: Typically for Kubernetes or remote SSH execution.
- *Extensibility*: We could potentially allow arbitrary keys here (e.g., `remote-staging`, `remote-prod`), but starting with fixed `local`/`remote` simplifies the initial refactor.

### 2. Runtime Block
The `runtime` block defines **how** the agent is executed.
- `type`: The underlying runtime engine (`docker`, `kubernetes`, `process`).
- `use_tmux`: Whether to wrap the command in tmux. This is now context-aware (e.g., might want tmux locally but not on K8s).
- *Runtime-specific fields*: `host` for Docker, `context`/`namespace` for K8s.

### 3. Providers Array
The `providers` array defines **what** is executed.
- It maps a provider name (e.g., "gemini", "claude") to specific artifacts.
- `image`: The container image to use. This solves the "intersection" problem where `remote` might use a signed production image while `local` uses a `:latest` or locally built image.
- `username`: Explicitly defines the user the container is built for, removing hardcoded assumptions in the code.

## Impact on Codebase

### `Settings` Struct
The Go struct will need to change from a flat list of runtimes to a map or structured nesting.

```go
type ProviderConfig struct {
    Name     string `json:"name"`
    Image    string `json:"image"`
    Username string `json:"username"`
}

type RuntimeConfig struct {
    Type    string                 `json:"type"` // "docker", "kubernetes"
    UseTmux bool                   `json:"use_tmux"`
    Config  map[string]interface{} `json:"config,omitempty"` // Runtime specific (host, context, etc)
    // Or use specific fields if we want strong typing, but map is more flexible for "Runtime specific"
}

type EnvironmentConfig struct {
    Runtime   RuntimeConfig    `json:"runtime"`
    Providers []ProviderConfig `json:"providers"`
}

type Settings struct {
    DefaultAlias string                       `json:"default_alias"` // e.g., "local"
    Environments map[string]EnvironmentConfig `json:"environments"`  // keys: "local", "remote"
}
```

### Resolution Logic
When starting an agent:
1. Determine the target environment (alias) from CLI flag (e.g., `--env=remote`) or use `DefaultAlias`.
2. Load the `EnvironmentConfig` for that alias.
3. Configure the Runtime using `EnvironmentConfig.Runtime`.
4. Lookup the Provider settings (based on the agent's template/provider) in `EnvironmentConfig.Providers`.
5. Combine these to form the final `RunConfig`.

## Benefits
- **Clarity**: It's obvious which image is used in which context.
- **Flexibility**: "tmux" decision is coupled to the runtime environment, not the agent itself.
- **Portability**: Users can have different `remote` settings (different clusters) without changing the agent's definition.
- **Decoupling**: Removes "node" vs "root" username assumptions from the Go code.
