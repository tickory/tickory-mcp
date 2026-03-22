# Tickory MCP Server

Real-time crypto scanner alerts, ad hoc scan execution, and TradingView relay routing, delivered straight to your AI agent.

[Tickory](https://tickory.app) monitors Binance Futures markets using programmable CEL rules and TradingView relay routing. This MCP server lets any agent framework create scans, run saved or ad hoc scans, configure relay sources and routes, inspect relay traces, read alert events, and understand *why* alerts triggered — all through the [Model Context Protocol](https://modelcontextprotocol.io).

[![Tickory Server MCP server](https://glama.ai/mcp/servers/tickory/tickory-mcp/badges/card.svg)](https://glama.ai/mcp/servers/tickory/tickory-mcp)

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

## Relay flow example

```
You: "Set up a TradingView relay to my Telegram"

Agent (via tickory_create_relay_source): Created source "Momentum Strategy".
  Webhook URL: https://api.tickory.app/api/webhooks/tradingview/src_123
  Payload template includes your source secret and TradingView placeholders.

Agent (via tickory_add_relay_route): Telegram relay route created.

You: "Why didn't my last webhook alert send?"

Agent (via tickory_list_relay_events): 1 recent event found for source src_123.
Agent (via tickory_get_relay_trace): Route route_123 failed downstream with timeout.
Agent (via tickory_replay_relay_event): Failed route queued for replay.
```

## Install

### npm / npx

```bash
npx @tickory/mcp
```

The npm package downloads the matching GitHub Release binary for macOS and Linux during install, so `npx @tickory/mcp` works without a separate build step.

### Go install

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

Relay workflows require API keys with both `manage_routing` and `read_events`. Saved-scan and ad hoc scan execution both require `manage_scans`.

### Claude Desktop

Add to your `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "tickory": {
      "command": "npx",
      "args": ["@tickory/mcp"],
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
      "command": "npx",
      "args": ["@tickory/mcp"],
      "env": {
        "TICKORY_API_BASE_URL": "https://api.tickory.app",
        "TICKORY_API_KEY": "tk_xxxxxxxx_yyyyyyyyyyyyyyyyyyyyyyyy"
      }
    }
  }
}
```

### Cursor / Windsurf / other MCP clients

Point either to `npx @tickory/mcp` or to the standalone `tickory-mcp` binary with the environment variables above. The server uses stdio (newline-delimited JSON-RPC 2.0).

## Tools

| Tool | Description |
|------|-------------|
| `tickory_list_scans` | List scans visible to the API key owner |
| `tickory_get_scan` | Fetch one scan by ID |
| `tickory_create_scan` | Create a new scan with CEL expression and hard gates |
| `tickory_update_scan` | Replace an existing scan definition |
| `tickory_run_scan` | Trigger a scan run immediately |
| `tickory_run_ad_hoc_scan` | Execute a one-off expression immediately without creating a saved scan |
| `tickory_describe_indicators` | Describe available CEL variables, recommended guards, and example expressions |
| `tickory_list_alert_events` | List alert events with cursor pagination |
| `tickory_get_alert_event` | Fetch one alert event by UUID |
| `tickory_explain_alert_event` | Explain why an alert triggered or was suppressed |
| `tickory_create_relay_source` | Create a TradingView relay source and return the paste-ready TradingView setup payload |
| `tickory_list_relay_sources` | List TradingView relay sources and direct-route summaries |
| `tickory_add_relay_route` | Add one direct relay route to telegram, webhook, discord, or email |
| `tickory_list_relay_events` | List recent inbound relay events for one TradingView source |
| `tickory_get_relay_trace` | Fetch the full lifecycle trace for one relay source event |
| `tickory_replay_relay_event` | Replay one failed relay route when the backend allows it |

All tools return `schema_version: "v1"` for contract stability.

`tickory_run_scan` executes an existing saved scan by `scan_id`. `tickory_run_ad_hoc_scan` executes a one-off expression and does not create or update a saved scan, so the returned run payload may have an empty `scan_id`.

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
| 5xx | `upstream_error` | Yes |

## Scoped API key permissions

| Scope | What it allows |
|-------|---------------|
| `read_events` | Read alert events, relay traces, scan runs, activity |
| `manage_scans` | Create/read/update/delete scans, execute scans |
| `manage_routing` | Create/manage relay sources and routes |

Create keys with the minimum scopes needed. See the [developer docs](https://tickory.app/docs/developers) for details.

## Protocol versions

This server negotiates MCP protocol versions: `2024-11-05`, `2025-03-26`, `2025-06-18`, `2025-11-05`, and `2025-11-25`.

## License

MIT