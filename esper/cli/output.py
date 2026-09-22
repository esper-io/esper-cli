"""
Central output helper — Rich-powered replacement for tabulate/typer.echo.

All existing ``render()`` call sites work without any modification.  The
improvement is entirely cosmetic/UX:

* Colour-coded status values (green=OK, yellow=warning, red=fail, …)
* Rich tables with an underlined header instead of plain text
* Syntax-highlighted JSON output
* Success messages get a leading "✓"; errors route to stderr in red
* ``prompt_options()`` displays a styled numbered menu
"""
from __future__ import annotations

import json as _json
from typing import Any

import typer
from rich import reconfigure as _rich_reconfigure
from rich.console import Console
from rich.json import JSON as RichJSON
from rich.panel import Panel
from rich.table import Table, box
from rich.text import Text

from esper.controllers.enums import OutputFormat

# ---------------------------------------------------------------------------
# Consoles
# ---------------------------------------------------------------------------
console = Console(highlight=False)
err_console = Console(stderr=True, highlight=False)

# ---------------------------------------------------------------------------
# Status → style mapping
# Covers DeviceState names, CommandState names & their human-readable values.
# ---------------------------------------------------------------------------
_STATUS_STYLES: dict[str, str] = {
    # ── Device states ─────────────────────────────────────────────────────
    "ACTIVE": "bold green",
    "ONBOARDED": "green",
    "REGISTERED": "green",
    "APPS_INSTALLED": "green",
    "DEVICE_SETTINGS_PROCESSED": "green",
    # Transitional / provisioning
    "PROVISIONING_BEGIN": "yellow",
    "GOOGLE_PLAY_CONFIGURATION": "yellow",
    "POLICY_APPLICATION_IN_PROGRESS": "yellow",
    "ONBOARDING_IN_PROGRESS": "yellow",
    "AFW_ACCOUNT_ADDED": "yellow",
    "BRANDING_PROCESSED": "yellow",
    "PERMISSION_POLICY_PROCESSED": "yellow",
    "DEVICE_POLICY_PROCESSED": "yellow",
    "SECURITY_POLICY_PROCESSED": "yellow",
    "PHONE_POLICY_PROCESSED": "yellow",
    "CUSTOM_SETTINGS_PROCESSED": "yellow",
    # Problem states
    "INACTIVE": "yellow",
    "DISABLED": "dim",
    "ONBOARDING_FAILED": "bold red",
    "WIPE_IN_PROGRESS": "bold red",
    # ── V2 command states (names) ──────────────────────────────────────────
    "SUCCESS": "bold green",
    "FAILURE": "bold red",
    "TIMEOUT": "red",
    "IN_PROGRESS": "cyan",
    "INITIATE": "cyan",
    "ACKNOWLEDGE": "cyan",
    "QUEUED": "blue",
    "SCHEDULED": "blue",
    "CANCELLED": "dim",
    # ── V2 command states (human-readable values) ─────────────────────────
    "Command Success": "bold green",
    "Command Failure": "bold red",
    "Command TimeOut": "red",
    "Command In Progress": "cyan",
    "Command Initiated": "cyan",
    "Command Acknowledged": "cyan",
    "Command Queued": "blue",
    "Command Scheduled": "blue",
    "Command Cancelled": "dim",
}

# Prefixes that indicate a successful action completed
_SUCCESS_PREFIXES = (
    "Created ", "Added ", "Edited ", "Removed ", "Deleted ",
    "Unset ", "Uploaded ", "Updated ", "Renewed ",
    "Pipeline execution",
    "Version with id",
    "Application with id",
    "Content with id",
)


def _styled_value(value: Any) -> Text:
    """Return a ``rich.Text`` with optional status styling for *value*."""
    s = str(value) if value is not None else ""
    style = _STATUS_STYLES.get(s)
    return Text(s, style=style or "")


# ---------------------------------------------------------------------------
# Public helpers
# ---------------------------------------------------------------------------

def disable_color() -> None:
    """Disable all ANSI colour (called when ``--no-color`` is passed)."""
    _rich_reconfigure(no_color=True)


