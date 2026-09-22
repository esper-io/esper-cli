"""
CommandsV2 commands — replaces esper/controllers/commandsV2/commandsV2.py.
`espercli commandsv2 list/status/history/command`
"""
import json
from typing import Optional

import typer
from esperclient import V0CommandRequest
from esperclient.rest import ApiException

from esper.cli.completions import (
    command_enum_complete,
    command_type_complete,
    command_state_complete,
    device_name_complete,
    device_type_complete,
    schedule_type_complete,
    days_complete,
    time_type_complete,
)
from esper.cli.output import render
from esper.cli.state import state, validate_creds, parse_error_message
from esper.controllers.enums import (
    OutputFormat,
    CommandEnum,
    CommandState,
    CommandRequestTypeEnum,
    CommandDeviceTypeEnum,
    WeekDays,
)
from esper.ext.api_client import APIClient
from esper.ext.db_wrapper import DBWrapper

app = typer.Typer(help="Fire commands to devices or groups")


# ---------------------------------------------------------------------------
# Shared helpers
# ---------------------------------------------------------------------------

def _request_basic_response(request, fmt=OutputFormat.TABULATED):
    if request.schedule_args is None:
        schedule_args = None
    else:
        schedule_args = {
            "name": request.schedule_args.name,
            "start_datetime": str(request.schedule_args.start_datetime),
            "end_datetime": str(request.schedule_args.end_datetime),
            "time_type": request.schedule_args.time_type,
            "window_start_time": str(request.schedule_args.window_start_time),
            "window_end_time": str(request.schedule_args.window_end_time),
            "days": request.schedule_args.days,
        }

    command_args = {
        k[1:]: v for k, v in request.command_args.__dict__.items() if v is not None
    }

    state_val = request.status[0].state if request.status else None
    issued_by = request.issued_by.split("'")[-2]

    if fmt == OutputFormat.TABULATED:
        title, details = "TITLE", "DETAILS"
        return [
            {title: "Id", details: request.id},
            {title: "Command", details: request.command},
            {title: "Command Args", details: command_args},
            {title: "Command Type", details: request.command_type},
            {title: "Devices", details: request.devices},
            {title: "Groups", details: request.groups},
            {title: "Device Type", details: request.device_type},
            {title: "Status", details: state_val},
            {title: "Issued by", details: issued_by},
            {title: "Schedule", details: request.schedule},
            {title: "Schedule Args", details: schedule_args},
            {title: "Created On", details: request.created_on},
        ]
    return {
        "id": request.id,
        "command": request.command,
        "command_type": request.command_type,
        "command_args": command_args,
        "devices": request.devices,
        "groups": request.groups,
        "device_type": request.device_type,
        "status": str(request.status),
        "issued_by": request.issued_by,
        "schedule": request.schedule,
        "schedule_args": schedule_args,
        "created_on": str(request.created_on),
    }


