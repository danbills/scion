---
title: Orchestrator Settings (settings.yaml)
---

This document describes the configuration for the Scion orchestrator, managed through `settings.yaml` (or `settings.json`) files.

## Purpose
The orchestrator settings define the available infrastructure components (Runtimes), the tools that can be run (Harness Configs), and how they are combined into environments (Profiles). It also contains client-side configuration for connecting to a Scion Hub.

## Versioned Format

Settings files use a versioned format identified by the `schema_version` field. The current version is `1`.

```yaml
schema_version: "1"
active_profile: local
default_template: gemini
```

Files without `schema_version` are treated as legacy format. Run `scion config migrate` to convert legacy files to the versioned format.

## Locations
- **Global**: `~/.scion/settings.yaml` (User-wide defaults)
- **Grove**: `.scion/settings.yaml` (Project-specific overrides)

Settings are resolved in order: Defaults → Global → Grove → Environment Variables. Later sources override earlier ones.

## Top-Level Fields

| Field | Type | Description |
| :--- | :--- | :--- |
| `schema_version` | string | Schema version identifier. Currently `"1"`. |
| `active_profile` | string | The active profile name (e.g., `local`, `remote`). |
| `default_template` | string | Default agent template (e.g., `gemini`, `claude`). |

## Hub Client Configuration (`hub`)
Settings for connecting the CLI to a Scion Hub.

| Field | Type | Description |
| :--- | :--- | :--- |
| `enabled` | bool | Whether to enable Hub integration for this grove. |
| `endpoint` | string | The Hub API endpoint URL (e.g., `https://hub.scion.dev`). |
| `grove_id` | string | The unique identifier for this grove on the Hub. |
| `local_only` | bool | If true, operate in local-only mode even if Hub is configured. |

> **Note**: The legacy fields `token`, `apiKey`, `hostId`, `hostNickname`, `brokerId`, `brokerNickname`, and `brokerToken` have been removed from the versioned format. Broker identity fields are now under `server.broker`. Runtime state such as `lastSyncedAt` is stored in `state.yaml`.

## Harness Configs (`harness_configs`)
Named configurations for agent harnesses. Each entry defines a harness type and its container settings.

```yaml
harness_configs:
  gemini:
    harness: gemini
    image: example.com/scion-gemini:latest
    user: scion
    model: gemini-2.5-pro
    args: ["--sandbox=strict"]
    env:
      MY_VAR: value
    volumes:
      - source: /host/path
        target: /container/path
```

| Field | Type | Description |
| :--- | :--- | :--- |
| `harness` | string | **(Required)** The harness type this config applies to (e.g., `gemini`, `claude`). |
| `image` | string | Container image to use. |
| `user` | string | User to run as inside the container. |
| `model` | string | Default model for this harness. |
| `args` | list | Additional arguments passed to the harness. |
| `env` | map | Environment variables for the container. |
| `volumes` | list | Volume mounts for the container. |
| `auth_selected_type` | string | Authentication type selection. |

> **Note**: The legacy `harnesses` key (without the `harness` field) is deprecated. Use `harness_configs` with the required `harness` field instead.

## Runtimes (`runtimes`)
Configuration for execution backends.

```yaml
runtimes:
  docker:
    type: docker
    host: tcp://localhost:2375
  container:
    type: container
  my-k8s:
    type: kubernetes
    namespace: scion
    context: prod-cluster
```

| Field | Type | Description |
| :--- | :--- | :--- |
| `type` | string | The runtime type (`docker`, `container`, `kubernetes`). Defaults to the map key name. |
| `host` | string | Runtime host address. |
| `context` | string | Kubernetes context name. |
| `namespace` | string | Kubernetes namespace. |
| `tmux` | bool | Whether to use tmux for terminal management. |
| `env` | map | Environment variables for the runtime. |
| `sync` | string | File sync strategy. |

## Profiles (`profiles`)
Named configurations that bind a runtime to harness settings and overrides.

```yaml
profiles:
  local:
    runtime: docker
    default_template: gemini
    default_harness_config: gemini
    tmux: true
    env:
      ENVIRONMENT: development
    harness_overrides:
      gemini:
        image: example.com/gemini:dev
```

| Field | Type | Description |
| :--- | :--- | :--- |
| `runtime` | string | **(Required)** Name of the runtime to use (must exist in `runtimes`). |
| `default_template` | string | Default template for agents created under this profile. |
| `default_harness_config` | string | Default harness config for agents under this profile. |
| `tmux` | bool | Whether to use tmux for terminal management. |
| `env` | map | Environment variables merged into harness and runtime configs. |
| `volumes` | list | Additional volume mounts appended to harness volumes. |
| `resources` | object | Resource requests/limits for the runtime (CPU, memory, disk). |
| `harness_overrides` | map | Per-harness overrides (image, user, env, volumes, auth_selected_type). |

## CLI Configuration (`cli`)
General CLI behavior settings.

| Field | Type | Description |
| :--- | :--- | :--- |
| `autohelp` | bool | Whether to print usage help on every error. |
| `interactive_disabled` | bool | Disable interactive prompts. |

## Environment Variable Overrides
Most settings can be overridden using environment variables with the `SCION_` prefix.

| Environment Variable | Maps To |
| :--- | :--- |
| `SCION_ACTIVE_PROFILE` | `active_profile` |
| `SCION_DEFAULT_TEMPLATE` | `default_template` |
| `SCION_HUB_ENDPOINT` | `hub.endpoint` |
| `SCION_HUB_GROVE_ID` | `hub.grove_id` |
| `SCION_HUB_ENABLED` | `hub.enabled` |
| `SCION_HUB_LOCAL_ONLY` | `hub.local_only` |
| `SCION_CLI_AUTOHELP` | `cli.autohelp` |
| `SCION_CLI_INTERACTIVE_DISABLED` | `cli.interactive_disabled` |

## Migration

To migrate legacy settings files to the versioned format:

```bash
# Preview what would change
scion config migrate --dry-run

# Migrate all settings files
scion config migrate

# Migrate only global settings
scion config migrate --global
```

The migration creates a backup of original files (`.bak`) and converts `harnesses` to `harness_configs`, removes deprecated Hub fields, and migrates `hub.lastSyncedAt` to `state.yaml`.

For a detailed walkthrough of orchestrator settings and environment variable substitution, see the [Local Governance Guide](/guides/local-governance/).
