"""
Pipeline Execute commands — nested under `pipeline`.
`espercli pipeline execute start/stop/continue/terminate/show`
"""
from typing import Optional

import typer

from esper.cli.output import render
from esper.cli.state import state, validate_creds
from esper.controllers.enums import OutputFormat
from esper.ext.db_wrapper import DBWrapper
from esper.ext.pipeline_api import (
    execute_pipeline, list_execute_pipeline, get_pipeline_execute_url,
    APIException, render_single_dict,
)

app = typer.Typer(help="Pipeline Execute commands")


# ---------------------------------------------------------------------------
# Shared helpers
# ---------------------------------------------------------------------------

def _handle_execute_error(response, tag: str):
    state.log.debug(f"[{tag}] Response not OK. Status: {response.status_code}")
    if response.status_code == 400:
        if isinstance(response.json(), dict):
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
        else:
            state.log.error(f"Validation Errors -> {response.json()}")
    elif response.status_code == 404:
        state.log.error("Pipeline URL not found!")
    elif response.status_code == 500:
        state.log.error(f"Internal Server Error! {response.json()}")


# ---------------------------------------------------------------------------
# Commands
# ---------------------------------------------------------------------------

@app.command("start")
def execute_start(
    pipeline_id: Optional[str] = typer.Option(None, "--pipeline-id", "-p", help="Pipeline ID"),
):
    """Start a Pipeline Execution."""
    validate_creds()
    db = DBWrapper(state.creds)
    environment = db.get_configure().get("environment")
    enterprise_id = db.get_enterprise_id()
    api_key = db.get_configure().get("api_key")

    if not pipeline_id:
        pipeline_id = typer.prompt("Enter the Pipeline ID")

    url = get_pipeline_execute_url(environment, enterprise_id, pipeline_id)
    try:
        response = execute_pipeline(url, api_key)
    except APIException:
        render("ERROR in connecting to Environment!")
        raise typer.Exit(1)

    if not response.ok:
        _handle_execute_error(response, "execute-start")
        raise typer.Exit(1)

    data = render_single_dict(response.json())
    render("Pipeline execution started! Details:")
    render(data, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")


@app.command("stop")
def execute_stop(
    pipeline_id: Optional[str] = typer.Option(None, "--pipeline-id", "-p", help="Pipeline ID"),
    execution_id: Optional[str] = typer.Option(None, "--execution-id", "-e", help="Execution ID"),
    reason: Optional[str] = typer.Option(None, "--reason", "-r", help="Reason for stopping"),
):
    """Stop a Pipeline Execution."""
    validate_creds()
    db = DBWrapper(state.creds)
    environment = db.get_configure().get("environment")
    enterprise_id = db.get_enterprise_id()
    api_key = db.get_configure().get("api_key")

    if not pipeline_id:
        pipeline_id = typer.prompt("Enter the Pipeline ID")
    if not execution_id:
        execution_id = typer.prompt("Enter the Execution ID")
    if not reason:
        reason = typer.prompt("Why do you want to stop this Execution?")

    url = get_pipeline_execute_url(environment, enterprise_id, pipeline_id, execution_id, "stop")
    try:
        response = execute_pipeline(url, api_key, {"reason": reason})
    except APIException:
        render("ERROR in connecting to Environment!")
        raise typer.Exit(1)

    if not response.ok:
        _handle_execute_error(response, "execute-stop")
        raise typer.Exit(1)

    data = render_single_dict(response.json())
    render("Pipeline execution stopped! Details:")
    render(data, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")


@app.command("continue")
def execute_continue(
    pipeline_id: Optional[str] = typer.Option(None, "--pipeline-id", "-p", help="Pipeline ID"),
    execution_id: Optional[str] = typer.Option(None, "--execution-id", "-e", help="Execution ID"),
):
    """Continue a paused Pipeline Execution."""
    validate_creds()
    db = DBWrapper(state.creds)
    environment = db.get_configure().get("environment")
    enterprise_id = db.get_enterprise_id()
    api_key = db.get_configure().get("api_key")

    if not pipeline_id:
        pipeline_id = typer.prompt("Enter the Pipeline ID")
    if not execution_id:
        execution_id = typer.prompt("Enter the Execution ID")

    url = get_pipeline_execute_url(environment, enterprise_id, pipeline_id, execution_id, "continue")
    try:
        response = execute_pipeline(url, api_key)
    except APIException:
        render("ERROR in connecting to Environment!")
        raise typer.Exit(1)

    if not response.ok:
        _handle_execute_error(response, "execute-continue")
        raise typer.Exit(1)

    data = render_single_dict(response.json())
    render("Pipeline execution continuing! Details:")
    render(data, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")


@app.command("terminate")
def execute_terminate(
    pipeline_id: Optional[str] = typer.Option(None, "--pipeline-id", "-p", help="Pipeline ID"),
    execution_id: Optional[str] = typer.Option(None, "--execution-id", "-e", help="Execution ID"),
    reason: Optional[str] = typer.Option(None, "--reason", "-r", help="Reason for termination"),
):
    """Terminate a Pipeline Execution."""
    validate_creds()
    db = DBWrapper(state.creds)
    environment = db.get_configure().get("environment")
    enterprise_id = db.get_enterprise_id()
    api_key = db.get_configure().get("api_key")

    if not pipeline_id:
        pipeline_id = typer.prompt("Enter the Pipeline ID")
    if not execution_id:
        execution_id = typer.prompt("Enter the Execution ID")
    if not reason:
        reason = typer.prompt("Why do you want to terminate this Execution?")

    url = get_pipeline_execute_url(environment, enterprise_id, pipeline_id, execution_id, "terminate")
    try:
        response = execute_pipeline(url, api_key, {"reason": reason})
    except APIException:
        render("ERROR in connecting to Environment!")
        raise typer.Exit(1)

    if not response.ok:
        _handle_execute_error(response, "execute-terminate")
        raise typer.Exit(1)

    data = render_single_dict(response.json())
    render("Pipeline execution Terminated! Details:")
    render(data, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")


@app.command("show")
def execute_show(
    pipeline_id: Optional[str] = typer.Option(None, "--pipeline-id", "-p", help="Pipeline ID"),
):
    """List all Executions for a Pipeline."""
    validate_creds()
    db = DBWrapper(state.creds)
    environment = db.get_configure().get("environment")
    enterprise_id = db.get_enterprise_id()
    api_key = db.get_configure().get("api_key")

    if not pipeline_id:
        pipeline_id = typer.prompt("Enter the Pipeline ID")

    url = get_pipeline_execute_url(environment, enterprise_id, pipeline_id)
    try:
        response = list_execute_pipeline(url, api_key)
    except APIException:
        render("ERROR in connecting to Environment!")
        raise typer.Exit(1)

    if not response.ok:
        _handle_execute_error(response, "execute-show")
        raise typer.Exit(1)

    data = response.json().get("results") or []
    render_data = [
        {
            "ID": ex.get("id"),
            "NAME": ex.get("name"),
            "DESCRIPTION": ex.get("description"),
            "STATE": ex.get("state"),
            "STATUS": ex.get("status"),
            "REASON": ex.get("reason"),
        }
        for ex in data
    ]

    render("Listing Executions for the Pipeline! Details:")
    render(render_data, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