def _render_status_response(status_response, limit: int, json_output: bool):
    render(f"Total Number of Statuses: {status_response.count}")
    if not json_output:
        label = {"id": "STATUS ID", "device": "DEVICE ID", "state": "STATE", "reason": "REASON"}
        statuses = [
            {
                label["id"]: s.id,
                label["device"]: s.device.split("/")[-2],
                label["state"]: s.state,
                label["reason"]: s.reason,
            }
            for s in status_response.results[:limit]
        ]
        render(statuses, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
    else:
        statuses = [
            {
                "id": s.id,
                "request_id": s.request,
                "device_id": s.device.split("/")[-2],
                "state": s.state,
                "reason": s.reason,
                "created_on": str(s.created_on),
                "updated_on": str(s.updated_on),
            }
            for s in status_response.results[:limit]
        ]
        render(statuses, format=OutputFormat.JSON.value)


# ---------------------------------------------------------------------------
# Commands
# ---------------------------------------------------------------------------

@app.command("list")
def commandsv2_list(
    command_type: Optional[str] = typer.Option(
        None, "--command-type", "-ct",
        help=f"Filter by command type ({', '.join(CommandRequestTypeEnum.choice_list_lower())})",
        shell_complete=command_type_complete,
    ),
    device: Optional[str] = typer.Option(
        None, "--device", "-d", help="Filter by device name",
        shell_complete=device_name_complete,
    ),
    device_type: Optional[str] = typer.Option(
        None, "--device-type", "-dt",
        help=f"Filter by device type ({', '.join(CommandDeviceTypeEnum.choice_list_lower())})",
        shell_complete=device_type_complete,
    ),
    command: Optional[str] = typer.Option(
        None, "--command", "-c",
        help=f"Filter by command name ({', '.join(CommandEnum.choice_list_lower())})",
        shell_complete=command_enum_complete,
    ),
    limit: int = typer.Option(10, "--limit", "-l", help="Number of results"),
    json_output: bool = typer.Option(False, "--json", "-j", help="Render result in JSON format"),
):
    """List command requests."""
    validate_creds()
    db = DBWrapper(state.creds)
    device_client = APIClient(db.get_configure()).get_device_api_client()
    commandsv2_client = APIClient(db.get_configure()).get_commandsV2_api_client()
    enterprise_id = db.get_enterprise_id()

    kwargs = {}
    if command_type:
        kwargs["command_type"] = command_type.upper()

    if device:
        try:
            search_response = device_client.get_all_devices(enterprise_id, limit=1, offset=0, name=device)
            if not search_response.results:
                render(f"Device does not exist with name {device}")
                raise typer.Exit(1)
            kwargs["devices"] = search_response.results[0].id
        except ApiException as e:
            state.log.error(f"[commandsv2-list] Failed to list devices: {e}")
            render(f"ERROR: {parse_error_message(e)}")
            raise typer.Exit(1)

    if device_type:
        kwargs["device_type"] = device_type
    if command:
        kwargs["command"] = command.upper()

    try:
        response = commandsv2_client.list_command_request(enterprise_id, **kwargs)
    except ApiException as e:
        state.log.error(f"[commandsv2-list] Failed to list command requests: {e}")
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)

    render(f"Total Number of Command Requests: {response.count}")
    if not json_output:
        label = {"id": "REQUEST ID", "command": "COMMAND", "issued_by": "ISSUED BY",
                 "command_type": "COMMAND TYPE", "created_on": "CREATED ON"}
        commandreqs = []
        for req in response.results[:limit]:
            issued_by_raw = req.issued_by.replace("'", '"')
            try:
                issued_by = json.loads(issued_by_raw).get("username", issued_by_raw)
            except Exception:
                issued_by = req.issued_by
            commandreqs.append({
                label["id"]: req.id,
                label["command"]: req.command,
                label["issued_by"]: issued_by,
                label["command_type"]: req.command_type,
                label["created_on"]: req.created_on,
            })
        render(commandreqs, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
    else:
        commandreqs = []
        for req in response.results[:limit]:
            sa = req.schedule_args
            schedule_args = None if sa is None else {
                "name": sa.name,
                "start_datetime": str(sa.start_datetime),
                "end_datetime": str(sa.end_datetime),
                "time_type": sa.time_type,
                "window_start_time": str(sa.window_start_time),
                "window_end_time": str(sa.window_end_time),
                "days": sa.days,
            }
            ca = req.command_args
            command_args = {
                "app_state": ca.app_state,
                "app_version": ca.app_version,
                "custom_settings_config": ca.custom_settings_config,
                "device_alias_name": ca.device_alias_name,
                "message": ca.message,
                "package_name": ca.package_name,
                "policy_url": ca.policy_url,
                "state": ca.state,
                "wifi_access_points": ca.wifi_access_points,
            }
            commandreqs.append({
                "id": req.id,
                "command": req.command,
                "command_type": req.command_type,
                "issued_by": req.issued_by,
                "devices": req.devices,
                "device_type": req.device_type,
                "groups": req.groups,
                "command_args": command_args,
                "schedule": req.schedule,
                "schedule_args": schedule_args,
                "created_on": str(req.created_on),
                "status": str(req.status),
            })
        render(commandreqs, format=OutputFormat.JSON.value)


@app.command("status")
def commandsv2_status(
    request_id: Optional[str] = typer.Option(None, "--request", "-r", help="Command Request ID"),
    device: Optional[str] = typer.Option(
        None, "--device", "-d", help="Filter by device name",
        shell_complete=device_name_complete,
    ),
    state_filter: Optional[str] = typer.Option(
        None, "--state", "-s",
        help=f"Filter by command state ({', '.join(CommandState.choice_list_lower())})",
        shell_complete=command_state_complete,
    ),
    limit: int = typer.Option(10, "--limit", "-l", help="Number of results"),
    json_output: bool = typer.Option(False, "--json", "-j", help="Render result in JSON format"),
):
    """Show command request status."""
    validate_creds()
    db = DBWrapper(state.creds)
    device_client = APIClient(db.get_configure()).get_device_api_client()
    commandsv2_client = APIClient(db.get_configure()).get_commandsV2_api_client()
    enterprise_id = db.get_enterprise_id()

    if not request_id:
        render("request id is not given")
        raise typer.Exit(1)

    kwargs = {}
    if device:
        try:
            search_response = device_client.get_all_devices(enterprise_id, limit=1, offset=0, name=device)
            if not search_response.results:
                render(f"Device does not exist with name {device}")
                raise typer.Exit(1)
            kwargs["device"] = search_response.results[0].id
        except ApiException as e:
            state.log.error(f"[commandsv2-status] Failed to list devices: {e}")
            render(f"ERROR: {parse_error_message(e)}")
            raise typer.Exit(1)

    if state_filter:
        kwargs["state"] = CommandState[state_filter.upper()].value

    try:
        response = commandsv2_client.get_command_request_status(enterprise_id, request_id, **kwargs)
    except ApiException as e:
        state.log.error(f"[commandsv2-status] Failed to get status for {request_id}: {e}")
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)

    _render_status_response(response, limit, json_output)


