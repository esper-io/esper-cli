"""
Group command controller — replaces esper/controllers/device/group_command.py.
`espercli group-command show/install/ping/lock/reboot`
"""
from ast import literal_eval
from typing import Optional

import typer
from esperclient import GroupCommandRequest
from esperclient.rest import ApiException

from esper.cli.completions import group_name_complete
from esper.cli.output import render
from esper.cli.state import state, validate_creds, parse_error_message
from esper.controllers.enums import DeviceCommandEnum, OutputFormat
from esper.ext.api_client import APIClient
from esper.ext.db_wrapper import DBWrapper

app = typer.Typer(help="Fire commands at a group of devices")


# ---------------------------------------------------------------------------
# Shared helpers
# ---------------------------------------------------------------------------

def _command_basic_response(command, fmt=OutputFormat.TABULATED):
    valid_keys = ["id", "command", "state"]

    success = failed = in_progress = inactive = None
    success_list: list = []
    failed_list: list = []
    inactive_list: list = []
    in_progress_list: list = []

    if command.details:
        details = literal_eval(command.details)
        if details.get("success"):
            success_list = [d.get("name") for d in details["success"]]
            success = "\n".join(success_list) if success_list else None
        if details.get("failed"):
            failed_list = [d.get("name") for d in details["failed"]]
            failed = "\n".join(failed_list) if failed_list else None
        if details.get("inactive"):
            inactive_list = [d.get("name") for d in details["inactive"]]
            inactive = "\n".join(inactive_list) if inactive_list else None
        for key in ("acknowledge", "initiate", "in_progress", "timeout"):
            if details.get(key):
                in_progress_list.extend([d.get("name") for d in details[key]])
        in_progress = "\n".join(in_progress_list) if in_progress_list else None

    if fmt == OutputFormat.TABULATED:
        title, det = "TITLE", "DETAILS"
        renderable = [{title: k, det: v} for k, v in command.to_dict().items() if k in valid_keys]
        renderable.extend([
            {title: "success", det: success},
            {title: "failed", det: failed},
            {title: "in_progress", det: in_progress},
            {title: "inactive", det: inactive},
        ])
    else:
        renderable = {k: v for k, v in command.to_dict().items() if k in valid_keys}
        renderable.update({"success": success_list, "failed": failed_list,
                           "in_progress": in_progress_list, "inactive": inactive_list})
    return renderable


def _resolve_group_id(db, enterprise_id, group_name: Optional[str], tag: str) -> str:
    """Return group_id for the given name or fall back to the active group."""
    group_client = APIClient(db.get_configure()).get_group_api_client()
    if group_name:
        try:
            search_response = group_client.get_all_groups(
                enterprise_id, limit=1, offset=0, name=group_name
            )
            if not search_response.results:
                render(f"Group does not exist with name {group_name}")
                raise typer.Exit(1)
            return search_response.results[0].id
        except ApiException as e:
            state.log.error(f"[{tag}] Failed to list groups: {e}")
            render(f"ERROR: {parse_error_message(e)}")
            raise typer.Exit(1)
    else:
        group = db.get_group()
        if group is None or group.get("name") is None:
            render("There is no active group.")
            raise typer.Exit(1)
        return group.get("id")


# ---------------------------------------------------------------------------
# Commands
# ---------------------------------------------------------------------------

