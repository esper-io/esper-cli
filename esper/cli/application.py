"""
Application commands — replaces esper/controllers/application/application.py.
`espercli app list/show/upload/download/delete/set-active/unset-active`
"""
import os
import random
import time
from pathlib import Path
from typing import Optional

import requests
import typer
from esperclient.rest import ApiException
from tqdm import tqdm

from esper.cli.completions import app_name_complete
from esper.cli.output import render
from esper.cli.state import state, validate_creds, parse_error_message
from esper.controllers.enums import OutputFormat
from esper.ext.api_client import APIClient
from esper.ext.db_wrapper import DBWrapper

app = typer.Typer(help="Application commands")


# ---------------------------------------------------------------------------
# Shared helpers
# ---------------------------------------------------------------------------

def _application_basic_response(application, fmt=OutputFormat.TABULATED):
    valid_keys = [
        "id", "application_name", "package_name", "developer",
        "category", "content_rating", "compatibility",
    ]
    if fmt == OutputFormat.JSON:
        return {k: v for k, v in application.to_dict().items() if k in valid_keys}
    title, details = "TITLE", "DETAILS"
    return [{title: k, details: v} for k, v in application.to_dict().items() if k in valid_keys]


# ---------------------------------------------------------------------------
# Commands
# ---------------------------------------------------------------------------

@app.command("list")
def app_list(
    name: Optional[str] = typer.Option(None, "--name", "-n", help="Application name"),
    package: Optional[str] = typer.Option(None, "--package", "-p", help="Package name"),
    limit: int = typer.Option(20, "--limit", "-l", help="Results per page"),
    offset: int = typer.Option(0, "--offset", "-i", help="Initial index"),
    all_results: bool = typer.Option(
        False, "--all", "-A",
        help="Fetch ALL applications, auto-paginating (ignores --limit / --offset).",
    ),
    json_output: bool = typer.Option(False, "--json", "-j", help="Render result in JSON format"),
):
    """List applications in the enterprise."""
    validate_creds()
    db = DBWrapper(state.creds)
    application_client = APIClient(db.get_configure()).get_application_api_client()
    enterprise_id = db.get_enterprise_id()

    kwargs: dict = {"is_hidden": False}
    if name:
        kwargs["application_name"] = name
    if package:
        kwargs["package_name"] = package

    try:
        if all_results:
            from esper.cli.output import console as _console
            results = []
            page_size, cur_offset = 20, 0
            with _console.status("[dim]Fetching all applications…[/dim]", spinner="dots"):
                while True:
                    page = application_client.get_all_applications(
                        enterprise_id, limit=page_size, offset=cur_offset, **kwargs
                    )
                    results.extend(page.results)
                    if len(results) >= page.count or not page.results:
                        break
                    cur_offset += page_size

            class _FR:
                def __init__(self, r):
                    self.results = r
                    self.count = len(r)
            response = _FR(results)
        else:
            response = application_client.get_all_applications(
                enterprise_id, limit=limit, offset=offset, **kwargs
            )
    except ApiException as e:
        state.log.error(f"[application-list] Failed to list applications: {e}")
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)

    render(f"Total Number of Applications: {response.count}")
    if not json_output:
        label = {"id": "ID", "name": "NAME", "package": "PACKAGE NAME"}
        applications = [
            {
                label["id"]: a.id,
                label["name"]: a.application_name,
                label["package"]: a.package_name,
            }
            for a in response.results
        ]
        render(applications, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
    else:
        applications = [
            {"id": a.id, "name": a.application_name, "package": a.package_name}
            for a in response.results
        ]
        render(applications, format=OutputFormat.JSON.value)


@app.command("show")
def app_show(
    application_id: str = typer.Argument(..., help="Application ID"),
    active: bool = typer.Option(
        False, "--active", "-a", help="Set as the active application"
    ),
    json_output: bool = typer.Option(False, "--json", "-j", help="Render result in JSON format"),
):
    """Show application details (and optionally set as active)."""
    validate_creds()
    db = DBWrapper(state.creds)
    application_client = APIClient(db.get_configure()).get_application_api_client()
    enterprise_id = db.get_enterprise_id()

    try:
        response = application_client.get_application(application_id, enterprise_id)
    except ApiException as e:
        state.log.error(f"[application-show] Failed to show application: {e}")
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)

    if active:
        db.set_application({"id": application_id})

    if not json_output:
        render(_application_basic_response(response), format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
    else:
        render(_application_basic_response(response, OutputFormat.JSON), format=OutputFormat.JSON.value)


@app.command("upload")
def app_upload(
    application_file: str = typer.Argument(..., help="Path to the APK/application file"),
    json_output: bool = typer.Option(False, "--json", "-j", help="Render result in JSON format"),
):
    """Upload an application."""
    validate_creds()
    db = DBWrapper(state.creds)
    application_client = APIClient(db.get_configure()).get_application_api_client()
    enterprise_id = db.get_enterprise_id()

    try:
        filesize = os.path.getsize(application_file)
        random_no = random.randint(1, 50)
        response = None
        with tqdm(total=int(filesize), unit="B", unit_scale=True, miniters=1,
                  desc="Uploading......", unit_divisor=1024) as pbar:
            for i in range(100):
                if i == random_no:
                    response = application_client.upload(enterprise_id, application_file)
                time.sleep(0.07)
                pbar.set_postfix(file=Path(application_file).name, refresh=False)
                pbar.update(int(filesize / 100))
        application = response.application
    except ApiException as e:
        state.log.error(f"[application-upload] Failed to upload application: {e}")
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)

    valid_keys = ["id", "application_name", "package_name", "developer",
                  "category", "content_rating", "compatibility"]

    if not json_output:
        title, details = "TITLE", "DETAILS"
        renderable = [{title: k, details: v}
                      for k, v in application.to_dict().items() if k in valid_keys]
        if application and application.versions:
            version = application.versions[0]
            renderable.append({title: "version_id", details: version.id})
            renderable.append({title: "version_code", details: version.version_code})
            if hasattr(version, "version_name") and version.version_name:
                renderable.append({title: "version_name", details: version.version_name})
            elif hasattr(version, "build_number") and version.build_number:
                renderable.append({title: "build_number", details: version.build_number})
        render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
    else:
        renderable = {k: v for k, v in application.to_dict().items() if k in valid_keys}
        if application and application.versions:
            version = application.versions[0]
            renderable["version_id"] = version.id
            renderable["version_code"] = version.version_code
            if hasattr(version, "version_name") and version.version_name:
                renderable["version_name"] = version.version_name
            elif hasattr(version, "build_number") and version.build_number:
                renderable["build_number"] = version.build_number
        render(renderable, format=OutputFormat.JSON.value)


