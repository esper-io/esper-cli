"""
Pipeline Stage Operation commands — nested under `pipeline stage`.
`espercli pipeline stage operation create/edit/show/remove`
"""
from enum import Enum
from typing import Optional

import typer
from esperclient.rest import ApiException

from esper.cli.output import render, prompt_options
from esper.cli.state import state, validate_creds, parse_error_message
from esper.controllers.enums import OutputFormat
from esper.ext.api_client import APIClient
from esper.ext.db_wrapper import DBWrapper
from esper.ext.pipeline_api import (
    get_operation_url, create_operation, edit_operation, list_stages,
    delete_api, APIException as PipelineAPIException, render_single_dict,
    get_group_command_url,
)

app = typer.Typer(help="Pipeline Stage Operation commands")


class ActionEnums(Enum):
    APP_INSTALL = "App Install to a Group of Devices"
    APP_UNINSTALL = "App Uninstall to a Group of Devices"
    REBOOT = "Reboot a Group of Devices"

    @classmethod
    def choices_values(cls):
        return [member.value for member in cls]


def _prompt_action() -> str:
    """Interactively prompt for an action type and return the enum name."""
    choices = ActionEnums.choices_values()
    options = [{"selector": i + 1, "prompt": v, "return": v} for i, v in enumerate(choices)]
    selected_value = prompt_options("Action for this Operation:", options=options)
    return ActionEnums(selected_value).name


def _validate_group_name(db, group_name: str):
    """Look up a group by name; raise ApiException if not found."""
    group_client = APIClient(db.get_configure()).get_group_api_client()
    enterprise_id = db.get_enterprise_id()
    response = group_client.get_all_groups(enterprise_id, name=group_name)
    if response.count > 0:
        return response.results[0]
    raise ApiException("No such Group-Name found!")


# ---------------------------------------------------------------------------
# Commands
# ---------------------------------------------------------------------------

