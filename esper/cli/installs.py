"""
Application installs commands — replaces esper/controllers/device/install.py.
`espercli installs list`
"""
from typing import Optional

import typer
from esperclient.rest import ApiException

from esper.cli.output import render
from esper.cli.state import state, validate_creds, parse_error_message
from esper.controllers.enums import OutputFormat
from esper.ext.api_client import APIClient
from esper.ext.db_wrapper import DBWrapper

app = typer.Typer(help="Application install commands")


@app.command("list")
def installs_list(
    device: Optional[str] = typer.Option(None, "--device", "-d", help="Device name"),
    appname: Optional[str] = typer.Option(None, "--appname", "-an", help="Application name"),
    package: Optional[str] = typer.Option(None, "--package", "-p", help="Package name"),
    state_filter: Optional[str] = typer.Option(None, "--state", "-s", help="Install state"),
    limit: int = typer.Option(20, "--limit", "-l", help="Results per page"),
    offset: int = typer.Option(0, "--offset", "-i", help="Initial index"),
    json_output: bool = typer.Option(False, "--json", "-j", help="Render result in JSON format"),
):
    """List application installs on a device."""
    validate_creds()
    db = DBWrapper(state.creds)
    device_client = APIClient(db.get_configure()).get_device_api_client()
    enterprise_id = db.get_enterprise_id()

    if device:
        try:
            search_response = device_client.get_all_devices(
                enterprise_id, limit=1, offset=0, name=device
            )
            if not search_response.results:
                render(f"Device does not exist with name {device}")
                raise typer.Exit(1)
            device_id = search_response.results[0].id
        except ApiException as e:
            state.log.error(f"[installs-list] Failed to list devices: {e}")
            render(f"ERROR: {parse_error_message(e)}")
            raise typer.Exit(1)
    else:
        dev = db.get_device()
        if not dev or not dev.get("id"):
            render("There is no active device.")
            raise typer.Exit(1)
        device_id = dev.get("id")

    kwargs = {}
    if appname:
        kwargs["application_name"] = appname
    if package:
        kwargs["package_name"] = package
    if state_filter:
        kwargs["install_state"] = state_filter

    try:
        response = device_client.get_app_installs(
            enterprise_id, device_id, limit=limit, offset=offset, **kwargs
        )
    except ApiException as e:
        state.log.error(f"[installs-list] Failed to list installs: {e}")
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)

    render(f"Total Number of Installs: {response.count}")
    if not json_output:
        label = {
            "id": "ID",
            "application_name": "APPLICATION",
            "package_name": "PACKAGE",
            "version_code": "VERSION",
            "install_state": "STATE",
        }
        installs = [
            {
                label["id"]: i.id,
                label["application_name"]: i.application.application_name,
                label["package_name"]: i.application.package_name,
                label["version_code"]: i.application.version.version_code,
                label["install_state"]: i.install_state,
            }
            for i in response.results
        ]
        render(installs, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
    else:
        installs = [
            {
                "id": i.id,
                "application_name": i.application.application_name,
                "package_name": i.application.package_name,
                "version_code": i.application.version.version_code,
                "install_state": i.install_state,
            }
            for i in response.results
        ]
        render(installs, format=OutputFormat.JSON.value)
