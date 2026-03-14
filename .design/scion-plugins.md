# Scion Plugin System Design

## Motivation

Scion currently hard-codes all message broker implementations (in-process only) and harness implementations (claude, gemini, opencode, codex, generic) directly into the binary. As we add external message brokers (NATS, Redis, etc.) and potentially new harnesses, this approach does not scale:

- Every new implementation increases binary size and dependency surface
- Users cannot add custom integrations without forking the project
- The hub/broker server carries code for harnesses it may never use

We want a **plugin system** that allows scion to load additional message broker and harness implementations at runtime from external binaries.

## Technology: hashicorp/go-plugin

[hashicorp/go-plugin](https://github.com/hashicorp/go-plugin) provides the foundation:

- **Subprocess model**: Each plugin runs as a separate OS process, communicating via gRPC (or net/rpc)
- **Crash isolation**: A plugin crash does not bring down the host
- **Language agnostic**: gRPC plugins can be written in any language
- **Versioning**: Protocol version negotiation between host and plugin
- **Security**: Magic cookie handshake prevents accidental plugin execution; optional mTLS
- **Health checking**: Built-in gRPC health service

### Key go-plugin Lifecycle

1. Host calls `plugin.NewClient()` with the path to a plugin binary
2. Host calls `client.Client()` then `raw.Dispense("pluginName")` to get a typed interface
3. The plugin subprocess starts and stays running for the lifetime of the `Client`
4. Host calls methods on the dispensed interface; these become gRPC calls
5. `client.Kill()` terminates the subprocess (graceful then force after 2s)

### Long-Running vs Per-Use

go-plugin is designed for **long-lived subprocesses**. The client starts the process once and reuses it for all calls. Per-invocation usage (start, call, kill) is technically possible but adds process-spawn overhead on every call.

**Implications for scion:**

| Plugin Type | Lifecycle | Rationale |
|---|---|---|
| Message Broker | Long-running | Brokers maintain connections, subscriptions, state. Must persist for the hub/broker server lifetime. |
| Harness | Per-agent-lifecycle | Harness methods are called during agent create/start/provision. Could be long-running (shared across agents) or per-use. |

**Recommendation**: Use long-running plugin processes for both types. For harnesses, one plugin process serves all agents using that harness - the overhead of keeping it alive is negligible vs. respawning per agent operation.

## Plugin Types

### Type 1: Message Broker (`broker`)

Implements the `broker.MessageBroker` interface over gRPC:

```
service MessageBrokerPlugin {
  rpc Publish(PublishRequest) returns (PublishResponse);
  rpc Subscribe(SubscribeRequest) returns (stream Message);
  rpc Unsubscribe(UnsubscribeRequest) returns (UnsubscribeResponse);
  rpc Close(CloseRequest) returns (CloseResponse);
}
```

**Key considerations:**
- Subscriptions are inherently streaming - gRPC server streaming maps well
- The plugin maintains the external connection (NATS, Redis, etc.)
- Configuration (connection URLs, auth) passed via gRPC `Configure()` call at startup
- Plugin must handle reconnection to the backing service internally

### Type 2: Harness (`harness`)

Implements the `api.Harness` interface over gRPC. The current interface has ~15 methods, most of which are simple getters or file operations.

**Key considerations:**
- `GetHarnessEmbedsFS()` returns an `embed.FS` - cannot cross process boundaries. Plugin must instead expose a `GetEmbeddedFiles()` RPC that returns file contents.
- `Provision()` operates on the local filesystem (agent home dir). The plugin process must have filesystem access to the same paths.
- Some methods are pure data (`Name()`, `GetEnv()`, `GetCommand()`) and could be batched into a single `GetMetadata()` call to reduce round-trips.
- Optional interfaces (`AuthSettingsApplier`, `TelemetrySettingsApplier`) need capability advertisement.

## Plugin Discovery and Loading

### Filesystem Layout

```
~/.scion/plugins/
  broker/
    scion-plugin-nats        # Message broker plugin
    scion-plugin-redis       # Message broker plugin
  harness/
    scion-plugin-cursor      # Harness plugin
    scion-plugin-aider       # Harness plugin
```

Plugin binaries follow a naming convention: `scion-plugin-<name>`.

### Settings Configuration

Add a `plugins` section to settings:

```yaml
plugins:
  broker:
    nats:
      path: ~/.scion/plugins/broker/scion-plugin-nats  # optional, auto-discovered if omitted
      config:
        url: "nats://localhost:4222"
        credentials_file: "/path/to/creds"
  harness:
    cursor:
      path: ~/.scion/plugins/harness/scion-plugin-cursor
      config:
        image: "cursor-agent:latest"
        user: "cursor"
```

**Discovery order:**
1. Explicit `path` in settings
2. Scan `~/.scion/plugins/<type>/` directory
3. Search `$PATH` for `scion-plugin-<name>` (lower priority, optional)

### Active Plugin Selection

For message brokers, the active broker is selected in server config:

```yaml
# In hub/broker server config
message_broker: nats   # selects the "nats" plugin (or "inprocess" for built-in)
```

For harnesses, plugin harnesses are available alongside built-in ones. The harness factory (`harness.New()`) checks plugins after built-in types:

```go
func New(harnessName string) api.Harness {
    switch harnessName {
    case "claude": return &ClaudeCode{}
    // ... built-in harnesses
    default:
        if plugin, ok := pluginRegistry.GetHarness(harnessName); ok {
            return plugin
        }
        return &Generic{}
    }
}
```

## Plugin Registration

### Static Registration (Settings-based)

Plugins are declared in settings and loaded at startup. This is sufficient for most use cases:

- CLI reads settings, loads relevant plugins when needed
- Hub/broker server loads all configured plugins at startup
- No runtime registration needed

### Dynamic Self-Registration (Hub API)

For hub-managed deployments, plugins could self-register via a hub API endpoint. This is useful when:

- Plugins are deployed as sidecars alongside the hub/broker
- Plugin availability changes at runtime
- Centralized plugin inventory is needed for multi-broker coordination

**Proposed endpoint:**

```
POST /api/v1/plugins/register
{
  "type": "broker",
  "name": "nats",
  "version": "1.0.0",
  "capabilities": ["publish", "subscribe", "durable-subscriptions"],
  "endpoint": "unix:///var/run/scion-plugin-nats.sock"
}
```

**Recommendation**: Start with static (settings-based) registration. Add dynamic registration as a future enhancement only if operational patterns require it. The static approach is simpler, debuggable, and covers the primary use cases.

## Local Mode Support

**Should plugins work in local (non-hub) mode?**

| Plugin Type | Local Mode? | Rationale |
|---|---|---|
| Message Broker | No (initially) | Messaging is a hub/broker feature. Local mode uses the CLI directly - no pub/sub needed. |
| Harness | Yes | A user may want to use a custom harness (e.g., Cursor, Aider) in local mode. The harness interface is used for agent create/start regardless of hub vs local. |

For harness plugins in local mode:
- Plugin process is started on-demand when an agent using that harness is created/started
- Plugin process is kept alive for the duration of the CLI command
- Cleaned up on CLI exit (go-plugin handles this via `CleanupClients()`)

## Implementation Architecture

### Core Package: `pkg/plugin`

```
pkg/plugin/
  manager.go          # Plugin lifecycle management (load, start, stop, health)
  registry.go         # Type-safe plugin registry
  discovery.go        # Filesystem scanning and settings-based discovery
  config.go           # Plugin configuration types
  broker_plugin.go    # gRPC client wrapper for MessageBroker plugins
  harness_plugin.go   # gRPC client wrapper for Harness plugins
  proto/
    broker.proto      # Protobuf definitions for broker plugin interface
    harness.proto     # Protobuf definitions for harness plugin interface
    shared.proto      # Common types
```

### Plugin Manager

Central component that owns plugin lifecycle:

```go
type Manager struct {
    clients  map[string]*plugin.Client  // type:name -> client
    mu       sync.RWMutex
}

func (m *Manager) LoadAll(cfg PluginsConfig) error     // Load from settings
func (m *Manager) Get(pluginType, name string) (interface{}, error)
func (m *Manager) GetBroker(name string) (broker.MessageBroker, error)
func (m *Manager) GetHarness(name string) (api.Harness, error)
func (m *Manager) Shutdown()                            // Kill all plugins
```

### Integration Points

**Hub Server** (`pkg/hub/server.go`):
- `Server` receives a `*plugin.Manager` at construction
- If `message_broker` setting names a plugin, dispense broker from manager
- Plugin broker replaces the in-process broker in `MessageBrokerProxy`

**Runtime Broker** (`pkg/runtimebroker/server.go`):
- Similar to hub - receives plugin manager for harness plugins
- When creating agents with a plugin harness, dispense from manager

**CLI** (`cmd/`):
- For local harness plugins: create a temporary manager, load needed plugin, use, cleanup
- No broker plugins in local mode

**Harness Factory** (`pkg/harness/harness.go`):
- Accept optional `*plugin.Manager` parameter
- Fall through to plugin lookup before defaulting to `Generic`

## gRPC Interface Design Considerations

### Broker Plugin

The `broker.MessageBroker` interface maps cleanly to gRPC with one exception: `Subscribe()` returns a callback-based handler. Over gRPC, this becomes a server-streaming RPC where the plugin streams messages back to the host.

**Host-side adapter:**
```go
type brokerPluginClient struct {
    client proto.MessageBrokerPluginClient
}

func (b *brokerPluginClient) Subscribe(pattern string, handler MessageHandler) (Subscription, error) {
    stream, err := b.client.Subscribe(ctx, &SubscribeRequest{Pattern: pattern})
    // Goroutine reads from stream and calls handler
}
```

### Harness Plugin

The harness interface has several methods that don't translate directly:

| Method | Challenge | Solution |
|---|---|---|
| `GetHarnessEmbedsFS()` | Returns `embed.FS` | Replace with `GetEmbeddedFiles()` returning `map[string][]byte` |
| `Provision()` | Writes to local filesystem | Plugin must access same filesystem; pass paths and let plugin write |
| `InjectAgentInstructions()` | Writes to local filesystem | Same as Provision |
| `ResolveAuth()` | Complex types | Serialize `AuthConfig`/`ResolvedAuth` as protobuf messages |

**Capability advertisement**: Plugin responds to a `GetCapabilities()` call indicating which optional interfaces it supports (auth settings, telemetry settings).

## Open Questions

### 1. Harness Embed Files Over Plugin Boundary

Built-in harnesses use `//go:embed` to package default config files. For plugin harnesses, these files must be transmitted over gRPC. Options:
- **a)** Plugin transmits files on demand via `GetEmbeddedFiles()` RPC
- **b)** Plugin ships a companion tarball that scion extracts on install
- **c)** Plugin writes its own embeds during `Provision()` directly

