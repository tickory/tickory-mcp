# Tickory MCP Server

Real-time crypto scanner alerts, delivered straight to your AI agent.

[Tickory](https://tickory.app) monitors Binance Futures markets using programmable CEL rules and sends alerts when conditions match. This MCP server lets any agent framework create scans, read alert events, and understand *why* alerts triggered — all through the [Model Context Protocol](https://modelcontextprotocol.io).

## Quick demo

```
You: "Create a scan that fires when RSI drops below 30 on any coin with volume over $100k"

Agent (via tickory_create_scan): Done — scan "Oversold Bounce" created.

You: "Run it now"

Agent (via tickory_run_scan): 3 matches found: ETHUSDT, SOLUSDT, DOGEUSDT.

You: "Why did ETH match?"

Agent (via tickory_explain_alert_event): RSI-14 was 24.7, below your threshold of 30.
  Volume gate passed: $487k USDT > $100k minimum. CEL expression evaluated true.
```

## Install

### Go install (recommended)

```bash
go install github.com/tickory/tickory-mcp@latest
```

### Pre-built binaries

Download from [GitHub Releases](https://github.com/tickory/tickory-mcp/releases) for Linux and macOS (amd64/arm64).

### Build from source

```bash
git clone https://github.com/tickory/tickory-mcp.git
cd tickory-mcp
go build -o tickory-mcp .
```

## Setup

1. **Get a Tickory account** at [tickory.app](https://tickory.app)
2. **Create a scoped API key** in your dashboard under Settings > API Keys
3. **Configure the MCP server**:

```bash
export TICKORY_API_BASE_URL=https://api.tickory.app
export TICKORY_API_KEY=tk_xxxxxxxx_yyyyyyyyyyyyyyyyyyyyyyyy
```

### Claude Desktop

Add to your `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "tickory": {
      "command": "tickory-mcp",
      "env": {
        "TICKORY_API_BASE_URL": "https://api.tickory.app",
        "TICKORY_API_KEY": "tk_xxxxxxxx_yyyyyyyyyyyyyyyyyyyyyyyy"
      }
    }
  }
}
```

### Claude Code

Add to your `.mcp.json`:

```json
{
  "mcpServers": {
    "tickory": {
      "command": "tickory-mcp",
      "env": {
        "TICKORY_API_BASE_URL": "https://api.tickory.app",
        "TICKORY_API_KEY": "tk_xxxxxxxx_yyyyyyyyyyyyyyyyyyyyyyyy"
      }
    }
  }
}
```

### Cursor / Windsurf / other MCP clients

Point to the `tickory-mcp` binary with the environment variables above. The server uses stdio (newline-delimited JSON-RPC 2.0).

## Tools

| Tool | Description |
|------|-------------|
| `tickory_list_scans` | List scans visible to the API key owner |
| `tickory_get_scan` | Fetch one scan by ID |
| `tickory_create_scan` | Create a new scan with CEL expression and hard gates |
| `tickory_update_scan` | Replace an existing scan definition |
| `tickory_run_scan` | Trigger a scan run immediately |
| `tickory_list_alert_events` | List alert events with cursor pagination |
| `tickory_get_alert_event` | Fetch one alert event by UUID |
| `tickory_explain_alert_event` | Explain why an alert triggered or was suppressed |

All tools return `schema_version: "v1"` for contract stability.

## Configuration

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `TICKORY_API_BASE_URL` | Yes | — | Tickory API base URL |
| `TICKORY_API_KEY` | Yes | — | Scoped API key (`tk_...`) |
| `TICKORY_TIMEOUT_SECONDS` | No | `15` | HTTP timeout for API requests |

CLI flags (`--api-base-url`, `--api-key`, `--timeout`) override environment variables.

## Error handling

Upstream HTTP errors are mapped to deterministic MCP error codes:

| HTTP Status | MCP Code | Retryable |
|------------|----------|-----------|
| 400 | `invalid_request` | No |
| 401 | `unauthorized` | No |
| 403 | `forbidden` | No |
| 404 | `not_found` | No |
| 409 | `conflict` | No |
| 429 | `rate_limited` | Yes |
| 5xx | `upstream_unavailable` | Yes |

## Scoped API key permissions

| Scope | What it allows |
|-------|---------------|
| `read_events` | Read alert events, scan runs, activity |
| `manage_scans` | Create/read/update/delete scans, execute scans |
| `manage_routing` | Create/manage alert sources and routes |

Create keys with the minimum scopes needed. See the [developer docs](https://tickory.app/docs/developers) for details.

## Protocol versions

This server negotiates MCP protocol versions: `2024-11-05`, `2025-03-26`, `2025-06-18`, `2025-11-05`, and `2025-11-25`.

## License

MIT