@app.command("create")
def operation_create(
    pipeline_id: Optional[str] = typer.Option(None, "--pipeline-id", "-p", help="Pipeline ID"),
    stage_id: Optional[str] = typer.Option(None, "--stage-id", "-s", help="Stage ID"),
    name: Optional[str] = typer.Option(None, "--name", "-n", help="Operation name"),
    action: Optional[str] = typer.Option(None, "--action", "-a", help="Action for this operation"),
    group_name: Optional[str] = typer.Option(None, "--group", "-g", help="Group name for commands"),
    desc: Optional[str] = typer.Option(None, "--desc", help="Operation description"),
):
    """Add an Operation to a Stage."""
    validate_creds()
    db = DBWrapper(state.creds)
    environment = db.get_configure().get("environment")
    enterprise_id = db.get_enterprise_id()
    api_key = db.get_configure().get("api_key")

    if not pipeline_id:
        pipeline_id = typer.prompt("Enter the Pipeline ID")
    if not stage_id:
        stage_id = typer.prompt("Enter the Stage ID")
    if not name:
        name = typer.prompt("Name of the Operation")
    if not action:
        action = _prompt_action()
    if not group_name:
        group_name = typer.prompt("Name of the Group (to which the command must be fired)")

    try:
        group_obj = _validate_group_name(db, group_name)
        group_url = get_group_command_url(environment, enterprise_id, group_obj.id)
    except ApiException as e:
        state.log.error(f"[operation-create] Failed to find group '{group_name}': {e}")
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)

    if desc is None:
        desc = typer.prompt("Description for this Operation [optional]", default="")

    url = get_operation_url(environment, enterprise_id, pipeline_id, stage_id)
    try:
        response = create_operation(url, api_key, name, action, desc, group_url)
    except PipelineAPIException:
        render("ERROR in connecting to Environment!")
        raise typer.Exit(1)

    if not response.ok:
        _handle_operation_error(response, "operation-create")
        raise typer.Exit(1)

    data = render_single_dict(response.json())
    render("Added Operation to Stage Successfully! Details:")
    render(data, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")


@app.command("edit")
def operation_edit(
    pipeline_id: Optional[str] = typer.Option(None, "--pipeline-id", "-p", help="Pipeline ID"),
    stage_id: Optional[str] = typer.Option(None, "--stage-id", "-s", help="Stage ID"),
    operation_id: Optional[str] = typer.Option(None, "--operation-id", "-o", help="Operation ID"),
    name: Optional[str] = typer.Option(None, "--name", "-n", help="New operation name"),
    action: Optional[str] = typer.Option(None, "--action", "-a", help="New action"),
    desc: Optional[str] = typer.Option(None, "--desc", help="New description"),
):
    """Edit an Operation."""
    validate_creds()
    db = DBWrapper(state.creds)
    environment = db.get_configure().get("environment")
    enterprise_id = db.get_enterprise_id()
    api_key = db.get_configure().get("api_key")

    if not pipeline_id:
        pipeline_id = typer.prompt("Enter the Pipeline ID")
    if not stage_id:
        stage_id = typer.prompt("Enter the Stage ID")
    if not operation_id:
        operation_id = typer.prompt("Enter the Operation ID")
    if name is None:
        name = typer.prompt("Change the name of the Operation", default="")
    if desc is None:
        desc = typer.prompt("Change the description [optional]", default="")
    if not action:
        action = _prompt_action()

    url = get_operation_url(
        environment, enterprise_id,
        pipeline_id=pipeline_id, stage_id=stage_id, operation_id=operation_id,
    )
    try:
        response = edit_operation(url, api_key, name, action, desc)
    except PipelineAPIException:
        render("ERROR in connecting to Environment!")
        raise typer.Exit(1)

    if not response.ok:
        _handle_operation_error(response, "operation-edit")
        raise typer.Exit(1)

    data = render_single_dict(response.json())
    render("Edited Operation for this Stage Successfully! Details:")
    render(data, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")


@app.command("show")
def operation_show(
    pipeline_id: Optional[str] = typer.Option(None, "--pipeline-id", "-p", help="Pipeline ID"),
    stage_id: Optional[str] = typer.Option(None, "--stage-id", "-s", help="Stage ID"),
):
    """List all Operations in a Stage."""
    validate_creds()
    db = DBWrapper(state.creds)
    environment = db.get_configure().get("environment")
    enterprise_id = db.get_enterprise_id()
    api_key = db.get_configure().get("api_key")

    if not pipeline_id:
        pipeline_id = typer.prompt("Enter the Pipeline ID")
    if not stage_id:
        stage_id = typer.prompt("Enter the Stage ID")

    url = get_operation_url(environment, enterprise_id, pipeline_id=pipeline_id, stage_id=stage_id)
    try:
        response = list_stages(url, api_key)
    except PipelineAPIException:
        render("ERROR in connecting to Environment!")
        raise typer.Exit(1)

    if not response.ok:
        _handle_operation_error(response, "operation-show")
        raise typer.Exit(1)

    data = response.json().get("results") or []
    render_data = [
        {
            "ID": op.get("id"),
            "NAME": op.get("name"),
            "DESCRIPTION": op.get("description"),
            "ORDERING": op.get("ordering"),
            "ACTION": op.get("action"),
            "VERSION": op.get("version"),
        }
        for op in data
    ]

    render("Listing Operations for the Stage! Details:")
    render(render_data, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")


@app.command("remove")
def operation_remove(
    pipeline_id: Optional[str] = typer.Option(None, "--pipeline-id", "-p", help="Pipeline ID"),
    stage_id: Optional[str] = typer.Option(None, "--stage-id", "-s", help="Stage ID"),
    operation_id: Optional[str] = typer.Option(None, "--operation-id", "-o", help="Operation ID"),
):
    """Remove an Operation from a Stage."""
    validate_creds()
    db = DBWrapper(state.creds)
    environment = db.get_configure().get("environment")
    enterprise_id = db.get_enterprise_id()
    api_key = db.get_configure().get("api_key")

    if not pipeline_id:
        pipeline_id = typer.prompt("Enter the Pipeline ID")
    if not stage_id:
        stage_id = typer.prompt("Enter the Stage ID")
    if not operation_id:
        operation_id = typer.prompt("Enter the Operation ID")

    url = get_operation_url(
        environment, enterprise_id,
        pipeline_id=pipeline_id, stage_id=stage_id, operation_id=operation_id,
    )
    try:
        response = delete_api(url, api_key)
    except PipelineAPIException:
        render("ERROR in connecting to Environment!")
        raise typer.Exit(1)

    if not response.ok:
        _handle_operation_error(response, "operation-remove")
        raise typer.Exit(1)

    render("Removed Operation for this Stage Successfully!")


# ---------------------------------------------------------------------------
# Internal
# ---------------------------------------------------------------------------

def _handle_operation_error(response, tag: str):
    state.log.debug(f"[{tag}] Response not OK. Status: {response.status_code}")
    if response.status_code == 400:
        errors = response.json().get("meta", {}).get("non_field_errors")
        if errors:
            state.log.error(f"Validation Error: {errors}")
        field_errors = response.json().get("errors")
        if field_errors:
            msg = response.json().get("message", "")
            if "The fields pipeline, ordering must make a unique set." in msg:
                state.log.error("Operation with same `name` already created for this Stage!")
            else:
                state.log.error(f"Validation Error: {field_errors}")
    elif response.status_code == 404:
        state.log.error("Stage URL not found!")
    elif response.status_code == 500:
        state.log.error(f"Internal Server Error! {response.json()}")
