"""
Enterprise commands — replaces esper/controllers/enterprise/enterprise.py.
`espercli enterprise show/update`
"""
from typing import Optional

import typer
from esperclient import EnterpriseUpdateV1
from esperclient.rest import ApiException

from esper.cli.output import render
from esper.cli.state import state, validate_creds, parse_error_message
from esper.controllers.enums import OutputFormat
from esper.ext.api_client import APIClient
from esper.ext.db_wrapper import DBWrapper

app = typer.Typer(help="Enterprise commands")


# ---------------------------------------------------------------------------
# Shared helpers
# ---------------------------------------------------------------------------

def _enterprise_basic_response(enterprise, fmt=OutputFormat.TABULATED):
    registered_name = getattr(enterprise, "registered_name", None)
    address = getattr(enterprise, "registered_address", None)
    location = getattr(enterprise, "location", None)
    zipcode = getattr(enterprise, "zipcode", None)
    email = getattr(enterprise, "contact_email", None)
    contact_person = getattr(enterprise, "contact_person", None)
    contact_number = getattr(enterprise, "contact_number", None)

    if fmt == OutputFormat.TABULATED:
        title, details = "TITLE", "DETAILS"
        return [
            {title: "Enterprise Id", details: enterprise.id},
            {title: "Name", details: enterprise.name},
            {title: "Registered Name", details: registered_name},
            {title: "Address", details: address},
            {title: "Location", details: location},
            {title: "Zip Code", details: zipcode},
            {title: "Email", details: email},
            {title: "Contact Person", details: contact_person},
            {title: "Contact Number", details: contact_number},
        ]
    return {
        "Enterprise Id": enterprise.id, "Name": enterprise.name,
        "Registered Name": registered_name, "Address": address,
        "Location": location, "Zip Code": zipcode, "Email": email,
        "Contact Person": contact_person, "Contact Number": contact_number,
    }


# ---------------------------------------------------------------------------
# Commands
# ---------------------------------------------------------------------------

@app.command("show")
def enterprise_show(
    json_output: bool = typer.Option(False, "--json", "-j", help="Render result in JSON format"),
):
    """Show enterprise details."""
    validate_creds()
    db = DBWrapper(state.creds)
    enterprise_client = APIClient(db.get_configure()).get_enterprise_api_client()
    enterprise_id = db.get_enterprise_id()

    try:
        response = enterprise_client.get_enterprise(enterprise_id)
    except ApiException as e:
        state.log.error(f"[enterprise-show] Failed to show enterprise: {e}")
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)

    if not json_output:
        render(_enterprise_basic_response(response), format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
    else:
        render(_enterprise_basic_response(response, OutputFormat.JSON), format=OutputFormat.JSON.value)


@app.command("update")
def enterprise_update(
    name: Optional[str] = typer.Option(None, "--name", "-n", help="Enterprise name"),
    display_name: Optional[str] = typer.Option(None, "--dispname", "-dn", help="Display name"),
    registered_name: Optional[str] = typer.Option(None, "--regname", "-rn", help="Registered name"),
    address: Optional[str] = typer.Option(None, "--address", "-a", help="Enterprise address"),
    location: Optional[str] = typer.Option(None, "--location", "-l", help="Enterprise location"),
    zipcode: Optional[str] = typer.Option(None, "--zipcode", "-z", help="Zip code"),
    email: Optional[str] = typer.Option(None, "--email", "-e", help="Contact email"),
    contact_person: Optional[str] = typer.Option(None, "--person", "-p", help="Contact person"),
    contact_number: Optional[str] = typer.Option(None, "--number", "-cn", help="Contact number"),
    json_output: bool = typer.Option(False, "--json", "-j", help="Render result in JSON format"),
):
    """Update enterprise details."""
    validate_creds()
    db = DBWrapper(state.creds)
    enterprise_client = APIClient(db.get_configure()).get_enterprise_api_client()
    enterprise_id = db.get_enterprise_id()

    update_dict = {}
    if name:
        update_dict["name"] = name
    if registered_name:
        update_dict["registered_name"] = registered_name
    if address:
        update_dict["registered_address"] = address
    if location:
        update_dict["location"] = location
    if zipcode:
        update_dict["zipcode"] = zipcode
    if email:
        update_dict["contact_email"] = email
    if contact_person:
        update_dict["contact_person"] = contact_person
    if contact_number:
        update_dict["contact_number"] = contact_number

    try:
        response = enterprise_client.partial_update_enterprise(enterprise_id, update_dict)
    except ApiException as e:
        state.log.error(f"[enterprise-update] Failed to update enterprise: {e}")
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)

    if not json_output:
        render(_enterprise_basic_response(response), format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
    else:
        render(_enterprise_basic_response(response, OutputFormat.JSON), format=OutputFormat.JSON.value)
