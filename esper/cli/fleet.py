# Fleet overview commands — merges v1 and v2 status-metrics endpoints.
from typing import Optional

import typer
from rich.console import Console
from rich.panel import Panel
from rich.table import Table, box

from esper.cli.output import render
from esper.cli.state import state, validate_creds
from esper.controllers.enums import OutputFormat
from esper.ext.api_rest import api_get
from esper.ext.db_wrapper import DBWrapper

app = typer.Typer(help="Fleet overview commands")

console = Console()


@app.command("status")
def fleet_status(
    as_json: bool = typer.Option(False, "--json", "-j", help="Output as JSON"),
) -> None:
    """Show fleet health, risk, and last-seen breakdown."""
    validate_creds()
    db = DBWrapper(state.creds)
    cfg = db.get_configure()
    environment = cfg.get("environment")
    api_key = cfg.get("api_key")
    eid = db.get_enterprise_id()

    try:
        v1 = api_get(environment, api_key, f"/v1/enterprise/{eid}/report/status-metrics/")
        v2 = api_get(environment, api_key, f"/v2/enterprise/{eid}/report/status-metrics")
    except Exception as e:
        render(f"ERROR: {e}")
        raise typer.Exit(1)

    last_seen = v2.get("last_seen", {})

    if as_json:
        render({
            "health": {
                "active": v1.get("active"),
                "inactive": v1.get("in_active"),
                "total": v1.get("active", 0) + v1.get("in_active", 0),
                "low_battery": v1.get("low_battery"),
                "under_provisioning": v1.get("under_provisioning"),
            },
            "risk": {
                "low_risk": v1.get("low_risk"),
                "medium_risk": v1.get("medium_risk"),
                "high_risk": v1.get("high_risk"),
                "secure": v1.get("secure"),
                "total_devices_under_risk": v1.get("total_devices_under_risk"),
            },
            "last_seen": {
                "last_24h": last_seen.get("first_period"),
                "last_1_7d": last_seen.get("second_period"),
                "last_7_30d": last_seen.get("third_period"),
            },
        }, format=OutputFormat.JSON.value)
        return

    # Device health table
    health_tbl = Table(box=box.SIMPLE_HEAD, show_header=True, header_style="bold cyan",
                       show_edge=False, pad_edge=False)
    health_tbl.add_column("Metric", style="bold")
    health_tbl.add_column("Count", justify="right")

    total = v1.get("active", 0) + v1.get("in_active", 0)
    health_tbl.add_row("Active",             str(v1.get("active", 0)))
    health_tbl.add_row("Inactive",           str(v1.get("in_active", 0)))
    health_tbl.add_row("Total",              str(total))
    health_tbl.add_row("Low battery",        str(v1.get("low_battery", 0)))
    health_tbl.add_row("Under provisioning", str(v1.get("under_provisioning", 0)))

    # Risk table
    risk_tbl = Table(box=box.SIMPLE_HEAD, show_header=True, header_style="bold cyan",
                     show_edge=False, pad_edge=False)
    risk_tbl.add_column("Metric", style="bold")
    risk_tbl.add_column("Count", justify="right")

    risk_tbl.add_row("Low risk",               str(v1.get("low_risk", 0)))
    risk_tbl.add_row("Medium risk",            str(v1.get("medium_risk", 0)))
    risk_tbl.add_row("High risk",              str(v1.get("high_risk", 0)))
    risk_tbl.add_row("Secure",                 str(v1.get("secure", 0)))
    risk_tbl.add_row("Total devices at risk",  str(v1.get("total_devices_under_risk", 0)))

    # Last seen table
    seen_tbl = Table(box=box.SIMPLE_HEAD, show_header=True, header_style="bold cyan",
                     show_edge=False, pad_edge=False)
    seen_tbl.add_column("Window", style="bold")
    seen_tbl.add_column("Devices", justify="right")

    seen_tbl.add_row("Last 24 hours",  str(last_seen.get("first_period", 0)))
    seen_tbl.add_row("1 – 7 days ago", str(last_seen.get("second_period", 0)))
    seen_tbl.add_row("7 – 30 days ago", str(last_seen.get("third_period", 0)))

    console.print(Panel(health_tbl, title="[bold]Device Health[/bold]", border_style="blue", expand=False))
    console.print(Panel(risk_tbl,   title="[bold]Risk[/bold]",           border_style="yellow", expand=False))
    console.print(Panel(seen_tbl,   title="[bold]Last Seen[/bold]",      border_style="cyan", expand=False))
