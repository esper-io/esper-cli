"""
Device command controller — replaces esper/controllers/device/command.py.
`espercli device-command show/install/uninstall/ping/lock/reboot/wipe/clear-app-data`
"""
from typing import Optional

import typer
from esperclient import CommandRequest
from esperclient.rest import ApiException

from esper.cli.completions import device_name_complete
from esper.cli.output import render
from esper.cli.state import state, validate_creds, parse_error_message
from esper.controllers.enums import DeviceCommandEnum, OutputFormat
from esper.ext.api_client import APIClient
from esper.ext.db_wrapper import DBWrapper

app = typer.Typer(help="Fire commands at a device")


# ---------------------------------------------------------------------------
# Shared helpers
# ---------------------------------------------------------------------------

def _command_basic_response(command, fmt=OutputFormat.TABULATED):
    valid_keys = ["id", "command", "state"]
    if fmt == OutputFormat.TABULATED:
        title, details = "TITLE", "DETAILS"
        return [{title: k, details: v} for k, v in command.to_dict().items() if k in valid_keys]
    return {k: v for k, v in command.to_dict().items() if k in valid_keys}


def _resolve_device_id(db, enterprise_id, device_name: Optional[str], tag: str) -> str:
    """Return a device_id for the given name or fall back to the active device."""
    device_client = APIClient(db.get_configure()).get_device_api_client()
    if device_name:
        try:
            search_response = device_client.get_all_devices(
                enterprise_id, limit=1, offset=0, name=device_name
            )
            if not search_response.results:
                render(f"Device does not exist with name {device_name}")
                raise typer.Exit(1)
            return search_response.results[0].id
        except ApiException as e:
            state.log.error(f"[{tag}] Failed to list devices: {e}")
            render(f"ERROR: {parse_error_message(e)}")
            raise typer.Exit(1)
    else:
        device = db.get_device()
        if not device or not device.get("id"):
            render("There is no active device.")
            raise typer.Exit(1)
        return device.get("id")


# ---------------------------------------------------------------------------
# Commands
# ---------------------------------------------------------------------------

