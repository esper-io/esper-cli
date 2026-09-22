"""
Content commands — replaces esper/controllers/content/content.py.
`espercli content list/show/upload/modify/delete`
"""
import os
import random
import time
from pathlib import Path
from typing import Optional

import typer
from esperclient.rest import ApiException
from tqdm import tqdm

from esper.cli.output import render
from esper.cli.state import state, validate_creds, parse_error_message
from esper.controllers.enums import OutputFormat
from esper.ext.api_client import APIClient
from esper.ext.db_wrapper import DBWrapper

app = typer.Typer(help="Content management commands")


# ---------------------------------------------------------------------------
# Shared helpers
# ---------------------------------------------------------------------------

def _content_basic_response(content, fmt=OutputFormat.TABULATED):
    enterprise_id = content.enterprise.split("/")[-2]
    if fmt == OutputFormat.TABULATED:
        title, details = "TITLE", "DETAILS"
        return [
            {title: "id", details: content.id},
            {title: "name", details: content.name},
            {title: "is_dir", details: content.is_dir},
            {title: "kind", details: content.kind},
            {title: "hash", details: content.hash},
            {title: "size", details: content.size},
            {title: "path", details: content.path},
            {title: "permissions", details: content.permissions},
            {title: "tags", details: content.tags},
            {title: "description", details: content.description},
            {title: "created", details: content.created},
            {title: "modified", details: content.modified},
            {title: "enterprise", details: enterprise_id},
            {title: "owner", details: content.owner.username},
        ]
    return {
        "id": content.id,
        "download_url": content.download_url,
        "name": content.name,
        "key": content.key,
        "is_dir": content.is_dir,
        "kind": content.kind,
        "hash": content.hash,
        "size": content.size,
        "path": content.path,
        "permissions": content.permissions,
        "tags": content.tags,
        "description": content.description,
        "created": str(content.created),
        "modified": str(content.modified),
        "enterprise": content.enterprise,
        "owner": str(content.owner),
    }


# ---------------------------------------------------------------------------
# Commands
# ---------------------------------------------------------------------------

