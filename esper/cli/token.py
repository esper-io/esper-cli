"""
Token commands — replaces esper/controllers/token/token.py.
`espercli token show/renew`
"""
from typing import Optional

import typer
from esperclient.rest import ApiException

from esper.cli.output import render
from esper.cli.state import state, validate_creds, parse_error_message
from esper.controllers.enums import OutputFormat
from esper.ext.api_client import APIClient
from esper.ext.db_wrapper import DBWrapper

app = typer.Typer(help="Token commands")


# ---------------------------------------------------------------------------
# Shared helpers
# ---------------------------------------------------------------------------

def _token_basic_response(token, fmt=OutputFormat.TABULATED):
    if fmt == OutputFormat.TABULATED:
        title, details = "TITLE", "DETAILS"
        return [
            {title: "Enterprise Id", details: token.enterprise},
            {title: "Token", details: token.token},
            {title: "Expires On", details: token.expires_on},
            {title: "Scope", details: token.scope},
            {title: "Created On", details: token.created_on},
            {title: "Updated On", details: token.updated_on},
        ]
    return {
        "Enterprise": token.enterprise,
        "Developer App": token.developer_app,
        "Token": token.token,
        "Expires On": token.expires_on,
        "Created On": str(token.created_on),
        "Updated On": str(token.updated_on),
    }


def _renew_token_basic_response(token, fmt=OutputFormat.TABULATED):
    enterprise_id = token.enterprise.split("[")[1].split("]")[0]
    developer_app = token.developerapp.split("(")[1].split(")")[0]
    if fmt == OutputFormat.TABULATED:
        title, details = "TITLE", "DETAILS"
        return [
            {title: "Id", details: token.id},
            {title: "User", details: token.user},
            {title: "Enterprise Id", details: enterprise_id},
            {title: "Developer App", details: developer_app},
            {title: "Token", details: token.token},
            {title: "Scope", details: token.scope},
            {title: "Created On", details: token.created_on},
            {title: "Updated On", details: token.updated_on},
            {title: "Expires On", details: token.expires_at},
        ]
    return {
        "Id": token.id,
        "User": token.user,
        "Enterprise Id": enterprise_id,
        "Developer App": developer_app,
        "Token": token.token,
        "Scope": token.scope,
        "Created On": str(token.created_on),
        "Updated On": str(token.updated_on),
        "Expires On": str(token.expires_at),
    }


# ---------------------------------------------------------------------------
# Commands
# ---------------------------------------------------------------------------

@app.command("show")
def token_show(
    json_output: bool = typer.Option(False, "--json", "-j", help="Render result in JSON format"),
):
    """Show token details."""
    validate_creds()
    db = DBWrapper(state.creds)
    token_client = APIClient(db.get_configure()).get_token_api_client()

    try:
        response = token_client.get_token_info()
    except ApiException as e:
        state.log.error(f"[token-show] Failed to show token details: {e}")
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)

    if not json_output:
        render(_token_basic_response(response), format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
    else:
        render(_token_basic_response(response, OutputFormat.JSON), format=OutputFormat.JSON.value)


@app.command("renew")
def token_renew(
    dev_app_id: Optional[str] = typer.Option(None, "--developerappid", "-d", help="Developer App ID"),
    token: Optional[str] = typer.Option(None, "--token", "-t", help="Token to renew"),
    json_output: bool = typer.Option(False, "--json", "-j", help="Render result in JSON format"),
):
    """Renew a token."""
    validate_creds()
    db = DBWrapper(state.creds)
    token_client = APIClient(db.get_configure()).get_token_api_client()
    enterprise_id = db.get_enterprise_id()

    if not token:
        render("There is no token given to renew.")
        raise typer.Exit(1)

    if not dev_app_id:
        render("DeveloperApp id is not given.")
        raise typer.Exit(1)

    try:
        response = token_client.renew_token(enterprise_id, dev_app_id, token)
    except ApiException as e:
        state.log.error(f"[token-renew] Failed to renew token: {e}")
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)

    if not json_output:
        render(_renew_token_basic_response(response), format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
    else:
        render(_renew_token_basic_response(response, OutputFormat.JSON), format=OutputFormat.JSON.value)
