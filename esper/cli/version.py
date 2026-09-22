"""
Application version commands — replaces esper/controllers/application/version.py.
`espercli version list/show/delete/devices`
"""
from http import HTTPStatus
from typing import Optional

import typer
from esperclient.rest import ApiException

from esper.cli.completions import app_name_complete, legacy_format_complete
from esper.cli.output import render
from esper.cli.state import state, validate_creds, parse_error_message
from esper.controllers.enums import OutputFormat
from esper.ext.api_client import APIClient
from esper.ext.db_wrapper import DBWrapper

app = typer.Typer(help="Application version commands")


# ---------------------------------------------------------------------------
# Shared helpers
# ---------------------------------------------------------------------------

def _get_application_id(db, application_arg: Optional[str], tag: str) -> str:
    if application_arg:
        return application_arg
    active = db.get_application()
    if not active or not active.get("id"):
        render("There is no active application.")
        raise typer.Exit(1)
    return active.get("id")


def _version_basic_response(version, fmt=OutputFormat.TABULATED, legacy_format=True):
    valid_keys = ["id", "version_code", "version_name", "build_number", "size_in_mb", "release_track"]

    if fmt == OutputFormat.JSON:
        renderable = {k: v for k, v in version.to_dict().items() if k in valid_keys}
        renderable["installed_count"] = version.installed_count or 0
    else:
        title, details = "TITLE", "DETAILS"
        if legacy_format:
            filtered = {k: v for k, v in version.to_dict().items()
                        if k in valid_keys and k != "version_name"}
        else:
            filtered = {k: v for k, v in version.to_dict().items()
                        if k in valid_keys and k != "build_number"}
        renderable = [{title: k, details: v} for k, v in filtered.items()]
        renderable.append({title: "installed_count", details: version.installed_count or 0})
    return renderable


# ---------------------------------------------------------------------------
# Commands
# ---------------------------------------------------------------------------