@app.command("show")
def group_command_show(
    command_id: str = typer.Argument(..., help="Group command ID"),
    group: Optional[str] = typer.Option(None, "--group", "-g", help="Group name", shell_complete=group_name_complete),
    json_output: bool = typer.Option(False, "--json", "-j", help="Render result in JSON format"),
):
    """Show group command details."""
    validate_creds()
    db = DBWrapper(state.creds)
    command_client = APIClient(db.get_configure()).get_group_command_api_client()
    enterprise_id = db.get_enterprise_id()
    group_id = _resolve_group_id(db, enterprise_id, group, "group-command-show")

    try:
        response = command_client.get_group_command(command_id, group_id, enterprise_id)
    except ApiException as e:
        state.log.error(f"[group-command-show] Failed to show command: {e}")
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)

    if not json_output:
        render(_command_basic_response(response), format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
    else:
        render(_command_basic_response(response, OutputFormat.JSON), format=OutputFormat.JSON.value)


@app.command("install")
def group_command_install(
    group: Optional[str] = typer.Option(None, "--group", "-g", help="Group name", shell_complete=group_name_complete),
    version: Optional[str] = typer.Option(None, "--version", "-V", help="Application version ID"),
    json_output: bool = typer.Option(False, "--json", "-j", help="Render result in JSON format"),
):
    """Install an application version on a group of devices."""
    validate_creds()
    db = DBWrapper(state.creds)
    command_client = APIClient(db.get_configure()).get_group_command_api_client()
    enterprise_id = db.get_enterprise_id()
    group_id = _resolve_group_id(db, enterprise_id, group, "group-command-install")

    command_request = GroupCommandRequest(
        command_args={"app_version": version},
        command=DeviceCommandEnum.INSTALL.name,
    )
    try:
        response = command_client.run_group_command(enterprise_id, group_id, command_request)
    except ApiException as e:
        state.log.error(f"[group-command-install] Failed to fire install command: {e}")
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)

    if not json_output:
        render(_command_basic_response(response), format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
    else:
        render(_command_basic_response(response, OutputFormat.JSON), format=OutputFormat.JSON.value)


@app.command("ping")
def group_command_ping(
    group: Optional[str] = typer.Option(None, "--group", "-g", help="Group name", shell_complete=group_name_complete),
    json_output: bool = typer.Option(False, "--json", "-j", help="Render result in JSON format"),
):
    """Ping a group of devices (update heartbeat)."""
    validate_creds()
    db = DBWrapper(state.creds)
    command_client = APIClient(db.get_configure()).get_group_command_api_client()
    enterprise_id = db.get_enterprise_id()
    group_id = _resolve_group_id(db, enterprise_id, group, "group-command-ping")

    command_request = GroupCommandRequest(command=DeviceCommandEnum.UPDATE_HEARTBEAT.name)
    try:
        response = command_client.run_group_command(enterprise_id, group_id, command_request)
    except ApiException as e:
        state.log.error(f"[group-command-ping] Failed to fire ping command: {e}")
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)

    if not json_output:
        render(_command_basic_response(response), format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
    else:
        render(_command_basic_response(response, OutputFormat.JSON), format=OutputFormat.JSON.value)


@app.command("lock")
def group_command_lock(
    group: Optional[str] = typer.Option(None, "--group", "-g", help="Group name", shell_complete=group_name_complete),
    json_output: bool = typer.Option(False, "--json", "-j", help="Render result in JSON format"),
):
    """Lock a group of devices."""
    validate_creds()
    db = DBWrapper(state.creds)
    command_client = APIClient(db.get_configure()).get_group_command_api_client()
    enterprise_id = db.get_enterprise_id()
    group_id = _resolve_group_id(db, enterprise_id, group, "group-command-lock")

    command_request = GroupCommandRequest(command=DeviceCommandEnum.LOCK.name)
    try:
        response = command_client.run_group_command(enterprise_id, group_id, command_request)
    except ApiException as e:
        state.log.error(f"[group-command-lock] Failed to fire lock command: {e}")
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)

    if not json_output:
        render(_command_basic_response(response), format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
    else:
        render(_command_basic_response(response, OutputFormat.JSON), format=OutputFormat.JSON.value)


@app.command("reboot")
def group_command_reboot(
    group: Optional[str] = typer.Option(None, "--group", "-g", help="Group name", shell_complete=group_name_complete),
    json_output: bool = typer.Option(False, "--json", "-j", help="Render result in JSON format"),
):
    """Reboot a group of devices."""
    validate_creds()
    db = DBWrapper(state.creds)
    command_client = APIClient(db.get_configure()).get_group_command_api_client()
    enterprise_id = db.get_enterprise_id()
    group_id = _resolve_group_id(db, enterprise_id, group, "group-command-reboot")

    command_request = GroupCommandRequest(command=DeviceCommandEnum.REBOOT.name)
    try:
        response = command_client.run_group_command(enterprise_id, group_id, command_request)
    except ApiException as e:
        state.log.error(f"[group-command-reboot] Failed to fire reboot command: {e}")
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)

    if not json_output:
        render(_command_basic_response(response), format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
    else:
        render(_command_basic_response(response, OutputFormat.JSON), format=OutputFormat.JSON.value)