@app.command("history")
def commandsv2_history(
    device: Optional[str] = typer.Option(
        None, "--device", "-d", help="Device name",
        shell_complete=device_name_complete,
    ),
    state_filter: Optional[str] = typer.Option(
        None, "--state", "-s",
        help=f"Filter by command state ({', '.join(CommandState.choice_list_lower())})",
        shell_complete=command_state_complete,
    ),
    limit: int = typer.Option(10, "--limit", "-l", help="Number of results"),
    json_output: bool = typer.Option(False, "--json", "-j", help="Render result in JSON format"),
):
    """Show device command history."""
    validate_creds()
    db = DBWrapper(state.creds)
    commandsv2_client = APIClient(db.get_configure()).get_commandsV2_api_client()
    device_client = APIClient(db.get_configure()).get_device_api_client()
    enterprise_id = db.get_enterprise_id()

    if not device:
        render("Device name is not given")
        raise typer.Exit(1)

    try:
        search_response = device_client.get_all_devices(enterprise_id, limit=1, offset=0, name=device)
        if not search_response.results:
            render(f"Device does not exist with name {device}")
            raise typer.Exit(1)
        device_id = search_response.results[0].id
    except ApiException as e:
        state.log.error(f"[commandsv2-history] Failed to list devices: {e}")
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)

    kwargs = {}
    if state_filter:
        kwargs["state"] = CommandState[state_filter.upper()].value

    try:
        response = commandsv2_client.get_device_command_history(enterprise_id, device_id, **kwargs)
    except ApiException as e:
        state.log.error(f"[commandsv2-history] Failed to get history for {device_id}: {e}")
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)

    _render_status_response(response, limit, json_output)


