"""
Main Typer application — entry point for ``espercli``.

Wires all sub-apps, builds the nested pipeline/stage/operation/execute
hierarchy, and registers the global --verbose / --no-color flags.
"""
from __future__ import annotations

import typer
from rich.console import Console
from rich.panel import Panel
from rich.table import Table

# ── Sub-app imports ──────────────────────────────────────────────────────────
from esper.cli.configure import app as configure_app
from esper.cli.device import app as device_app
from esper.cli.device_command import app as device_command_app
from esper.cli.group_command import app as group_command_app
from esper.cli.installs import app as installs_app
from esper.cli.status import app as status_app
from esper.cli.application import app as application_app
from esper.cli.version import app as version_app
from esper.cli.enterprise import app as enterprise_app
from esper.cli.group import app as group_app
from esper.cli.pipeline import app as pipeline_app
from esper.cli.stage import app as stage_app
from esper.cli.operation import app as operation_app
from esper.cli.execute import app as execute_app
from esper.cli.token import app as token_app
from esper.cli.content import app as content_app
from esper.cli.commandsv2 import app as commandsv2_app
from esper.cli.secureadb import app as secureadb_app
from esper.cli.telemetry import app as telemetry_app
from esper.cli.fleet import app as fleet_app
from esper.cli.heartbeat import app as heartbeat_app
from esper.cli.location import app as location_app
from esper.cli.policy import app as policy_app
from esper.cli.user import app as user_app
from esper.cli.output import disable_color, install_rich_errors
from esper.cli.state import state

# ── Install Rich error rendering ASAP ───────────────────────────────────────
# Must run before Click/Typer parse any arguments so every UsageError (typos,
# missing args, unknown options) is rendered inside a Rich panel.
install_rich_errors()

# ── Wire the pipeline hierarchy ──────────────────────────────────────────────
#   espercli pipeline stage operation <cmd>
stage_app.add_typer(operation_app, name="operation")
#   espercli pipeline stage <cmd>  /  espercli pipeline execute <cmd>
pipeline_app.add_typer(stage_app, name="stage")
pipeline_app.add_typer(execute_app, name="execute")

# ── Root application ─────────────────────────────────────────────────────────
main = typer.Typer(
    help="Esper CLI — command line tool for the Esper APIs",
    no_args_is_help=True,
    rich_markup_mode="rich",
)


@main.callback()
def _global_options(
    verbose: bool = typer.Option(
        False, "--verbose", "-v",
        help="Enable debug logging.",
        is_eager=True,
        expose_value=True,
    ),
    no_color: bool = typer.Option(
        False, "--no-color",
        help="Disable coloured output (useful for piping).",
        is_eager=True,
        expose_value=True,
    ),
) -> None:
    """Esper CLI — command line tool for the Esper APIs."""
    if verbose:
        state.set_debug(True)
    if no_color:
        disable_color()


# ── Built-in utility commands ────────────────────────────────────────────────

@main.command("context")
def show_context() -> None:
    """
    Show the current active context at a glance.

    Displays the configured environment, enterprise ID, and whatever
    active device / application / group is set in the local store.
    """
    from esper.ext.db_wrapper import DBWrapper

    _console = Console()
    db = DBWrapper(state.creds)

    config      = db.get_configure()    or {}
    device      = db.get_device()       or {}
    application = db.get_application()  or {}
    group       = db.get_group()        or {}

    tbl = Table(show_header=False, box=None, padding=(0, 3))
    tbl.add_column("key",   style="bold cyan", no_wrap=True)
    tbl.add_column("value", no_wrap=False)

    def _val(v: str | None) -> str:
        return v if v else "[dim]none[/dim]"

    env        = config.get("environment")
    raw_eid = config.get("enterprise_id") or ""
    # enterprise_id may be stored as a full URL like
    # "http://.../enterprise/UUID/"  — extract the trailing UUID if so
    parts = [p for p in raw_eid.split("/") if p]
    eid = parts[-1] if parts else None
    dev_label  = device.get("name") or device.get("id")
    app_label  = application.get("id")
    grp_label  = group.get("name") or group.get("id")

    if not env:
        tbl.add_row(
            "Status",
            "[bold red]Not configured[/bold red] — run [cyan]espercli configure[/cyan] first",
        )
    else:
        tbl.add_row("Environment",   _val(env))
        tbl.add_row("Enterprise ID", _val(eid))
        tbl.add_row("Active Device", _val(dev_label))
        tbl.add_row("Active App",    _val(app_label))
        tbl.add_row("Active Group",  _val(grp_label))

    _console.print(
        Panel(tbl, title="[bold]Esper CLI Context[/bold]", border_style="bold blue", expand=False)
    )


@main.command("about")
def show_version() -> None:
    """Show the espercli version and build info."""
    try:
        from importlib.metadata import version as _pkg_version
        ver = _pkg_version("espercli")
    except Exception:
        ver = "0.0.16"
    Console().print(f"[bold]espercli[/bold] version [cyan]{ver}[/cyan]")


@main.command("completion")
def install_completion(
    shell: str = typer.Argument(
        ...,
        help="Shell to install completion for: bash, zsh, fish, powershell",
    ),
) -> None:
    """
    Install tab-completion for your shell.

    Run once, then restart your terminal (or re-source your shell config):

        espercli completion zsh
        espercli completion bash
        espercli completion fish
    """
    import os
    import sys
    from typer._completion_shared import install as _install

    supported = {"bash", "zsh", "fish", "powershell", "pwsh"}
    if shell.lower() not in supported:
        Console(stderr=True).print(
            Panel(
                f"[bold red]✗[/bold red]  [bold]{shell}[/bold] is not supported.\n\n"
                f"[dim]Supported shells: {', '.join(sorted(supported))}[/dim]",
                title="[bold red]Error[/bold red]",
                border_style="red",
                expand=False,
                padding=(0, 1),
            ),
            file=sys.stderr,
        )
        raise typer.Exit(1)

    # Bypass shellingham — pass the shell name directly.
    detected_shell, path = _install(shell=shell.lower())
    Console().print(
        f"[bold green]✓[/bold green]  [bold]{detected_shell}[/bold] completion "
        f"installed in [cyan]{path}[/cyan]\n\n"
        "[dim]Restart your terminal or run:[/dim]  "
        f"[cyan]source {'~/.zshrc' if shell == 'zsh' else '~/.bashrc'}[/cyan]"
    )


# ── Register all sub-apps ────────────────────────────────────────────────────
main.add_typer(configure_app,      name="configure")
main.add_typer(device_app,         name="device")
main.add_typer(device_command_app, name="device-command")
main.add_typer(group_command_app,  name="group-command")
main.add_typer(installs_app,       name="installs")
main.add_typer(status_app,         name="status")
main.add_typer(application_app,    name="app")
main.add_typer(version_app,        name="version")
main.add_typer(enterprise_app,     name="enterprise")
main.add_typer(group_app,          name="group")
main.add_typer(pipeline_app,       name="pipeline")
main.add_typer(token_app,          name="token")
main.add_typer(content_app,        name="content")
main.add_typer(commandsv2_app,     name="commandsV2")
main.add_typer(secureadb_app,      name="secureadb")
main.add_typer(telemetry_app,      name="telemetry")
main.add_typer(fleet_app,          name="fleet")
main.add_typer(heartbeat_app,      name="heartbeat")
main.add_typer(location_app,       name="location")
main.add_typer(policy_app,         name="policy")
main.add_typer(user_app,           name="user")


def main_entry() -> None:
    """Console-script entry point registered in setup.py."""
    main()


if __name__ == "__main__":
    main_entry()
