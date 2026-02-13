"""
MCP client that spawns the Go google-mcp-server and calls tools via stdio.
"""

import asyncio
import os
from pathlib import Path
from typing import Any, Optional

from mcp import ClientSession, StdioServerParameters, types
from mcp.client.stdio import stdio_client


def _default_server_path() -> str:
    """Default path to the google-mcp-server binary."""
    # Try repo-relative path first (when running from python/ in repo)
    repo_root = Path(__file__).resolve().parent.parent.parent
    local_binary = repo_root / "google-mcp-server"
    if local_binary.exists():
        return str(local_binary)
    # Fallback to PATH
    return "google-mcp-server"


def _default_server_cwd() -> Optional[str]:
    """Default cwd for the server process (so config.json in repo root is found)."""
    repo_root = Path(__file__).resolve().parent.parent.parent
    config_path = repo_root / "config.json"
    if config_path.exists():
        return str(repo_root)
    return None


class GoogleMCPClient:
    """
    Async context manager that spawns the Go MCP server and provides tool calls.
    """

    def __init__(
        self,
        server_path: Optional[str] = None,
        server_cwd: Optional[str] = None,
        env: Optional[dict[str, str]] = None,
    ):
        """
        Args:
            server_path: Path to google-mcp-server binary. Default: repo binary or PATH.
            server_cwd: Working directory for the server. Default: repo root if config.json exists.
            env: Extra environment variables for the server. Merged with os.environ.
        """
        self._server_path = server_path or _default_server_path()
        self._server_cwd = server_cwd if server_cwd is not None else _default_server_cwd()
        self._env = {**os.environ, **(env or {})}
        self._session: Optional[ClientSession] = None
        self._read = None
        self._write = None
        self._stdio_context = None

    async def __aenter__(self) -> "GoogleMCPClient":
        params = StdioServerParameters(
            command=self._server_path,
            args=[],
            env=self._env,
        )
        self._stdio_context = stdio_client(params)
        self._read, self._write = await self._stdio_context.__aenter__()
        self._session = ClientSession(self._read, self._write)
        await self._session.__aenter__()
        await self._session.initialize()
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb) -> None:
        if self._session:
            await self._session.__aexit__(exc_type, exc_val, exc_tb)
            self._session = None
        if self._stdio_context:
            await self._stdio_context.__aexit__(exc_type, exc_val, exc_tb)
            self._stdio_context = None
        self._read, self._write = None, None

    async def call_tool(self, name: str, **kwargs: Any) -> str:
        """
        Call an MCP tool by name. Drops None values from kwargs.
        Returns the tool result as text.
        """
        if self._session is None:
            raise RuntimeError("Client not connected. Use 'async with GoogleMCPClient()'.")
        args = {k: v for k, v in kwargs.items() if v is not None}
        result = await self._session.call_tool(name, arguments=args)
        if not result.content:
            return ""
        text_parts = []
        for block in result.content:
            if isinstance(block, types.TextContent):
                text_parts.append(block.text)
        return "\n".join(text_parts)

    # Phase 1 tool shortcuts
    async def calendar_list(self, account: Optional[str] = None) -> str:
        return await self.call_tool("calendar_list", account=account)

    async def calendar_events_list(
        self,
        calendar_id: str,
        time_min: Optional[str] = None,
        time_max: Optional[str] = None,
        max_results: Optional[int] = None,
        account: Optional[str] = None,
    ) -> str:
        return await self.call_tool(
            "calendar_events_list",
            calendar_id=calendar_id,
            time_min=time_min,
            time_max=time_max,
            max_results=max_results,
            account=account,
        )

    async def calendar_event_create(
        self,
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
        return await self.call_tool(
            "calendar_event_create",
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

    async def drive_files_list(
        self,
        parent_id: Optional[str] = None,
        page_size: Optional[int] = None,
        account: Optional[str] = None,
    ) -> str:
        return await self.call_tool(
            "drive_files_list",
            parent_id=parent_id,
            page_size=page_size,
            account=account,
        )

    async def drive_file_download(self, file_id: str, account: Optional[str] = None) -> str:
        return await self.call_tool("drive_file_download", file_id=file_id, account=account)

    async def drive_markdown_upload(
        self,
        name: str,
        markdown: str,
        parent_id: Optional[str] = None,
        account: Optional[str] = None,
    ) -> str:
        return await self.call_tool(
            "drive_markdown_upload",
            name=name,
            markdown=markdown,
            parent_id=parent_id,
            account=account,
        )

    async def gmail_messages_list(
        self,
        query: Optional[str] = None,
        max_results: Optional[int] = None,
        account: Optional[str] = None,
    ) -> str:
        return await self.call_tool(
            "gmail_messages_list",
            query=query,
            max_results=max_results,
            account=account,
        )

    async def gmail_message_get(self, message_id: str, account: Optional[str] = None) -> str:
        return await self.call_tool("gmail_message_get", message_id=message_id, account=account)

    async def accounts_list(self) -> str:
        return await self.call_tool("accounts_list")


def run_async(coro) -> Any:
    """Run an async coroutine from sync code."""
    return asyncio.run(coro)
