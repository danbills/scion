---
title: Server Configuration (Hub & Runtime Broker)
---

This document describes the configuration for the Scion Hub (State Server) and the Scion Runtime Broker services.

## Purpose
Server configuration controls the operational behavior of the Scion backend components in a "Hosted" or distributed architecture. This includes network settings, database connections, and security configurations.

## Locations
- **Primary**: `~/.scion/settings.yaml` under the `server` key (versioned format).
- **Legacy fallback**: `~/.scion/server.yaml` — still supported but deprecated. The `settings.yaml` `server` key takes precedence when present.
- **Environment Variables**: Overridden using the `SCION_SERVER_` prefix (e.g., `SCION_SERVER_HUB_PORT`).

> **Note**: `server.yaml` is deprecated. Use `scion config migrate --server` to consolidate it into `settings.yaml`.

### Example (settings.yaml)

```yaml
schema_version: "1"
server:
  log_level: debug
  hub:
    port: 9810
    host: 0.0.0.0
    cors:
      enabled: true
      allowed_origins: ["*"]
  broker:
    enabled: true
    port: 9800
    broker_id: my-broker-id
  database:
    driver: sqlite
    url: hub.db
```

## Configuration Sections

### Hub Section (`server.hub`)
Configuration for the central Hub API server.

| Field | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| `port` | int | `9810` | The HTTP port to listen on. |
| `host` | string | `0.0.0.0` | The network address to bind to. |
| `public_url` | string | | Public-facing URL for the Hub (used in callbacks). |
| `read_timeout` | duration | `30s` | Maximum duration for reading the entire request. |
| `write_timeout` | duration | `60s` | Maximum duration before timing out writes. |
| `admin_emails` | list | `[]` | Email addresses granted admin access. |
| `cors` | object | | CORS configuration (see below). |

#### CORS Configuration (`server.hub.cors`)

| Field | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| `enabled` | bool | `true` | Whether to enable Cross-Origin Resource Sharing. |
| `allowed_origins` | list | `["*"]` | List of origins allowed to make CORS requests. |
| `allowed_methods` | list | `[...]` | Standard HTTP methods allowed (GET, POST, etc). |
| `allowed_headers` | list | `[...]` | Allowed headers including Scion-specific tokens. |
| `max_age` | int | `3600` | How long the results of a preflight request can be cached. |

### Broker Section (`server.broker`)
Configuration for the Runtime Broker service. Includes broker identity fields previously in hub client config.

| Field | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| `enabled` | bool | `false` | Whether to start the Runtime Broker API. |
| `port` | int | `9800` | The HTTP port to listen on. |
| `host` | string | `0.0.0.0` | The network address to bind to. |
| `read_timeout` | duration | `30s` | Maximum duration for reading requests. |
| `write_timeout` | duration | `60s` | Maximum duration before timing out writes. |
| `hub_endpoint` | string | | The Hub API endpoint for status reporting. |
| `broker_id` | string | (auto) | Unique identifier for this broker (persisted in settings). |
| `broker_name` | string | | Human-readable name for this runtime broker. |
| `broker_nickname` | string | | Short display name for this broker. |
| `broker_token` | string | | Token received when registering with the Hub. |
| `cors` | object | | CORS configuration (same schema as `server.hub.cors`). |

### Database Section (`server.database`)
Persistence settings for the Hub.

| Field | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| `driver` | string | `sqlite` | Database driver (`sqlite` or `postgres`). |
| `url` | string | `hub.db` | Connection path for SQLite or DSN for PostgreSQL. |

### Auth Section (`server.auth`)
Settings for development and domain authorization.

| Field | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| `dev_mode` | bool | `false` | Enable development token authentication. **Not for production.** |
| `dev_token` | string | | An explicitly configured development token. |
| `dev_token_file` | string | `~/.scion/dev-token` | Path to the auto-generated development token file. |
| `authorized_domains` | list | `[]` | List of email domains allowed to authenticate. Empty allows all. |

### OAuth Section (`server.oauth`)
OAuth credentials for Web, CLI, and Device clients.

```yaml
server:
  oauth:
    web:
      google:
        client_id: "..."
        client_secret: "..."
      github:
        client_id: "..."
        client_secret: "..."
    cli:
      google:
        client_id: "..."
    device:
      google:
        client_id: "..."
```

### Storage Section (`server.storage`)
Storage backend settings for template and artifact persistence.

| Field | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| `provider` | string | `local` | Storage provider (`local`, `gcs`). |
| `bucket` | string | | Cloud storage bucket name. |
| `local_path` | string | | Local filesystem path for storage. |

### Secrets Section (`server.secrets`)
Secrets backend configuration.

| Field | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| `backend` | string | `local` | Secrets backend (`local`, `gcp`). |
| `gcp_project_id` | string | | GCP project ID for Secret Manager. |
| `gcp_credentials` | string | | Path to GCP credentials file. |

### Logging (`server.log_level`, `server.log_format`)

| Field | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| `log_level` | string | `info` | Logging verbosity (`debug`, `info`, `warn`, `error`). |
| `log_format` | string | `text` | Log output format (`text` or `json`). |

## Environment Variables
Server settings use a nested naming convention for environment variables with the `SCION_SERVER_` prefix.

| Environment Variable | Maps To |
| :--- | :--- |
| `SCION_SERVER_LOG_LEVEL` | `server.log_level` |
| `SCION_SERVER_LOG_FORMAT` | `server.log_format` |
| `SCION_SERVER_HUB_PORT` | `server.hub.port` |
| `SCION_SERVER_HUB_HOST` | `server.hub.host` |
| `SCION_SERVER_HUB_PUBLIC_URL` | `server.hub.public_url` |
| `SCION_SERVER_HUB_READ_TIMEOUT` | `server.hub.read_timeout` |
| `SCION_SERVER_HUB_WRITE_TIMEOUT` | `server.hub.write_timeout` |
| `SCION_SERVER_HUB_CORS_ENABLED` | `server.hub.cors.enabled` |
| `SCION_SERVER_BROKER_PORT` | `server.broker.port` |
| `SCION_SERVER_BROKER_BROKER_ID` | `server.broker.broker_id` |
| `SCION_SERVER_BROKER_BROKER_NAME` | `server.broker.broker_name` |
| `SCION_SERVER_BROKER_HUB_ENDPOINT` | `server.broker.hub_endpoint` |
| `SCION_SERVER_DATABASE_DRIVER` | `server.database.driver` |
| `SCION_SERVER_DATABASE_URL` | `server.database.url` |
| `SCION_SERVER_AUTH_DEV_MODE` | `server.auth.dev_mode` |
| `SCION_SERVER_AUTH_DEV_TOKEN` | `server.auth.dev_token` |
| `SCION_SERVER_AUTH_AUTHORIZED_DOMAINS` | `server.auth.authorized_domains` |
| `SCION_SERVER_STORAGE_PROVIDER` | `server.storage.provider` |
| `SCION_SERVER_SECRETS_BACKEND` | `server.secrets.backend` |

**Shorthand Environment Variables:**
- `SCION_AUTHORIZED_DOMAINS`: Maps to `auth.authorizedDomains` (comma-separated list).

## Migration

To consolidate a legacy `server.yaml` into the versioned `settings.yaml`:

```bash
# Preview what would change
scion config migrate --server --dry-run

# Migrate server.yaml into settings.yaml
scion config migrate --server
```

The migration moves all server configuration under the `server` key in `settings.yaml`, renames fields from camelCase to snake_case, nests CORS settings, and creates a backup of the original `server.yaml`.