Recommendation: **(a)** is simplest and keeps the plugin self-contained.

### 2. Plugin Versioning and Compatibility

go-plugin supports protocol version negotiation. We need to define:
- What constitutes a breaking change to the plugin protocol?
- Should scion refuse to load plugins with incompatible versions, or degrade gracefully?
- How do we communicate minimum scion version requirements from the plugin side?

### 3. Plugin Configuration Schema

How does the host know what config a plugin needs?
- **a)** Plugin exposes a `GetConfigSchema()` RPC returning a JSON Schema
- **b)** Config is opaque `map[string]string` passed to `Configure()` - plugin validates
- **c)** Plugin documentation describes required config

Recommendation: **(b)** for v1 - keep it simple. Plugin validates its own config and returns clear errors.

### 4. Security Model

- Plugin binaries are user-installed in `~/.scion/plugins/` - same trust as any local binary
- go-plugin's magic cookie prevents accidental execution
- Should we support signature verification for plugin binaries? (future consideration)
- mTLS between host and plugin? (go-plugin supports this but adds complexity)

### 5. Multiple Broker Plugins Simultaneously?

Can a hub run multiple message broker plugins (e.g., NATS for inter-agent messaging, Redis for notifications)?
- Current `MessageBroker` interface assumes one active broker
- Could support named broker instances: `plugins.broker.nats` and `plugins.broker.redis` with different roles
- Defer this to future design if the need arises

