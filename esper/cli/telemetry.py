"""
Telemetry commands — replaces esper/controllers/telemetry/telemetry.py.
`espercli telemetry get-data`
"""
from datetime import datetime, timedelta
from typing import Optional

import requests
import typer
from esperclient.rest import ApiException

from esper.cli.completions import device_name_complete, period_complete, statistic_complete
from esper.cli.output import render
from esper.cli.state import state, validate_creds, parse_error_message
from esper.controllers.enums import OutputFormat
from esper.ext.api_client import APIClient
from esper.ext.db_wrapper import DBWrapper
from esper.ext.telemetry_api import get_telemetry_url

app = typer.Typer(help="Telemetry commands")


@app.command("get-data")
def telemetry_get_data(
    device_name: Optional[str] = typer.Option(
        None, "--device", "-d", help="Device name",
        shell_complete=device_name_complete,
    ),
    metric: Optional[str] = typer.Option(None, "--metric", "-m", help="Metric name ({category}-{metric})"),
    from_time: Optional[str] = typer.Option(
        None, "--from", "-f", help="Start date-time (ISO format)"
    ),
    period: str = typer.Option(
        "hour", "--period", "-p", help="Period: hour, month, or day",
        shell_complete=period_complete,
    ),
    statistic: str = typer.Option(
        "avg", "--statistic", "-s", help="Statistic: avg, sum, or count",
        shell_complete=statistic_complete,
    ),
    last: Optional[str] = typer.Option(
        None, "--last", "-l", help="Relative lookback (n hours/days based on period)"
    ),
    to_time: Optional[str] = typer.Option(
        None, "--to", "-t", help="End date-time (ISO format)"
    ),
):
    """Get telemetry data for a device."""
    validate_creds()
    db = DBWrapper(state.creds)
    environment = db.get_configure().get("environment")
    enterprise_id = db.get_enterprise_id()
    api_key = db.get_configure().get("api_key")
    device_client = APIClient(db.get_configure()).get_device_api_client()

    if not device_name:
        render("No device specified. Use the --device option to specify a device")
        raise typer.Exit(1)

    if not metric:
        render("No metric specified. Use the --metric option (format: {category}-{metric})")
        raise typer.Exit(1)

    # Resolve device ID
    try:
        response = device_client.get_all_devices(enterprise_id, limit=1, offset=0, name=device_name)
        if not response.results:
            render(f"Device does not exist with name {device_name}")
            raise typer.Exit(1)
        device_id = response.results[0].id
    except ApiException as e:
        state.log.error(f"[telemetry-get-data] Failed to fetch device {device_name}: {e}")
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)

    # Compute from/to timestamps
    now = datetime.now()
    computed_from = datetime.fromisoformat(from_time) if from_time else (now - timedelta(days=2))
    computed_to = datetime.fromisoformat(to_time) if to_time else now

    if last:
        n = int(last)
        if period == "hour":
            computed_from = now - timedelta(hours=n)
        elif period == "day":
            computed_from = now - timedelta(days=n)
        else:
            computed_from = now.replace(month=now.month - n)
        computed_to = now

    def _fmt_dt(dt) -> str:
        s = str(dt).replace(" ", "T")
        if "." not in s:
            s += ".0000Z"
        elif "Z" not in s:
            s += "Z"
        return s

    from_str = _fmt_dt(computed_from)
    to_str = _fmt_dt(computed_to)

    if "-" not in metric:
        render("ERROR: Metric must be of format {category}-{metric name}")
        raise typer.Exit(1)

    category, metric_name = metric.split("-", 1)

    url = get_telemetry_url(
        environment, enterprise_id, device_id,
        category, metric_name, from_str, to_str, period, statistic,
    )

    api_response = requests.get(url, headers={"Authorization": f"Bearer {api_key}"}, timeout=30)
    response_json = api_response.json()

    if api_response.status_code != 200:
        if response_json.get("meta", {}).get("non_field_errors"):
            render(f"ERRORS: {response_json['meta']['non_field_errors']}")
        elif response_json.get("errors"):
            render(f"ERRORS: {response_json['errors']}")
        else:
            render("ERROR: Unknown error occurred")
        raise typer.Exit(1)

    data = response_json.get("data", {})
    render_data = [{"Time": d["x"], "Value": d["y"]} for d in data]

    if not render_data:
        render(f"No telemetry data for device {device_name} found between time range")
        raise typer.Exit(0)

    render(f"Telemetry data for device {device_name}")
    render(render_data, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
