"""
Pipeline commands — replaces esper/controllers/pipeline/pipeline.py.
`espercli pipeline create/edit/show/remove`
"""
from enum import Enum
from typing import Optional

import typer

from esper.cli.output import render, prompt_options
from esper.cli.state import state, validate_creds
from esper.controllers.enums import OutputFormat
from esper.ext.db_wrapper import DBWrapper
from esper.ext.pipeline_api import (
    get_pipeline_url, create_pipeline, edit_pipeline,
    list_pipelines, fetch_pipelines, delete_api, render_single_dict, APIException,
)

app = typer.Typer(help="Pipeline commands")


class TriggerEventType(Enum):
    NEW_APP_VERSION_EVENT = "NewAppVersionEvent"
    MANUAL_PIPELINE_START_EVENT = "StartPipelineEvent"


def _get_trigger_interactively():
    """Prompt the user for a trigger type and return a trigger dict or None."""
    choice = prompt_options(
        "What type of trigger do you want for your Pipeline?",
        options=[
            {"selector": 1, "prompt": TriggerEventType.NEW_APP_VERSION_EVENT.value,
             "return": TriggerEventType.NEW_APP_VERSION_EVENT.name},
            {"selector": 2, "prompt": "Skip for now...", "return": "skip"},
        ],
    )
    if choice == "skip":
        return None
    if choice == TriggerEventType.NEW_APP_VERSION_EVENT.name:
        app_name = typer.prompt("Enter the Application name")
        package_name = typer.prompt("Enter the Package name")
        return {
            "trigger_event": TriggerEventType.NEW_APP_VERSION_EVENT.value,
            "pre_conditions": {"application_name": app_name, "package_name": package_name},
        }
    return None


# ---------------------------------------------------------------------------
# Commands
# ---------------------------------------------------------------------------

@app.command("create")
def pipeline_create(
    name: Optional[str] = typer.Option(None, "--name", "-n", help="Pipeline name"),
    desc: Optional[str] = typer.Option(None, "--desc", help="Pipeline description"),
):
    """Create a new pipeline."""
    validate_creds()
    db = DBWrapper(state.creds)
    environment = db.get_configure().get("environment")
    enterprise_id = db.get_enterprise_id()
    api_key = db.get_configure().get("api_key")

    if not name:
        name = typer.prompt("Name of the Pipeline")
    if desc is None:
        desc = typer.prompt("Description for this Pipeline [optional]", default="")

    trigger = _get_trigger_interactively()

    url = get_pipeline_url(environment, enterprise_id)
    try:
        response = create_pipeline(url, api_key, name, desc, trigger)
    except APIException:
        render("ERROR in connecting to Environment!")
        raise typer.Exit(1)

    if not response.ok:
        _handle_pipeline_error(response, "pipeline-create")
        raise typer.Exit(1)

    data = render_single_dict(response.json())
    render("Created Pipeline Successfully! Details:")
    render(data, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")


@app.command("edit")
def pipeline_edit(
    pipeline_id: Optional[str] = typer.Option(None, "--pipeline-id", "-p", help="Pipeline ID"),
    name: Optional[str] = typer.Option(None, "--name", "-n", help="New pipeline name"),
    desc: Optional[str] = typer.Option(None, "--desc", help="New pipeline description"),
):
    """Edit a pipeline."""
    validate_creds()
    db = DBWrapper(state.creds)
    environment = db.get_configure().get("environment")
    enterprise_id = db.get_enterprise_id()
    api_key = db.get_configure().get("api_key")

    if not pipeline_id:
        pipeline_id = typer.prompt("Enter the Pipeline ID")
    if name is None:
        name = typer.prompt("Change the name of the Pipeline", default="")
    if desc is None:
        desc = typer.prompt("Change the description [optional]", default="")

    trigger = _get_trigger_interactively()

    if not name and not desc and not trigger:
        render("No changes requested. Exiting.")
        return

    url = get_pipeline_url(environment, enterprise_id, pipeline_id=pipeline_id)
    try:
        response = edit_pipeline(url, api_key, name, desc, trigger)
    except APIException:
        render("ERROR in connecting to Environment!")
        raise typer.Exit(1)

    if not response.ok:
        _handle_pipeline_error(response, "pipeline-edit")
        raise typer.Exit(1)

    data = render_single_dict(response.json())
    render("Edited Pipeline Successfully! Details:")
    render(data, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")


@app.command("show")
def pipeline_show(
    pipeline_id: Optional[str] = typer.Option(None, "--pipeline-id", "-p", help="Pipeline ID"),
):
    """List or fetch pipeline(s)."""
    validate_creds()
    db = DBWrapper(state.creds)
    environment = db.get_configure().get("environment")
    enterprise_id = db.get_enterprise_id()
    api_key = db.get_configure().get("api_key")

    url = get_pipeline_url(environment, enterprise_id, pipeline_id=pipeline_id)
    try:
        response = fetch_pipelines(url, api_key) if pipeline_id else list_pipelines(url, api_key)
    except APIException:
        render("ERROR in connecting to Environment!")
        raise typer.Exit(1)

    if not response.ok:
        _handle_pipeline_error(response, "pipeline-show")
        raise typer.Exit(1)

    data = [response.json()] if pipeline_id else (response.json().get("results") or [])

    render_data = []
    for pipeline in data:
        trigger_name = trigger_app = None
        trigger = pipeline.get("trigger")
        if trigger:
            trigger_name = trigger.get("trigger_event")
            pre = trigger.get("pre_conditions")
            trigger_app = pre.get("application_name") if pre else None
        render_data.append({
            "ID": pipeline.get("id"),
            "NAME": pipeline.get("name"),
            "DESCRIPTION": pipeline.get("description"),
            "STAGES": len(pipeline.get("stages", [])),
            "VERSION": pipeline.get("version"),
            "TRIGGER": trigger_name,
            "TRIGGER-APP": trigger_app,
        })

    render("Listing Pipeline! Details:")
    render(render_data, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")


@app.command("remove")
def pipeline_remove(
    pipeline_id: Optional[str] = typer.Option(None, "--pipeline-id", "-p", help="Pipeline ID"),
):
    """Remove a pipeline."""
    validate_creds()
    db = DBWrapper(state.creds)
    environment = db.get_configure().get("environment")
    enterprise_id = db.get_enterprise_id()
    api_key = db.get_configure().get("api_key")

    if not pipeline_id:
        pipeline_id = typer.prompt("Enter the Pipeline ID")

    url = get_pipeline_url(environment, enterprise_id, pipeline_id=pipeline_id)
    try:
        response = delete_api(url, api_key)
    except APIException:
        render("ERROR in connecting to Environment!")
        raise typer.Exit(1)

    if not response.ok:
        _handle_pipeline_error(response, "pipeline-remove")
        raise typer.Exit(1)

    render("Removed Pipeline Successfully!")


# ---------------------------------------------------------------------------
# Internal
# ---------------------------------------------------------------------------

def _handle_pipeline_error(response, tag: str):
    state.log.debug(f"[{tag}] Response not OK. Status: {response.status_code}")
    if response.status_code == 400:
        errors = response.json().get("meta", {}).get("non_field_errors")
        if errors:
            state.log.error(f"Validation Error: {errors}")
    elif response.status_code == 404:
        state.log.error("Pipeline URL not found!")
    elif response.status_code == 500:
        state.log.error(f"Internal Server Error! {response.json()}")