### 6. Plugin Distribution

How do users obtain plugins?
- Manual download to `~/.scion/plugins/<type>/`
- `scion plugin install <name>` fetching from a registry or GitHub releases
- Package managers (brew, apt, etc.)

Distribution is out of scope for v1 but the filesystem layout should accommodate future tooling.

### 7. Hot Reload

Can plugins be reloaded without restarting the hub/broker server?
- go-plugin supports reattachment to existing plugin processes
- The plugin manager could watch for binary changes and restart plugins
- Adds complexity; defer to future versions

### 8. Logging and Observability

go-plugin captures plugin stdout/stderr and routes it through the host's logger. We should:
- Use structured (JSON) logging from plugins for clean integration
- Include plugin name/type in log context
- Expose plugin health status through the hub's existing health endpoint

## Phased Implementation Plan

### Phase 1: Plugin Infrastructure
- `pkg/plugin/` package with Manager, Registry, Discovery
- Protobuf definitions for broker plugin interface
- Settings schema additions for `plugins` section
- Integration with hub server for broker plugins

### Phase 2: Message Broker Plugins
- NATS broker plugin (first external implementation)
- Test the full lifecycle: discovery, loading, configuration, operation, shutdown
- Validate streaming subscription model over gRPC

### Phase 3: Harness Plugins
- Protobuf definitions for harness plugin interface
- Adapter for embed.FS replacement
- Integration with harness factory and local mode
- Example harness plugin

### Phase 4: Polish
- `scion plugin list` command showing discovered/loaded plugins
- Health status reporting
- Documentation and plugin authoring guide

## Related Design Documents

- [Message Broker](hosted/hub-messaging.md) - Current messaging architecture
- [Hosted Architecture](hosted/hosted-architecture.md) - Hub/broker separation
- [Server Implementation](hosted/server-implementation-design.md) - Unified server command
- [Settings Schema](settings-schema.md) - Settings configuration format