@app.command("list")
def content_list(
    search: Optional[str] = typer.Option(None, "--search", "-s", help="Search by name/tags/description"),
    limit: int = typer.Option(20, "--limit", "-l", help="Results per page"),
    offset: int = typer.Option(0, "--offset", "-i", help="Initial index"),
    json_output: bool = typer.Option(False, "--json", "-j", help="Render result in JSON format"),
):
    """List all content."""
    validate_creds()
    db = DBWrapper(state.creds)
    content_client = APIClient(db.get_configure()).get_content_api_client()
    enterprise_id = db.get_enterprise_id()

    kwargs = {}
    if search:
        kwargs["search"] = search

    try:
        response = content_client.get_all_content(enterprise_id, limit=limit, offset=offset, **kwargs)
    except ApiException as e:
        state.log.error(f"[content-list] Failed to list contents: {e}")
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)

    render(f"Total Number of Contents: {response.count}")
    if not json_output:
        label = {"id": "ID", "name": "NAME", "description": "DESCRIPTION",
                 "tags": "TAGS", "size": "SIZE", "created_on": "CREATED ON"}
        contents = [
            {
                label["id"]: c.id,
                label["name"]: c.name,
                label["description"]: c.description,
                label["tags"]: c.tags,
                label["size"]: c.size,
                label["created_on"]: c.created,
            }
            for c in response.results
        ]
        render(contents, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
    else:
        contents = [
            {
                "id": c.id,
                "download_url": c.download_url,
                "name": c.name,
                "key": c.key,
                "is_dir": c.is_dir,
                "kind": c.kind,
                "hash": c.hash,
                "size": c.size,
                "path": c.path,
                "permissions": c.permissions,
                "tags": c.tags,
                "description": c.description,
                "created": str(c.created),
                "modified": str(c.modified),
                "enterprise": c.enterprise,
                "owner": str(c.owner),
            }
            for c in response.results
        ]
        render(contents, format=OutputFormat.JSON.value)


@app.command("show")
def content_show(
    content_id: str = typer.Argument(..., help="Content ID"),
    json_output: bool = typer.Option(False, "--json", "-j", help="Render result in JSON format"),
):
    """Show content details."""
    validate_creds()
    db = DBWrapper(state.creds)
    content_client = APIClient(db.get_configure()).get_content_api_client()
    enterprise_id = db.get_enterprise_id()

    try:
        response = content_client.get_content(content_id, enterprise_id)
    except ApiException as e:
        state.log.error(f"[content-show] Failed to show content: {e}")
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)

    if not json_output:
        render(_content_basic_response(response), format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
    else:
        render(_content_basic_response(response, OutputFormat.JSON), format=OutputFormat.JSON.value)


@app.command("upload")
def content_upload(
    content_file: str = typer.Argument(..., help="File to upload"),
    json_output: bool = typer.Option(False, "--json", "-j", help="Render result in JSON format"),
):
    """Upload content."""
    validate_creds()
    db = DBWrapper(state.creds)
    content_client = APIClient(db.get_configure()).get_content_api_client()
    enterprise_id = db.get_enterprise_id()

    try:
        filesize = os.path.getsize(content_file)
        random_no = random.randint(1, 50)
        response = None
        with tqdm(total=int(filesize), unit="B", unit_scale=True, miniters=1,
                  desc="Uploading......", unit_divisor=1024) as pbar:
            for i in range(100):
                if i == random_no:
                    response = content_client.post_content(enterprise_id, content_file)
                time.sleep(0.07)
                pbar.set_postfix(file=Path(content_file).name, refresh=False)
                pbar.update(int(filesize / 100))
    except ApiException as e:
        state.log.error(f"[content-upload] Failed to upload content: {e}")
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)

    if not json_output:
        render(_content_basic_response(response), format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
    else:
        render(_content_basic_response(response, OutputFormat.JSON), format=OutputFormat.JSON.value)


@app.command("modify")
def content_modify(
    content_id: str = typer.Argument(..., help="Content ID"),
    tags: Optional[str] = typer.Option(
        None, "--tags", "-t", help="Space-separated list of tags"
    ),
    description: Optional[str] = typer.Option(None, "--description", "-d", help="Description"),
    json_output: bool = typer.Option(False, "--json", "-j", help="Render result in JSON format"),
):
    """Modify content tags and/or description."""
    validate_creds()
    db = DBWrapper(state.creds)
    content_client = APIClient(db.get_configure()).get_content_api_client()
    enterprise_id = db.get_enterprise_id()

    data = {}
    if tags:
        data["tags"] = tags.split()
    if description:
        data["description"] = description

    if not data:
        render("Both tags and description values are empty")
        raise typer.Exit(1)

    try:
        response = content_client.patch_content(content_id, enterprise_id, data=data)
    except ApiException as e:
        state.log.error(f"[content-modify] Failed to modify content: {e}")
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)

    if not json_output:
        render(_content_basic_response(response), format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
    else:
        render(_content_basic_response(response, OutputFormat.JSON), format=OutputFormat.JSON.value)


@app.command("delete")
def content_delete(
    content_id: str = typer.Argument(..., help="Content ID to delete"),
    yes: bool = typer.Option(False, "--yes", "-y", help="Skip confirmation prompt"),
):
    """Delete content."""
    if not yes:
        typer.confirm(f"Delete content {content_id}? This cannot be undone.", abort=True)
    validate_creds()
    db = DBWrapper(state.creds)
    content_client = APIClient(db.get_configure()).get_content_api_client()
    enterprise_id = db.get_enterprise_id()

    try:
        content_client.delete_content(content_id, enterprise_id)
        render(f"Content with id {content_id} deleted successfully.")
    except ApiException as e:
        state.log.error(f"[content-delete] Failed to delete content: {e}")
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)
