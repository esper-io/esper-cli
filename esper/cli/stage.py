"""
Pipeline Stage commands — nested under `pipeline`.
`espercli pipeline stage create/edit/show/remove`
"""
from typing import Optional

import typer

from esper.cli.output import render
from esper.cli.state import state, validate_creds
from esper.controllers.enums import OutputFormat
from esper.ext.db_wrapper import DBWrapper
from esper.ext.pipeline_api import (
    get_stage_url, create_stage, edit_stage, list_stages, delete_api,
    APIException, render_single_dict,
)

app = typer.Typer(help="Pipeline Stage commands")


# ---------------------------------------------------------------------------
# Commands
# ---------------------------------------------------------------------------

@app.command("create")
def stage_create(
    pipeline_id: Optional[str] = typer.Option(None, "--pipeline-id", "-p", help="Pipeline ID"),
    name: Optional[str] = typer.Option(None, "--name", "-n", help="Stage name"),
    order: Optional[int] = typer.Option(None, "--order", "-o", help="Stage ordering (unique within pipeline)"),
    desc: Optional[str] = typer.Option(None, "--desc", help="Stage description"),
):
    """Add a Stage to a Pipeline."""
    validate_creds()
    db = DBWrapper(state.creds)
    environment = db.get_configure().get("environment")
    enterprise_id = db.get_enterprise_id()
    api_key = db.get_configure().get("api_key")

    if not pipeline_id:
        pipeline_id = typer.prompt("Enter the Pipeline ID")
    if not name:
        name = typer.prompt("Name of the Stage")
    if order is None:
        order = int(typer.prompt("Order of this Stage"))
    if desc is None:
        desc = typer.prompt("Description for this Stage [optional]", default="")

    url = get_stage_url(environment, enterprise_id, pipeline_id)
    try:
        response = create_stage(url, api_key, name, order, desc)
    except APIException:
        render("ERROR in connecting to Environment!")
        raise typer.Exit(1)

    if not response.ok:
        _handle_stage_error(response, "stage-create")
        raise typer.Exit(1)

    data = render_single_dict(response.json())
    render("Added Stage to Pipeline Successfully! Details:")
    render(data, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")


@app.command("edit")
def stage_edit(
    pipeline_id: Optional[str] = typer.Option(None, "--pipeline-id", "-p", help="Pipeline ID"),
    stage_id: Optional[str] = typer.Option(None, "--stage-id", "-s", help="Stage ID"),
    name: Optional[str] = typer.Option(None, "--name", "-n", help="New stage name"),
    order: Optional[int] = typer.Option(None, "--order", "-o", help="New stage ordering"),
    desc: Optional[str] = typer.Option(None, "--desc", help="New stage description"),
):
    """Edit a Stage."""
    validate_creds()
    db = DBWrapper(state.creds)
    environment = db.get_configure().get("environment")
    enterprise_id = db.get_enterprise_id()
    api_key = db.get_configure().get("api_key")

    if not pipeline_id:
        pipeline_id = typer.prompt("Enter the Pipeline ID")
    if not stage_id:
        stage_id = typer.prompt("Enter the Stage ID")
    if name is None:
        name = typer.prompt("Change the name of the Stage", default="")
    if desc is None:
        desc = typer.prompt("Change the description [optional]", default="")
    if order is None:
        order_str = typer.prompt("Change the ordering [optional]", default="")
        order = int(order_str) if order_str else None

    url = get_stage_url(environment, enterprise_id, pipeline_id=pipeline_id, stage_id=stage_id)
    try:
        response = edit_stage(url, api_key, name, order, desc)
    except APIException:
        render("ERROR in connecting to Environment!")
        raise typer.Exit(1)

    if not response.ok:
        _handle_stage_error(response, "stage-edit")
        raise typer.Exit(1)

    data = render_single_dict(response.json())
    render("Edited Stage for this Pipeline Successfully! Details:")
    render(data, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")


@app.command("show")
def stage_show(
    pipeline_id: Optional[str] = typer.Option(None, "--pipeline-id", "-p", help="Pipeline ID"),
):
    """List all Stages in a Pipeline."""
    validate_creds()
    db = DBWrapper(state.creds)
    environment = db.get_configure().get("environment")
    enterprise_id = db.get_enterprise_id()
    api_key = db.get_configure().get("api_key")

    if not pipeline_id:
        pipeline_id = typer.prompt("Enter the Pipeline ID")

    url = get_stage_url(environment, enterprise_id, pipeline_id=pipeline_id)
    try:
        response = list_stages(url, api_key)
    except APIException:
        render("ERROR in connecting to Environment!")
        raise typer.Exit(1)

    if not response.ok:
        _handle_stage_error(response, "stage-show")
        raise typer.Exit(1)

    data = response.json().get("results") or []
    render_data = [
        {
            "ID": s.get("id"),
            "NAME": s.get("name"),
            "DESCRIPTION": s.get("description"),
            "ORDERING": s.get("ordering"),
            "OPERATIONS": len(s.get("operations", [])),
            "VERSION": s.get("version"),
        }
        for s in data
    ]

    render("Listing Stages for the Pipeline! Details:")
    render(render_data, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")


@app.command("remove")
def stage_remove(
    pipeline_id: Optional[str] = typer.Option(None, "--pipeline-id", "-p", help="Pipeline ID"),
    stage_id: Optional[str] = typer.Option(None, "--stage-id", "-s", help="Stage ID"),
):
    """Remove a Stage from a Pipeline."""
    validate_creds()
    db = DBWrapper(state.creds)
    environment = db.get_configure().get("environment")
    enterprise_id = db.get_enterprise_id()
    api_key = db.get_configure().get("api_key")

    if not pipeline_id:
        pipeline_id = typer.prompt("Enter the Pipeline ID")
    if not stage_id:
        stage_id = typer.prompt("Enter the Stage ID")

    url = get_stage_url(environment, enterprise_id, pipeline_id=pipeline_id, stage_id=stage_id)
    try:
        response = delete_api(url, api_key)
    except APIException:
        render("ERROR in connecting to Environment!")
        raise typer.Exit(1)

    if not response.ok:
        _handle_stage_error(response, "stage-remove")
        raise typer.Exit(1)

    render("Removed Stage for this Pipeline Successfully!")


# ---------------------------------------------------------------------------
# Internal
# ---------------------------------------------------------------------------

def _handle_stage_error(response, tag: str):
    state.log.debug(f"[{tag}] Response not OK. Status: {response.status_code}")
    if response.status_code == 400:
        errors = response.json().get("meta", {}).get("non_field_errors")
        if errors:
            state.log.error(f"Validation Error: {errors}")
        field_errors = response.json().get("errors")
        if field_errors:
            state.log.error(f"Validation Error: {field_errors}")
    elif response.status_code == 404:
        state.log.error("Stage URL not found!")
    elif response.status_code == 500:
        state.log.error(f"Internal Server Error! {response.json()}")