def render(data, format=None, **kwargs) -> None:
    """
    Print *data* in the requested format.

    Compatible with every calling convention used in the CLI modules::

        render("some message")
        render(list_of_dicts, format=OutputFormat.TABULATED.value,
               headers="keys", tablefmt="plain")
        render(dict_or_list, format=OutputFormat.JSON.value)
    """
    # ── Plain string ──────────────────────────────────────────────────────
    if format is None or isinstance(data, str):
        s = str(data).rstrip("\n")
        if not s:
            return
        if s.upper().startswith("ERROR"):
            err_console.print(f"[bold red]{s}[/bold red]")
        elif any(s.startswith(p) for p in _SUCCESS_PREFIXES):
            console.print(f"[bold green]✓[/bold green] {s}")
        else:
            console.print(s)

    # ── Tabulated ─────────────────────────────────────────────────────────
    elif format == OutputFormat.TABULATED.value:
        if not data:
            return
        if isinstance(data, list) and data and isinstance(data[0], dict):
            headers = list(data[0].keys())
            tbl = Table(
                box=box.SIMPLE_HEAD,
                show_header=True,
                header_style="bold cyan",
                show_edge=False,
                pad_edge=False,
            )
            for h in headers:
                tbl.add_column(str(h), no_wrap=False, overflow="fold")
            for row in data:
                tbl.add_row(*[_styled_value(row.get(h)) for h in headers])
            console.print(tbl)
        else:
            # Fallback (shouldn't normally happen)
            from tabulate import tabulate as _tab
            console.print(_tab(data, **kwargs))

    # ── JSON ──────────────────────────────────────────────────────────────
    elif format == OutputFormat.JSON.value:
        console.print(RichJSON(_json.dumps(data, indent=2, default=str)))

    else:
        console.print(str(data))


def install_rich_errors() -> None:
    """
    Patch Typer's ``rich_format_error`` so every usage-error automatically
    shows the relevant help page immediately below the error, rather than
    telling the user to go run ``--help`` themselves.

    Typer already intercepts ``ClickException`` and routes it through
    ``rich_utils.rich_format_error`` when Rich is available, so we hook
    there rather than ``click.exceptions.UsageError.show``.

    Call this once, early — ``app.py`` calls it before ``main()`` runs.
    """
    try:
        from typer import rich_utils as _ru
    except ImportError:
        return  # Typer not installed or very old version — skip silently

    def _enhanced_format_error(exc) -> None:  # noqa: ANN001
        # Typer passes the exception as the first positional arg (not `self`)
        ctx = getattr(exc, "ctx", None)
        msg = exc.format_message()

        # ── Build a short, human-readable error description ──────────────────
        if "Missing argument" in msg and "'" in msg:
            arg = msg.split("'")[1].lower().replace("_", " ")
            description = f"[bold]{arg}[/bold] is required."
        elif "Missing option" in msg and "'" in msg:
            opt = msg.split("'")[1]
            description = f"[bold]{opt}[/bold] is required."
        elif "Got unexpected extra" in msg:
            description = "Unexpected extra argument — check argument order."
        else:
            description = msg  # includes "Did you mean X?" from Click 8

        # ── Error panel (compact — shown first) ──────────────────────────────
        rich_console = _ru._get_rich_console(stderr=True)
        rich_console.print(
            Panel(
                f"[bold red]✗[/bold red]  {description}",
                title="[bold red]Error[/bold red]",
                border_style="red",
                expand=False,
                padding=(0, 1),
            )
        )

        # ── Full help for the relevant command (shown immediately after) ──────
        if ctx is not None:
            try:
                _ru.rich_format_help(
                    obj=ctx.command,
                    ctx=ctx,
                    markup_mode="rich",
                )
            except Exception:
                # Fallback to plain text if rich_format_help fails for any reason
                rich_console.print(ctx.get_help())

    _ru.rich_format_error = _enhanced_format_error  # type: ignore[assignment]


def prompt_options(prompt_text: str, options: list) -> str:
    """
    Styled numbered-menu prompt.

    *options* is a list of dicts with keys ``"prompt"`` (display text) and
    ``"return"`` (the value returned on selection).
    """
    console.print(f"\n[bold]{prompt_text}[/bold]")
    for i, opt in enumerate(options, 1):
        console.print(f"  [cyan]\\[{i}][/cyan] {opt['prompt']}")
    while True:
        raw = typer.prompt("Enter choice")
        try:
            choice = int(raw)
            if 1 <= choice <= len(options):
                return options[choice - 1]["return"]
        except ValueError:
            pass
        err_console.print(
            f"[red]Invalid choice.[/red]  Please enter a number between 1 and {len(options)}."
        )