@app.command("list")
def version_list(
    application: Optional[str] = typer.Option(
        None, "--app", "-a", help="Application ID (default: active application)",
        shell_complete=app_name_complete,
    ),
    version_code: Optional[str] = typer.Option(None, "--code", "-c", help="Version code filter"),
    build_number: Optional[str] = typer.Option(None, "--number", "-n", help="Build number filter"),
    legacy_format: str = typer.Option(
        "true", "--legacy-format", "-lf", help="Use legacy format [true/false]",
        shell_complete=legacy_format_complete,
    ),
    limit: int = typer.Option(20, "--limit", "-l", help="Results per page"),
    offset: int = typer.Option(0, "--offset", "-i", help="Initial index"),
    json_output: bool = typer.Option(False, "--json", "-j", help="Render result in JSON format"),
):
    """List application versions."""
    validate_creds()
    db = DBWrapper(state.creds)
    application_client = APIClient(db.get_configure()).get_application_api_client()
    enterprise_id = db.get_enterprise_id()

    application_id = _get_application_id(db, application, "version-list")
    use_legacy = legacy_format.lower() == "true"

    kwargs = {}
    if version_code:
        kwargs["version_code"] = version_code
    if build_number:
        kwargs["build_number"] = build_number

    try:
        response = application_client.get_app_versions(
            application_id, enterprise_id, limit=limit, offset=offset,
            legacy_format=use_legacy, **kwargs
        )
    except ApiException as e:
        state.log.error(f"[version-list] Failed to list versions: {e}")
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)

    render(f"Total Number of Versions: {response.count}")
    if not json_output:
        if use_legacy:
            label = {"id": "ID", "version_code": "VERSION CODE", "build_number": "BUILD NUMBER",
                     "size_in_mb": "SIZE IN MB", "release_track": "RELEASE TRACK",
                     "installed_count": "INSTALLED COUNT"}
        else:
            label = {"id": "ID", "version_name": "VERSION NAME", "version_code": "VERSION CODE",
                     "size_in_mb": "SIZE IN MB", "release_track": "RELEASE TRACK",
                     "installed_count": "INSTALLED COUNT"}

        versions = []
        for v in response.results:
            if use_legacy:
                row = {
                    label["id"]: v.id,
                    label["version_code"]: v.version_code,
                    label["build_number"]: getattr(v, "build_number", "") or "",
                    label["size_in_mb"]: v.size_in_mb,
                    label["release_track"]: v.release_track,
                    label["installed_count"]: v.installed_count or 0,
                }
            else:
                row = {
                    label["id"]: v.id,
                    label["version_name"]: getattr(v, "version_name", "") or "",
                    label["version_code"]: v.version_code,
                    label["size_in_mb"]: v.size_in_mb,
                    label["release_track"]: v.release_track,
                    label["installed_count"]: v.installed_count or 0,
                }
            versions.append(row)
        render(versions, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
    else:
        versions = []
        for v in response.results:
            if use_legacy:
                row = {
                    "id": v.id, "version_code": v.version_code,
                    "build_number": getattr(v, "build_number", None),
                    "size_in_mb": v.size_in_mb, "release_track": v.release_track,
                    "installed_count": v.installed_count or 0,
                }
            else:
                row = {
                    "id": v.id,
                    "version_name": getattr(v, "version_name", None),
                    "version_code": v.version_code,
                    "size_in_mb": v.size_in_mb, "release_track": v.release_track,
                    "installed_count": v.installed_count or 0,
                }
            versions.append(row)
        render(versions, format=OutputFormat.JSON.value)


@app.command("show")
def version_show(
    version_id: str = typer.Argument(..., help="Version ID"),
    application: Optional[str] = typer.Option(
        None, "--app", "-a", help="Application ID (default: active application)",
        shell_complete=app_name_complete,
    ),
    legacy_format: str = typer.Option(
        "true", "--legacy-format", "-lf", help="Use legacy format [true/false]",
        shell_complete=legacy_format_complete,
    ),
    json_output: bool = typer.Option(False, "--json", "-j", help="Render result in JSON format"),
):
    """Show application version details."""
    validate_creds()
    db = DBWrapper(state.creds)
    application_client = APIClient(db.get_configure()).get_application_api_client()
    enterprise_id = db.get_enterprise_id()

    application_id = _get_application_id(db, application, "version-show")
    use_legacy = legacy_format.lower() == "true"

    try:
        response = application_client.get_app_version(
            version_id, application_id, enterprise_id, legacy_format=use_legacy
        )
    except ApiException as e:
        state.log.error(f"[version-show] Failed to show version: {e}")
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)

    if not json_output:
        render(_version_basic_response(response, fmt=OutputFormat.TABULATED, legacy_format=use_legacy),
               format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
    else:
        render(_version_basic_response(response, fmt=OutputFormat.JSON, legacy_format=use_legacy),
               format=OutputFormat.JSON.value)


@app.command("delete")
def version_delete(
    version_id: str = typer.Argument(..., help="Version ID to delete"),
    application: Optional[str] = typer.Option(
        None, "--app", "-a", help="Application ID (default: active application)",
        shell_complete=app_name_complete,
    ),
    yes: bool = typer.Option(False, "--yes", "-y", help="Skip confirmation prompt"),
):
    """Delete an application version."""
    if not yes:
        typer.confirm(f"Delete version {version_id}? This cannot be undone.", abort=True)
    validate_creds()
    db = DBWrapper(state.creds)
    application_client = APIClient(db.get_configure()).get_application_api_client()
    enterprise_id = db.get_enterprise_id()

    application_id = _get_application_id(db, application, "version-delete")

    try:
        application_client.delete_app_version(version_id, application_id, enterprise_id)
        render(f"Version with id {version_id} deleted successfully")
    except ApiException as e:
        state.log.error(f"[version-delete] Failed to delete version: {e}")
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)

    # Unset active application if it was deleted
    try:
        application_client.get_application(application_id, enterprise_id)
    except ApiException as e:
        if e.status == HTTPStatus.NOT_FOUND:
            active_app = db.get_application()
            if active_app and active_app.get("id") == application_id:
                db.unset_application()


@app.command("devices")
def version_devices(
    version_id: str = typer.Argument(..., help="Version ID"),
    application: Optional[str] = typer.Option(
        None, "--app", "-a", help="Application ID (default: active application)",
        shell_complete=app_name_complete,
    ),
    search: Optional[str] = typer.Option(None, "--search", "-s", help="Search term"),
    limit: int = typer.Option(20, "--limit", "-l", help="Results per page"),
    offset: int = typer.Option(0, "--offset", "-o", help="Initial index"),
    json_output: bool = typer.Option(False, "--json", "-j", help="Render result in JSON format"),
):
    """List devices with this application version installed."""
    validate_creds()
    db = DBWrapper(state.creds)
    application_client = APIClient(db.get_configure()).get_application_api_client()
    enterprise_id = db.get_enterprise_id()

    application_id = _get_application_id(db, application, "version-devices")

    kwargs = {}
    if search:
        kwargs["search"] = search

    try:
        response = application_client.get_install_devices(
            version_id, application_id, enterprise_id, limit=limit, offset=offset, **kwargs
        )
    except ApiException as e:
        state.log.error(f"[version-devices] Failed to list devices: {e}")
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)

    render(f"Total Number of Devices: {response.count}")
    if not json_output:
        label = {"id": "ID", "device_name": "DEVICE NAME",
                 "alias_name": "ALIAS NAME", "group_name": "GROUP NAME"}
        devices = [
            {
                label["id"]: d.id,
                label["device_name"]: d.device_name,
                label["alias_name"]: d.alias_name,
                label["group_name"]: d.group_name,
            }
            for d in response.results
        ]
        render(devices, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
    else:
        devices = [
            {"id": d.id, "device_name": d.device_name,
             "alias_name": d.alias_name, "group_name": d.group_name}
            for d in response.results
        ]
        render(devices, format=OutputFormat.JSON.value)
