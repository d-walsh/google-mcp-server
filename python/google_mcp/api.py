"""
Sync Python API that wraps MCP tool calls. Each function spawns the Go server,
calls the tool, and returns the result. For token-efficient scripts and automation.
"""

from typing import Optional

from .client import GoogleMCPClient, run_async


def _with_client(coro):
    """Run an async coroutine that takes a client, by creating and closing a client."""
    async def _run():
        async with GoogleMCPClient() as client:
            return await coro(client)
    return run_async(_run())


def calendar_list(account: Optional[str] = None) -> str:
    """List all accessible calendars."""
    return _with_client(lambda c: c.calendar_list(account=account))


def calendar_events_list(
    calendar_id: str,
    time_min: Optional[str] = None,
    time_max: Optional[str] = None,
    max_results: Optional[int] = None,
    account: Optional[str] = None,
) -> str:
    """List events from a calendar with optional date range."""
    return _with_client(
        lambda c: c.calendar_events_list(
            calendar_id=calendar_id,
            time_min=time_min,
            time_max=time_max,
            max_results=max_results,
            account=account,
        )
    )


def calendar_event_create(
    calendar_id: str,
    summary: str,
    start_time: str,
    end_time: str,
    description: Optional[str] = None,
    location: Optional[str] = None,
    attendees: Optional[list[str]] = None,
    reminders: Optional[list[int]] = None,
    account: Optional[str] = None,
) -> str:
    """Create a new calendar event."""
    return _with_client(
        lambda c: c.calendar_event_create(
            calendar_id=calendar_id,
            summary=summary,
            start_time=start_time,
            end_time=end_time,
            description=description,
            location=location,
            attendees=attendees,
            reminders=reminders,
            account=account,
        )
    )


def drive_files_list(
    parent_id: Optional[str] = None,
    page_size: Optional[int] = None,
    account: Optional[str] = None,
) -> str:
    """List files and folders in Google Drive."""
    return _with_client(
        lambda c: c.drive_files_list(
            parent_id=parent_id,
            page_size=page_size,
            account=account,
        )
    )


def drive_file_download(file_id: str, account: Optional[str] = None) -> str:
    """Download a file from Google Drive."""
    return _with_client(lambda c: c.drive_file_download(file_id=file_id, account=account))


def drive_markdown_upload(
    name: str,
    markdown: str,
    parent_id: Optional[str] = None,
    account: Optional[str] = None,
) -> str:
    """Upload Markdown content as a Google Doc."""
    return _with_client(
        lambda c: c.drive_markdown_upload(
            name=name,
            markdown=markdown,
            parent_id=parent_id,
            account=account,
        )
    )


def gmail_messages_list(
    query: Optional[str] = None,
    max_results: Optional[int] = None,
    account: Optional[str] = None,
) -> str:
    """List email messages."""
    return _with_client(
        lambda c: c.gmail_messages_list(
            query=query,
            max_results=max_results,
            account=account,
        )
    )


def gmail_message_get(message_id: str, account: Optional[str] = None) -> str:
    """Get email message details."""
    return _with_client(lambda c: c.gmail_message_get(message_id=message_id, account=account))


def accounts_list() -> str:
    """List configured Google accounts."""
    return _with_client(lambda c: c.accounts_list())