@app.command("command")
def commandsv2_command(
    command_type: Optional[str] = typer.Option(
        None, "--command-type", "-ct",
        help="Command type: device, group, or dynamic",
        shell_complete=command_type_complete,
    ),
    devices: Optional[str] = typer.Option(
        None, "--devices", "-d",
        help="Space-separated list of device names",
    ),
    groups: Optional[str] = typer.Option(
        None, "--groups", "-g",
        help="Space-separated list of group IDs",
    ),
    device_type: str = typer.Option(
        "active", "--device-type", "-dt",
        help="Device type: active, inactive, or all",
        shell_complete=device_type_complete,
    ),
    command: Optional[str] = typer.Option(
        None, "--command", "-c",
        help=f"Command name ({', '.join(CommandEnum.choice_list_lower())})",
        shell_complete=command_enum_complete,
    ),
    schedule: str = typer.Option(
        "immediate", "--schedule", "-s",
        help="Schedule type: immediate, window, or recurring",
        shell_complete=schedule_type_complete,
    ),
    schedule_name: Optional[str] = typer.Option(None, "--schedule-name", "-sn", help="Schedule name"),
    start_datetime: Optional[str] = typer.Option(None, "--start", "-st", help="Start date-time"),
    end_datetime: Optional[str] = typer.Option(None, "--end", "-en", help="End date-time"),
    time_type: str = typer.Option(
        "console", "--time-type", "-tt", help="Time type: console or device",
        shell_complete=time_type_complete,
    ),
    window_start_time: Optional[str] = typer.Option(None, "--window-start", "-ws", help="Window start time"),
    window_end_time: Optional[str] = typer.Option(None, "--window-end", "-we", help="Window end time"),
    days: str = typer.Option(
        "all", "--days", "-dy",
        help="Comma-separated days: monday..sunday or all",
        shell_complete=days_complete,
    ),
    app_state: Optional[str] = typer.Option(None, "--app-state", "-as", help="App state"),
    app_version: Optional[str] = typer.Option(None, "--app-version", "-av", help="App version"),
    custom_settings_config: Optional[str] = typer.Option(None, "--custom-config", "-cs", help="Custom settings config (JSON)"),
    device_alias_name: Optional[str] = typer.Option(None, "--device-alias", "-dv", help="Device alias name"),
    message: Optional[str] = typer.Option(None, "--message", "-m", help="Message"),
    package_name: Optional[str] = typer.Option(None, "--package-name", "-pk", help="Package name"),
    policy_url: Optional[str] = typer.Option(None, "--policy-url", "-po", help="Policy URL"),
    cmd_state: Optional[str] = typer.Option(None, "--state", "-se", help="State"),
    wifi_access_points: Optional[str] = typer.Option(None, "--wifi-access-points", "-wap", help="Wifi access points"),
    json_output: bool = typer.Option(False, "--json", "-j", help="Render result in JSON format"),
):
    """Fire commands to devices and groups."""
    validate_creds()
    db = DBWrapper(state.creds)
    commandsv2_client = APIClient(db.get_configure()).get_commandsV2_api_client()
    enterprise_id = db.get_enterprise_id()

    if not command:
        render("command cannot be empty.")
        raise typer.Exit(1)

    if not command_type:
        render("command type cannot be empty.")
        raise typer.Exit(1)

    command_upper = command.upper()
    command_type_upper = command_type.upper()

    device_list = devices.split() if devices else []
    group_list = groups.split() if groups else []

    if command_type_upper in (CommandRequestTypeEnum.DEVICE.name, CommandRequestTypeEnum.DYNAMIC.name):
        if not device_list:
            render("devices cannot be empty.")
            raise typer.Exit(1)

    if command_type_upper == CommandRequestTypeEnum.GROUP.name:
        if not group_list:
            render("groups cannot be empty.")
            raise typer.Exit(1)

    # Build schedule_args
    req_days = days.split(",") if days else ["all"]
    resolved_days = []
    for day in req_days:
        day = day.strip()
        if day == "all":
            resolved_days.append(WeekDays.ALL_DAYS.value)
        else:
            resolved_days.append(WeekDays[day.upper()].value)

    schedule_args = None
    if schedule_name:
        schedule_args = {
            "name": schedule_name,
            "start_datetime": start_datetime,
            "end_datetime": end_datetime,
            "window_start_time": window_start_time,
            "window_end_time": window_end_time,
            "time_type": time_type,
            "days": resolved_days,
        }

    # Resolve device names → IDs
    device_client = APIClient(db.get_configure()).get_device_api_client()
    device_ids = []
    for device_name in device_list:
        try:
            search_response = device_client.get_all_devices(enterprise_id, limit=1, offset=0, name=device_name)
            if not search_response.results:
                render(f"Device does not exist with name {device_name}")
                raise typer.Exit(1)
            device_ids.append(search_response.results[0].id)
        except ApiException as e:
            state.log.error(f"Failed to list devices: {e}")
            render(f"ERROR: {parse_error_message(e)}")
            raise typer.Exit(1)

    # Validate group IDs
    resolved_group_ids = []
    if group_list:
        group_client = APIClient(db.get_configure()).get_group_api_client()
        for group_id in group_list:
            try:
                group_client.get_group_by_id(group_id, enterprise_id)
                resolved_group_ids.append(group_id)
            except ApiException as e:
                state.log.error(f"Group does not exist with id {group_id}: {e}")
                render(f"ERROR: {parse_error_message(e)}")
                raise typer.Exit(1)

    # Parse custom config JSON
    parsed_custom_config = None
    if custom_settings_config:
        try:
            parsed_custom_config = json.loads(custom_settings_config)
        except Exception as e:
            state.log.error(f"Could not parse JSON custom_settings_config: {e}")
            render(f"ERROR: Invalid JSON for custom_settings_config")
            raise typer.Exit(1)

    command_args_raw = {
        "app_state": app_state,
        "app_version": app_version,
        "custom_settings_config": parsed_custom_config,
        "device_alias_name": device_alias_name,
        "message": message,
        "package_name": package_name,
        "policy_url": policy_url,
        "state": cmd_state,
        "wifi_access_points": wifi_access_points,
    }
    command_args = {k: v for k, v in command_args_raw.items() if v is not None}

    command_request = V0CommandRequest(
        command_type=command_type_upper,
        devices=device_ids,
        device_type=device_type,
        command=command_upper,
        command_args=command_args,
        schedule=schedule.upper(),
        schedule_args=schedule_args,
        groups=resolved_group_ids if resolved_group_ids else None,
    )

    try:
        response = commandsv2_client.create_command(enterprise_id, command_request)
    except ApiException as e:
        state.log.error(f"[commandsv2-command] Failed to fire command {command_upper}: {e}")
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)

    if not json_output:
        render(_request_basic_response(response), format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
    else:
        render(_request_basic_response(response, OutputFormat.JSON), format=OutputFormat.JSON.value)