@app.command("show")
def command_show(
    command_id: str = typer.Argument(..., help="Device command ID"),
    device: Optional[str] = typer.Option(None, "--device", "-d", help="Device name", shell_complete=device_name_complete),
    json_output: bool = typer.Option(False, "--json", "-j", help="Render result in JSON format"),
):
    """Show command details."""
    validate_creds()
    db = DBWrapper(state.creds)
    command_client = APIClient(db.get_configure()).get_command_api_client()
    enterprise_id = db.get_enterprise_id()

    device_id = _resolve_device_id(db, enterprise_id, device, "device-command-show")

    try:
        response = command_client.get_command(command_id, device_id, enterprise_id)
    except ApiException as e:
        state.log.error(f"[device-command-show] Failed to show command: {e}")
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)

    if not json_output:
        render(_command_basic_response(response), format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
    else:
        render(_command_basic_response(response, OutputFormat.JSON), format=OutputFormat.JSON.value)


@app.command("install")
def command_install(
    device: Optional[str] = typer.Option(None, "--device", "-d", help="Device name", shell_complete=device_name_complete),
    version: Optional[str] = typer.Option(None, "--version", "-V", help="Application version ID"),
    json_output: bool = typer.Option(False, "--json", "-j", help="Render result in JSON format"),
):
    """Install an application version on a device."""
    validate_creds()
    db = DBWrapper(state.creds)
    command_client = APIClient(db.get_configure()).get_command_api_client()
    enterprise_id = db.get_enterprise_id()
    device_id = _resolve_device_id(db, enterprise_id, device, "device-command-install")

    command_request = CommandRequest(
        command_args={"app_version": version},
        command=DeviceCommandEnum.INSTALL.name,
    )
    try:
        response = command_client.run_command(enterprise_id, device_id, command_request)
    except ApiException as e:
        state.log.error(f"[device-command-install] Failed to fire install command: {e}")
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)

    if not json_output:
        render(_command_basic_response(response), format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
    else:
        render(_command_basic_response(response, OutputFormat.JSON), format=OutputFormat.JSON.value)


@app.command("uninstall")
def command_uninstall(
    device: Optional[str] = typer.Option(None, "--device", "-d", help="Device name", shell_complete=device_name_complete),
    version: Optional[str] = typer.Option(None, "--version", "-V", help="Application version ID"),
    json_output: bool = typer.Option(False, "--json", "-j", help="Render result in JSON format"),
):
    """Uninstall an application version from a device."""
    validate_creds()
    db = DBWrapper(state.creds)
    command_client = APIClient(db.get_configure()).get_command_api_client()
    enterprise_id = db.get_enterprise_id()
    device_id = _resolve_device_id(db, enterprise_id, device, "device-command-uninstall")

    command_request = CommandRequest(
        command_args={"app_version": version},
        command=DeviceCommandEnum.UNINSTALL.name,
    )
    try:
        response = command_client.run_command(enterprise_id, device_id, command_request)
    except ApiException as e:
        state.log.error(f"[device-command-uninstall] Failed to fire uninstall command: {e}")
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)

    if not json_output:
        render(_command_basic_response(response), format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
    else:
        render(_command_basic_response(response, OutputFormat.JSON), format=OutputFormat.JSON.value)


@app.command("ping")
def command_ping(
    device: Optional[str] = typer.Option(None, "--device", "-d", help="Device name", shell_complete=device_name_complete),
    json_output: bool = typer.Option(False, "--json", "-j", help="Render result in JSON format"),
):
    """Ping a device (update heartbeat)."""
    validate_creds()
    db = DBWrapper(state.creds)
    command_client = APIClient(db.get_configure()).get_command_api_client()
    enterprise_id = db.get_enterprise_id()
    device_id = _resolve_device_id(db, enterprise_id, device, "device-command-ping")

    command_request = CommandRequest(command=DeviceCommandEnum.UPDATE_HEARTBEAT.name)
    try:
        response = command_client.run_command(enterprise_id, device_id, command_request)
    except ApiException as e:
        state.log.error(f"[device-command-ping] Failed to fire ping command: {e}")
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)

    if not json_output:
        render(_command_basic_response(response), format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
    else:
        render(_command_basic_response(response, OutputFormat.JSON), format=OutputFormat.JSON.value)


@app.command("lock")
def command_lock(
    device: Optional[str] = typer.Option(None, "--device", "-d", help="Device name", shell_complete=device_name_complete),
    json_output: bool = typer.Option(False, "--json", "-j", help="Render result in JSON format"),
):
    """Lock a device."""
    validate_creds()
    db = DBWrapper(state.creds)
    command_client = APIClient(db.get_configure()).get_command_api_client()
    enterprise_id = db.get_enterprise_id()
    device_id = _resolve_device_id(db, enterprise_id, device, "device-command-lock")

    command_request = CommandRequest(command=DeviceCommandEnum.LOCK.name)
    try:
        response = command_client.run_command(enterprise_id, device_id, command_request)
    except ApiException as e:
        state.log.error(f"[device-command-lock] Failed to fire lock command: {e}")
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)

    if not json_output:
        render(_command_basic_response(response), format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
    else:
        render(_command_basic_response(response, OutputFormat.JSON), format=OutputFormat.JSON.value)


@app.command("reboot")
def command_reboot(
    device: Optional[str] = typer.Option(None, "--device", "-d", help="Device name", shell_complete=device_name_complete),
    json_output: bool = typer.Option(False, "--json", "-j", help="Render result in JSON format"),
):
    """Reboot a device."""
    validate_creds()
    db = DBWrapper(state.creds)
    command_client = APIClient(db.get_configure()).get_command_api_client()
    enterprise_id = db.get_enterprise_id()
    device_id = _resolve_device_id(db, enterprise_id, device, "device-command-reboot")

    command_request = CommandRequest(command=DeviceCommandEnum.REBOOT.name)
    try:
        response = command_client.run_command(enterprise_id, device_id, command_request)
    except ApiException as e:
        state.log.error(f"[device-command-reboot] Failed to fire reboot command: {e}")
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)

    if not json_output:
        render(_command_basic_response(response), format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
    else:
        render(_command_basic_response(response, OutputFormat.JSON), format=OutputFormat.JSON.value)


@app.command("wipe")
def command_wipe(
    device: Optional[str] = typer.Option(None, "--device", "-d", help="Device name", shell_complete=device_name_complete),
    external_storage: bool = typer.Option(
        False, "--exstorage", "-e", help="Wipe external storage"
    ),
    frp: bool = typer.Option(False, "--frp", "-f", help="Factory reset protection"),
    yes: bool = typer.Option(False, "--yes", "-y", help="Skip confirmation prompt"),
    json_output: bool = typer.Option(False, "--json", "-j", help="Render result in JSON format"),
):
    """Wipe a device (⚠ destructive — requires confirmation)."""
    target = device or "(active device)"
    if not yes:
        typer.confirm(
            f"⚠  This will WIPE {target}.  All data will be erased.  Continue?",
            abort=True,
        )
    validate_creds()
    db = DBWrapper(state.creds)
    command_client = APIClient(db.get_configure()).get_command_api_client()
    enterprise_id = db.get_enterprise_id()
    device_id = _resolve_device_id(db, enterprise_id, device, "device-command-wipe")

    command_request = CommandRequest(
        command_args={"wipe_external_storage": external_storage, "wipe_FRP": frp},
        command=DeviceCommandEnum.WIPE.name,
    )
    try:
        response = command_client.run_command(enterprise_id, device_id, command_request)
    except ApiException as e:
        state.log.error(f"[device-command-wipe] Failed to fire wipe command: {e}")
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)

    if not json_output:
        render(_command_basic_response(response), format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
    else:
        render(_command_basic_response(response, OutputFormat.JSON), format=OutputFormat.JSON.value)


@app.command("clear-app-data")
def command_clear_app_data(
    device: Optional[str] = typer.Option(None, "--device", "-d", help="Device name", shell_complete=device_name_complete),
    package_name: Optional[str] = typer.Option(
        None, "--package-name", "-P", help="Application package name"
    ),
    json_output: bool = typer.Option(False, "--json", "-j", help="Render result in JSON format"),
):
    """Clear application data on a device."""
    validate_creds()
    db = DBWrapper(state.creds)
    command_client = APIClient(db.get_configure()).get_command_api_client()
    enterprise_id = db.get_enterprise_id()
    device_id = _resolve_device_id(db, enterprise_id, device, "device-command-clear-app-data")

    if not package_name:
        render("Package name is empty")
        raise typer.Exit(1)

    command_request = CommandRequest(
        command_args={"package_name": package_name},
        command=DeviceCommandEnum.CLEAR_APP_DATA.name,
    )
    try:
        response = command_client.run_command(enterprise_id, device_id, command_request)
    except ApiException as e:
        state.log.error(f"[device-command-clear-app-data] Failed to fire CLEAR_APP_DATA command: {e}")
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)

    if not json_output:
        render(_command_basic_response(response), format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
    else:
        render(_command_basic_response(response, OutputFormat.JSON), format=OutputFormat.JSON.value)
