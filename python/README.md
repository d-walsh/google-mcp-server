# Google MCP Python API

Call the Google MCP server from Python scripts. Uses the same OAuth config as the Go server.

## Prerequisites

- Python 3.10+
- **google-mcp-server** binary in PATH or built in repo root:
  - Install via Homebrew: `brew install google-mcp-server`
  - Or build: `cd .. && go build -o google-mcp-server .`
- OAuth configured for the Go server (config.json, ~/.google-mcp-accounts/)

## Installation

From the repo root:

```bash
cd python
pip install -e .
```

Or with uv:

```bash
cd python
uv pip install -e .
```

## Configuration

Uses the same config as the Go server. Ensure:

- `config.json` in the repo root (or `~/.google-mcp-server/config.json`), or
- Env vars: `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`
- Tokens in `~/.google-mcp-accounts/` (multi-account)

Run from the repo root so the server finds `config.json`, or use the standard config paths.

## Usage

### Sync (simple scripts)

```python
from google_mcp import (
    accounts_list,
    calendar_events_list,
    calendar_list,
    drive_files_list,
    gmail_messages_list,
)

# List accounts
print(accounts_list())

# Calendar
calendars = calendar_list()
events = calendar_events_list("primary", max_results=10)
events_work = calendar_events_list("primary", account="work@example.com")

# Drive
files = drive_files_list(parent_id="root", page_size=20)

# Gmail
messages = gmail_messages_list(query="is:unread", max_results=5)
```

### Async (efficient for multiple calls)

```python
import asyncio
from google_mcp import GoogleMCPClient

async def main():
    async with GoogleMCPClient() as client:
        calendars = await client.calendar_list()
        events = await client.calendar_events_list("primary", max_results=10)
        files = await client.drive_files_list(parent_id="root")
        # Or call any tool by name:
        result = await client.call_tool("drive_markdown_upload", name="Note", markdown="# Hello")

asyncio.run(main())
```

### Custom server path

```python
from google_mcp import GoogleMCPClient

async with GoogleMCPClient(server_path="/path/to/google-mcp-server") as client:
    ...
```

## Phase 1 API

| Function | Description |
|----------|-------------|
| `accounts_list()` | List configured Google accounts |
| `calendar_list(account=None)` | List calendars |
| `calendar_events_list(calendar_id, time_min=None, time_max=None, max_results=None, account=None)` | List events |
| `calendar_event_create(calendar_id, summary, start_time, end_time, ...)` | Create event |
| `drive_files_list(parent_id=None, page_size=None, account=None)` | List Drive files |
| `drive_file_download(file_id, account=None)` | Download file |
| `drive_markdown_upload(name, markdown, parent_id=None, account=None)` | Upload Markdown as Doc |
| `gmail_messages_list(query=None, max_results=None, account=None)` | List messages |
| `gmail_message_get(message_id, account=None)` | Get message |

Additional tools can be called via `client.call_tool("tool_name", **kwargs)`.
