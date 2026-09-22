# Location commands — GPS coordinates for devices.
from typing import Optional
import re

import typer
from esperclient.rest import ApiException

from esper.cli.completions import device_name_complete
from esper.cli.output import render
from esper.cli.state import state, validate_creds, parse_error_message
from esper.controllers.enums import OutputFormat
from esper.ext.api_client import APIClient
from esper.ext.api_rest import api_get, api_get_all
from esper.ext.db_wrapper import DBWrapper

app = typer.Typer(help="Device location commands")

_DEVICE_ID_RE = re.compile(r"/device/([^/]+)/?$")


def _extract_device_id(url: str) -> Optional[str]:
    m = _DEVICE_ID_RE.search(url or "")
    return m.group(1) if m else None


def _build_device_map(device_client, eid: str) -> dict:
    """Return {device_id_str: name} for all devices (auto-paginated)."""
    from esper.cli.state import state as _state
    page_size = 100
    offset = 0
    mapping: dict = {}
    while True:
        try:
            resp = device_client.get_all_devices(eid, limit=page_size, offset=offset)
        except ApiException as e:
            _state.log.error(f"[location] Failed to fetch device list at offset {offset}: {e}")
            break
        for d in resp.results or []:
            mapping[str(d.id)] = d.alias_name or d.device_name
        if len(mapping) >= (resp.count or 0) or not resp.results:
            break
        offset += page_size
    return mapping


def _city_state(item: dict) -> str:
    parts = [p for p in [item.get("city"), item.get("state"), item.get("country")] if p]
    return ", ".join(parts) if parts else "—"


@app.command("list")
def location_list(
    as_json: bool = typer.Option(False, "--json", "-j", help="Output as JSON"),
) -> None:
    """Show GPS location for all devices."""
    validate_creds()
    db = DBWrapper(state.creds)
    cfg = db.get_configure()
    environment = cfg.get("environment")
    api_key = cfg.get("api_key")
    eid = db.get_enterprise_id()
    device_client = APIClient(cfg).get_device_api_client()

    try:
        items = api_get_all(environment, api_key, f"/v1/enterprise/{eid}/report/location/")
    except Exception as e:
        render(f"ERROR: {e}")
        raise typer.Exit(1)

    if not items:
        render("No location data found.")
        raise typer.Exit(0)

    device_map = _build_device_map(device_client, eid)

    if as_json:
        out = []
        for item in items:
            did = _extract_device_id(item.get("device", ""))
            out.append({
                "name": device_map.get(did, did or "unknown"),
                "latitude": item.get("latitude"),
                "longitude": item.get("longitude"),
                "city_state": _city_state(item),
                "last_updated_on": item.get("last_updated_on"),
            })
        render(out, format=OutputFormat.JSON.value)
        return

    rows = []
    for item in items:
        did = _extract_device_id(item.get("device", ""))
        rows.append({
            "NAME": device_map.get(did, did or "unknown"),
            "LAT": item.get("latitude", "—"),
            "LON": item.get("longitude", "—"),
            "CITY/STATE": _city_state(item),
            "LAST UPDATED": item.get("last_updated_on", "—"),
        })
    render(rows, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")


@app.command("show")
def location_show(
    device_name: str = typer.Option(
        ..., "--device", "-d", help="Device name",
        shell_complete=device_name_complete,
    ),
    as_json: bool = typer.Option(False, "--json", "-j", help="Output as JSON"),
) -> None:
    """Show GPS location for one device."""
    validate_creds()
    db = DBWrapper(state.creds)
    cfg = db.get_configure()
    environment = cfg.get("environment")
    api_key = cfg.get("api_key")
    eid = db.get_enterprise_id()
    device_client = APIClient(cfg).get_device_api_client()

    try:
        resp = device_client.get_all_devices(eid, limit=1, offset=0, name=device_name)
        if not resp.results:
            render(f"ERROR: Device not found: {device_name}")
            raise typer.Exit(1)
        device_id = str(resp.results[0].id)
    except ApiException as e:
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)

    try:
        items = api_get_all(environment, api_key, f"/v1/enterprise/{eid}/report/location/")
    except Exception as e:
        render(f"ERROR: {e}")
        raise typer.Exit(1)

    match = None
    for item in items:
        did = _extract_device_id(item.get("device", ""))
        if did == device_id:
            match = item
            break

    if not match:
        render(f"No location data found for device {device_name}.")
        raise typer.Exit(0)

    if as_json:
        render({
            "name": device_name,
            "latitude": match.get("latitude"),
            "longitude": match.get("longitude"),
            "city_state": _city_state(match),
            "last_updated_on": match.get("last_updated_on"),
        }, format=OutputFormat.JSON.value)
        return

    rows = [{
        "NAME": device_name,
        "LAT": match.get("latitude", "—"),
        "LON": match.get("longitude", "—"),
        "CITY/STATE": _city_state(match),
        "LAST UPDATED": match.get("last_updated_on", "—"),
    }]
    render(rows, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
