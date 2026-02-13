"""
Python API for Google MCP Server.

Use the Go-based MCP server from Python scripts. Same OAuth config as the server
(config.json, ~/.google-mcp-accounts/, env vars).

Async:
    from google_mcp import GoogleMCPClient

    async def main():
        async with GoogleMCPClient() as client:
            events = await client.calendar_events_list("primary", max_results=10)
            print(events)

Sync:
    from google_mcp import calendar_events_list, drive_files_list

    events = calendar_events_list("primary", account="work")
    files = drive_files_list(parent_id="root", page_size=5)
"""

from .api import (
    accounts_list,
    calendar_events_list,
    calendar_event_create,
    calendar_list,
    drive_file_download,
    drive_files_list,
    drive_markdown_upload,
    gmail_message_get,
    gmail_messages_list,
)
from .client import GoogleMCPClient, run_async

__all__ = [
    "GoogleMCPClient",
    "run_async",
    "accounts_list",
    "calendar_list",
    "calendar_events_list",
    "calendar_event_create",
    "drive_files_list",
    "drive_file_download",
    "drive_markdown_upload",
    "gmail_messages_list",
    "gmail_message_get",
]
