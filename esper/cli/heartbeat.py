# Heartbeat commands — last check-in time and online/offline status per device.
from concurrent.futures import ThreadPoolExecutor, as_completed
from datetime import datetime, timezone
from typing import Optional

import typer
from esperclient.rest import ApiException
from rich.text import Text

from esper.cli.completions import device_name_complete
from esper.cli.output import render
from esper.cli.state import state, validate_creds, parse_error_message
from esper.controllers.enums import OutputFormat
from esper.ext.api_client import APIClient
from esper.ext.api_rest import api_get
from esper.ext.db_wrapper import DBWrapper

app = typer.Typer(help="Device heartbeat (last check-in) commands")


def _ago(ts: str) -> str:
    """Return a human-readable 'X days/hours ago' string from an ISO timestamp."""
    try:
        dt = datetime.fromisoformat(ts.replace("Z", "+00:00"))
        delta = datetime.now(timezone.utc) - dt
        days = delta.days
        hours, rem = divmod(delta.seconds, 3600)
        minutes = rem // 60
        if days > 0:
            return f"{days}d ago"
        if hours > 0:
            return f"{hours}h ago"
        return f"{minutes}m ago"
    except Exception:
        return ts


def _status_label(status: int) -> str:
    return "ONLINE" if status == 1 else "OFFLINE"


def _fetch_heartbeat(environment, api_key, eid, device_id):
    return api_get(environment, api_key, f"/device/v0/heartbeat/{device_id}/")


@app.command("list")
def heartbeat_list(
    as_json: bool = typer.Option(False, "--json", "-j", help="Output as JSON"),
) -> None:
    """Show last heartbeat for all devices, sorted by timestamp descending."""
    validate_creds()
    db = DBWrapper(state.creds)
    cfg = db.get_configure()
    environment = cfg.get("environment")
    api_key = cfg.get("api_key")
    eid = db.get_enterprise_id()
    device_client = APIClient(cfg).get_device_api_client()

    try:
        devices = []
        page_size, offset = 100, 0
        while True:
            page = device_client.get_all_devices(eid, limit=page_size, offset=offset)
            devices.extend(page.results or [])
            if len(devices) >= (page.count or 0) or not page.results:
                break
            offset += page_size
    except ApiException as e:
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)

    if not devices:
        render("No devices found.")
        raise typer.Exit(0)

    device_map = {str(d.id): (d.alias_name or d.device_name) for d in devices}

    results = []
    with ThreadPoolExecutor(max_workers=10) as pool:
        futures = {
            pool.submit(_fetch_heartbeat, environment, api_key, eid, d.id): str(d.id)
            for d in devices
        }
        for fut in as_completed(futures):
            device_id = futures[fut]
            try:
                data = fut.result()
                content = data.get("content", {})
                results.append({
                    "name": device_map.get(device_id, device_id),
                    "device_id": device_id,
                    "status": content.get("status", 0),
                    "timestamp": content.get("timestamp", ""),
                })
            except Exception:
                results.append({
                    "name": device_map.get(device_id, device_id),
                    "device_id": device_id,
                    "status": 0,
                    "timestamp": "",
                })

    results.sort(key=lambda r: r["timestamp"], reverse=True)

    if as_json:
        render(results, format=OutputFormat.JSON.value)
        return

    rows = [
        {
            "NAME": r["name"],
            "STATUS": _status_label(r["status"]),
            "LAST SEEN": r["timestamp"],
            "AGO": _ago(r["timestamp"]) if r["timestamp"] else "—",
        }
        for r in results
    ]
    render(rows, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")


@app.command("show")
def heartbeat_show(
    device_name: Optional[str] = typer.Option(
        None, "--device", "-d", help="Device name",
        shell_complete=device_name_complete,
    ),
    as_json: bool = typer.Option(False, "--json", "-j", help="Output as JSON"),
) -> None:
    """Show heartbeat for a single device."""
    validate_creds()
    db = DBWrapper(state.creds)
    cfg = db.get_configure()
    environment = cfg.get("environment")
    api_key = cfg.get("api_key")
    eid = db.get_enterprise_id()
    device_client = APIClient(cfg).get_device_api_client()

    if not device_name:
        render("ERROR: Specify a device with --device NAME")
        raise typer.Exit(1)

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
        data = api_get(environment, api_key, f"/device/v0/heartbeat/{device_id}/")
    except Exception as e:
        render(f"ERROR: {e}")
        raise typer.Exit(1)

    content = data.get("content", {})

    if as_json:
        render(content, format=OutputFormat.JSON.value)
        return

    ts = content.get("timestamp", "")
    status = content.get("status", 0)
    rows = [
        {
            "NAME": device_name,
            "STATUS": _status_label(status),
            "LAST SEEN": ts,
            "AGO": _ago(ts) if ts else "—",
        }
    ]
    render(rows, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
