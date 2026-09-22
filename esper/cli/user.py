# User management commands — list, invite, and remove enterprise users.
from typing import Optional

import typer

from esper.cli.output import render
from esper.cli.state import state, validate_creds
from esper.controllers.enums import OutputFormat
from esper.ext.api_rest import api_delete, api_get, api_post
from esper.ext.db_wrapper import DBWrapper

app = typer.Typer(help="User management commands")

_DEVICE_USER_PREFIX = "device-"
_DEVICE_USER_DOMAIN = "@esper.io"


def _is_system_user(email: str) -> bool:
    return email.startswith(_DEVICE_USER_PREFIX) and email.endswith(_DEVICE_USER_DOMAIN)


@app.command("list")
def user_list(
    as_json: bool = typer.Option(False, "--json", "-j", help="Output as JSON"),
) -> None:
    """List all enterprise users (excluding device system accounts)."""
    validate_creds()
    db = DBWrapper(state.creds)
    cfg = db.get_configure()
    environment = cfg.get("environment")
    api_key = cfg.get("api_key")
    eid = db.get_enterprise_id()

    try:
        data = api_get(environment, api_key, "/authn2/v1/users/")
    except Exception as e:
        render(f"ERROR: {e}")
        raise typer.Exit(1)

    content = data.get("content", data)
    users = content.get("results", []) if isinstance(content, dict) else []
    users = [u for u in users if not _is_system_user(u.get("email", ""))]

    if not users:
        render("No users found.")
        raise typer.Exit(0)

    if as_json:
        render(users, format=OutputFormat.JSON.value)
        return

    rows = [
        {
            "USER ID": u.get("user_id", "")[:8],
            "EMAIL": u.get("email", "—"),
            "NAME": u.get("name", "—"),
            "CREATED AT": u.get("created_at", "—"),
        }
        for u in users
    ]
    render(rows, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")


@app.command("invite")
def user_invite(
    email: str = typer.Option(..., "--email", "-e", help="Email address to invite"),
    role: str = typer.Option("viewer", "--role", "-r", help="Role to assign (default: viewer)"),
    as_json: bool = typer.Option(False, "--json", "-j", help="Output as JSON"),
) -> None:
    """Invite a user to the enterprise."""
    validate_creds()
    db = DBWrapper(state.creds)
    cfg = db.get_configure()
    environment = cfg.get("environment")
    api_key = cfg.get("api_key")
    eid = db.get_enterprise_id()

    try:
        result = api_post(environment, api_key, f"/authn2/v0/tenant/{eid}/invite", {"email": email, "role": role})
    except Exception as e:
        render(f"ERROR: {e}")
        raise typer.Exit(1)

    if as_json:
        render(result, format=OutputFormat.JSON.value)
        return

    render(f"Invited {email} with role '{role}'")


@app.command("remove")
def user_remove(
    user_id: str = typer.Argument(..., help="User ID to remove"),
    yes: bool = typer.Option(False, "--yes", "-y", help="Skip confirmation prompt"),
) -> None:
    """Remove a user from the enterprise."""
    validate_creds()

    if not yes:
        typer.confirm(f"Remove user {user_id}?", abort=True)

    db = DBWrapper(state.creds)
    cfg = db.get_configure()
    environment = cfg.get("environment")
    api_key = cfg.get("api_key")
    eid = db.get_enterprise_id()

    try:
        status = api_delete(environment, api_key, f"/authn2/v0/tenant/{eid}/user/{user_id}/")
    except Exception as e:
        render(f"ERROR: {e}")
        raise typer.Exit(1)

    if status in (200, 204):
        render(f"Removed user {user_id}")
    else:
        render(f"ERROR: Unexpected status code {status}")
        raise typer.Exit(1)
