# Policy commands — list, show, and delete enterprise policies.
from typing import Optional

import typer

from esper.cli.output import render
from esper.cli.state import state, validate_creds
from esper.controllers.enums import OutputFormat
from esper.ext.api_rest import api_delete, api_get
from esper.ext.db_wrapper import DBWrapper

app = typer.Typer(help="Policy management commands")


@app.command("list")
def policy_list(
    as_json: bool = typer.Option(False, "--json", "-j", help="Output as JSON"),
) -> None:
    """List all enterprise policies."""
    validate_creds()
    db = DBWrapper(state.creds)
    cfg = db.get_configure()
    environment = cfg.get("environment")
    api_key = cfg.get("api_key")
    eid = db.get_enterprise_id()

    try:
        data = api_get(environment, api_key, f"/enterprise/{eid}/policy/")
    except Exception as e:
        render(f"ERROR: {e}")
        raise typer.Exit(1)

    items = data if isinstance(data, list) else data.get("results", data.get("content", []))
    if not items:
        render("No policies found.")
        raise typer.Exit(0)

    if as_json:
        render(items, format=OutputFormat.JSON.value)
        return

    rows = [{"ID": p.get("id"), "POLICY NAME": p.get("policy_name", "—")} for p in items]
    render(rows, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")


@app.command("show")
def policy_show(
    policy_id: str = typer.Argument(..., help="Policy ID"),
    as_json: bool = typer.Option(False, "--json", "-j", help="Output as JSON"),
) -> None:
    """Show details for a single policy."""
    validate_creds()
    db = DBWrapper(state.creds)
    cfg = db.get_configure()
    environment = cfg.get("environment")
    api_key = cfg.get("api_key")
    eid = db.get_enterprise_id()

    try:
        data = api_get(environment, api_key, f"/enterprise/{eid}/policy/{policy_id}/")
    except Exception as e:
        render(f"ERROR: {e}")
        raise typer.Exit(1)

    if as_json:
        render(data, format=OutputFormat.JSON.value)
        return

    rows = [{"FIELD": k, "VALUE": str(v)} for k, v in data.items()]
    render(rows, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")


@app.command("delete")
def policy_delete(
    policy_id: str = typer.Argument(..., help="Policy ID"),
    yes: bool = typer.Option(False, "--yes", "-y", help="Skip confirmation prompt"),
) -> None:
    """Delete a policy."""
    validate_creds()

    if not yes:
        typer.confirm(f"Delete policy {policy_id}?", abort=True)

    db = DBWrapper(state.creds)
    cfg = db.get_configure()
    environment = cfg.get("environment")
    api_key = cfg.get("api_key")
    eid = db.get_enterprise_id()

    try:
        status = api_delete(environment, api_key, f"/enterprise/{eid}/policy/{policy_id}/")
    except Exception as e:
        render(f"ERROR: {e}")
        raise typer.Exit(1)

    if status in (200, 204):
        render(f"Deleted policy {policy_id}")
    else:
        render(f"ERROR: Unexpected status code {status}")
        raise typer.Exit(1)
