"""
Device status commands — replaces esper/controllers/device/status.py.
`espercli status latest`
"""
from ast import literal_eval
from typing import Optional

import typer
from esperclient.rest import ApiException

from esper.cli.output import render
from esper.cli.state import state, validate_creds, parse_error_message
from esper.controllers.enums import OutputFormat
from esper.ext.api_client import APIClient
from esper.ext.db_wrapper import DBWrapper

app = typer.Typer(help="Device status commands")


@app.command("latest")
def status_latest(
    device: Optional[str] = typer.Option(None, "--device", "-d", help="Device name"),
    json_output: bool = typer.Option(False, "--json", "-j", help="Render result in JSON format"),
):
    """Show the latest device status."""
    validate_creds()
    db = DBWrapper(state.creds)
    device_client = APIClient(db.get_configure()).get_device_api_client()
    enterprise_id = db.get_enterprise_id()

    if device:
        try:
            search_response = device_client.get_all_devices(
                enterprise_id, limit=1, offset=0, name=device
            )
            if not search_response.results:
                render(f"Device does not exist with name {device}")
                raise typer.Exit(1)
            device_id = search_response.results[0].id
        except ApiException as e:
            state.log.error(f"[status-latest] Failed to list devices: {e}")
            render(f"ERROR: {parse_error_message(e)}")
            raise typer.Exit(1)
    else:
        dev = db.get_device()
        if not dev or not dev.get("id"):
            render("There is no active device.")
            raise typer.Exit(1)
        device_id = dev.get("id")

    try:
        response = device_client.get_device_event(enterprise_id, device_id, latest_event=1)
    except ApiException as e:
        state.log.error(f"[status-latest] Failed to get device status: {e}")
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)

    battery_level = battery_temp = data_download = data_upload = None
    memory_storage = memory_ram = link_speed = signal_strength = None

    if response.results:
        data = literal_eval(response.results[0].data)
        pm = data.get("powerManagementEvent", {})
        bs = pm.get("batteryStatus", {})
        battery_level = bs.get("batteryLevel")
        battery_temp = bs.get("batteryTemperature")

        du = data.get("dataUsageStats", {})
        data_download = du.get("totalDataDownload")
        data_upload = du.get("totalDataUpload")

        mem_events = data.get("memoryEvents", [])
        if len(mem_events) > 1:
            memory_storage = mem_events[1].get("countInMb")
        if len(mem_events) > 0:
            memory_ram = mem_events[0].get("countInMb")

        net = data.get("networkEvent", {})
        wifi = net.get("wifiNetworkInfo", {})
        link_speed = wifi.get("linkSpeed")
        signal_strength = wifi.get("signalStrength")

    if not json_output:
        title, details = "TITLE", "DETAILS"
        renderable = [
            {title: "battery_level", details: battery_level},
            {title: "battery_temperature", details: battery_temp},
            {title: "data_download", details: data_download},
            {title: "data_upload", details: data_upload},
            {title: "memory_storage", details: memory_storage},
            {title: "memory_ram", details: memory_ram},
            {title: "link_speed", details: link_speed},
            {title: "signal_strength", details: signal_strength},
        ]
        render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
    else:
        renderable = {
            "battery_level": battery_level,
            "battery_temperature": battery_temp,
            "data_download": data_download,
            "data_upload": data_upload,
            "memory_storage": memory_storage,
            "memory_ram": memory_ram,
            "link_speed": link_speed,
            "signal_strength": signal_strength,
        }
        render(renderable, format=OutputFormat.JSON.value)