@app.command("download")
def app_download(
    version_id: str = typer.Argument(..., help="Version ID to download"),
    application: Optional[str] = typer.Option(
        None, "--app", "-a", help="Application ID (default: active application)",
        shell_complete=app_name_complete,
    ),
    dest: Optional[str] = typer.Option(None, "--dest", "-d", help="Destination file path"),
):
    """Download an application version."""
    validate_creds()
    db = DBWrapper(state.creds)
    application_client = APIClient(db.get_configure()).get_application_api_client()
    enterprise_id = db.get_enterprise_id()

    if application:
        application_id = application
    else:
        active_app = db.get_application()
        if not active_app or not active_app.get("id"):
            render("There is no active application.")
            raise typer.Exit(1)
        application_id = active_app.get("id")

    if not dest:
        render("Destination file path cannot be empty.")
        raise typer.Exit(1)

    try:
        response = application_client.get_app_version(version_id, application_id, enterprise_id)
    except ApiException as e:
        state.log.error(f"[app-download] Failed to get version: {e}")
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)

    url = response.app_file
    file_size = int(response.size_in_mb * 1024 * 1024)
    pbar = tqdm(total=file_size, initial=0, unit="B", unit_scale=True, desc="Downloading......")
    req = requests.get(url, stream=True, timeout=30)
    with open(dest, "ab") as f:
        for chunk in req.iter_content(chunk_size=1024):
            if chunk:
                f.write(chunk)
                pbar.update(1024)
                time.sleep(0.001)
    pbar.close()


@app.command("delete")
def app_delete(
    application_id: str = typer.Argument(..., help="Application ID to delete"),
    yes: bool = typer.Option(False, "--yes", "-y", help="Skip confirmation prompt"),
):
    """Delete an application."""
    if not yes:
        typer.confirm(
            f"Delete application {application_id}? This cannot be undone.",
            abort=True,
        )
    validate_creds()
    db = DBWrapper(state.creds)
    application_client = APIClient(db.get_configure()).get_application_api_client()
    enterprise_id = db.get_enterprise_id()

    try:
        application_client.delete_application(application_id, enterprise_id)
        render(f"Application with id {application_id} deleted successfully")
        active_app = db.get_application()
        if active_app and active_app.get("id") == application_id:
            db.unset_application()
    except ApiException as e:
        state.log.debug(f"[application-delete] Failed to delete application: {e}")
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)


@app.command("set-active")
def app_set_active(
    app_id: Optional[str] = typer.Option(None, "--id", "-i", help="Application ID"),
    json_output: bool = typer.Option(False, "--json", "-j", help="Render result in JSON format"),
):
    """Set or show the active application."""
    validate_creds()
    db = DBWrapper(state.creds)
    application_client = APIClient(db.get_configure()).get_application_api_client()
    enterprise_id = db.get_enterprise_id()

    if app_id:
        application_id = app_id
    else:
        active_app = db.get_application()
        if active_app is None or active_app.get("id") is None:
            render("There is no active application.")
            raise typer.Exit(1)
        application_id = active_app.get("id")

    try:
        response = application_client.get_application(application_id, enterprise_id)
        db.set_application({"id": application_id})
    except ApiException as e:
        state.log.error(f"[application-active] Failed to show active application: {e}")
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)

    if not json_output:
        render(_application_basic_response(response), format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
    else:
        render(_application_basic_response(response, OutputFormat.JSON), format=OutputFormat.JSON.value)


@app.command("unset-active")
def app_unset_active():
    """Unset the current active application."""
    validate_creds()
    db = DBWrapper(state.creds)

    application = db.get_application()
    if application is None or application.get("id") is None:
        render("There is no active application.")
        raise typer.Exit(1)

    db.unset_application()
    render(f"Unset the active application {application.get('id')}")
