"""
Configure command — replaces esper/controllers/configure.py.
Embedded at the root level: `espercli configure`
"""
import json
from http import HTTPStatus
from typing import Optional

import typer
from esperclient.rest import ApiException

from esper.cli.output import render
from esper.cli.state import state
from esper.controllers.enums import OutputFormat
from esper.ext.api_client import APIClient
from esper.ext.db_wrapper import DBWrapper

app = typer.Typer(help="Configure credentials for the Esper.io API Service")


@app.callback(invoke_without_command=True)
def configure(
    set_creds: bool = typer.Option(
        False, "--set", "-s", help="Create or update credentials for the Esper.io API Service"
    ),
    list_creds: bool = typer.Option(
        False, "--list", "-l", help="List stored credentials"
    ),
    json_output: bool = typer.Option(
        False, "--json", "-j", help="Render result in JSON format"
    ),
):
    """Configure (or display) Esper.io API credentials."""
    db = DBWrapper(state.creds)
    credentials = db.get_configure()

    # ------------------------------------------------------------------ set
    if set_creds or not credentials:
        environment = typer.prompt("Environment name")
        api_key = typer.prompt("Esper API Key", hide_input=True)

        token_client = APIClient(
            {"api_key": api_key, "environment": environment}
        ).get_token_api_client()
        try:
            response = token_client.get_token_info()
        except ApiException as e:
            state.log.error(f"[configure] Failed to get token info: {e}")
            if e.status == HTTPStatus.UNAUTHORIZED:
                render("You are not authorized, invalid API Key.")
            else:
                error_message = (
                    json.loads(e.body).get("message")
                    if e.body and json.loads(e.body).get("message")
                    else e.reason
                )
                render(f"ERROR: {error_message}")
            raise typer.Exit(1)

        if response:
            enterprise_id = response.enterprise
        else:
            state.log.info("[configure] API key is not associated with any enterprise.")
            render("API key is not associated with any enterprise.")
            raise typer.Exit(1)

        credentials = {
            "environment": environment,
            "api_key": api_key,
            "enterprise_id": enterprise_id,
        }
        db.set_configure(credentials)

    # ----------------------------------------------------------------- list
    if list_creds or credentials:
        if not credentials:
            render("No credentials stored. Run `espercli configure --set` to add them.")
            raise typer.Exit(1)

        if not json_output:
            title, details = "TITLE", "DETAILS"
            renderable = [
                {title: "environment", details: credentials.get("environment")},
                {title: "api_key", details: credentials.get("api_key")},
            ]
            render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
        else:
            renderable = {
                "environment": credentials.get("environment"),
                "api_key": credentials.get("api_key"),
            }
            render(renderable, format=OutputFormat.JSON.value)
