---
title: Configuration Overview
---

Scion uses a multi-layered configuration system to manage orchestrator behavior, agent execution, and server operations. Settings files use a versioned format identified by the `schema_version` field. The current version is `1`.

## Configuration Domains

The documentation is divided into the following domains:

### [Orchestrator Settings (settings.yaml)](/reference/orchestrator-settings/)
Global and project-level settings for the `scion` CLI and orchestrator. Defines Runtimes, Harness Configs, and execution Profiles. Uses the versioned format with `schema_version`.

### [Agent & Template Configuration (scion-agent.json)](/reference/agent-config/)
Configuration for agent blueprints (templates) and individual agent instances. Defines container images, volumes, and environment variables.

### [Server Configuration (Hub & Runtime Broker)](/reference/server-config/)
Operational settings for the Scion Hub and Runtime Broker services. Server configuration is now part of `settings.yaml` under the `server` key. The standalone `server.yaml` is deprecated but still supported as a fallback.

### [Web Dashboard Configuration](/reference/web-config/)
Environment variables and settings for the Web Dashboard frontend and BFF.

### [Harness-Specific Settings](/reference/harness-settings/)
Guide to configuring the LLM tools and harnesses running *inside* the agent containers.

---

## Key Files

| File | Purpose |
| :--- | :--- |
| `settings.yaml` | Primary configuration file (global and grove-level). Contains orchestrator settings and optionally server configuration under the `server` key. |
| `state.yaml` | Runtime-managed state (e.g., `last_synced_at`). Not user-edited. |
| `scion-agent.json` | Agent and template configuration. |

> **Note**: The standalone `server.yaml` is deprecated. Use `scion config migrate --server` to consolidate it into `settings.yaml`.

## Resolution Hierarchy

Scion typically resolves configuration using the following precedence (from highest to lowest):

1. **CLI Flags**: `--hub`, `--profile`, etc.
2. **Environment Variables**: `SCION_*` and `SCION_SERVER_*`.
3. **Grove Settings**: `.scion/settings.yaml` in the current project.
4. **Global Settings**: `~/.scion/settings.yaml` in the user's home directory.
5. **Defaults**: Hardcoded system defaults.

## Migration

Legacy settings files (without `schema_version`) can be migrated to the versioned format using the `scion config migrate` command:

```bash
# Preview changes
scion config migrate --dry-run

# Migrate all legacy settings files
scion config migrate

# Migrate server.yaml into settings.yaml
scion config migrate --server
```

See the [Orchestrator Settings](/reference/orchestrator-settings/) and [Server Configuration](/reference/server-config/) pages for details.
