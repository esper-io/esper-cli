"""
Group commands — replaces esper/controllers/enterprise/group.py.
`espercli group list/show/set-active/unset-active/create/update/delete/add/remove/devices/move`
"""
from typing import Optional

import typer
from esperclient import DeviceGroup, DeviceGroupUpdate, DeviceGroupPartialUpdate
from esperclient.rest import ApiException

from esper.cli.completions import device_state_complete, group_name_complete
from esper.cli.output import render
from esper.cli.state import state, validate_creds, parse_error_message
from esper.controllers.enums import DeviceState, OutputFormat
from esper.ext.api_client import APIClient
from esper.ext.db_wrapper import DBWrapper

app = typer.Typer(help="Device group commands")


# ---------------------------------------------------------------------------
# Shared helpers
# ---------------------------------------------------------------------------

def _group_basic_response(group, fmt=OutputFormat.TABULATED):
    parent = group.parent.split("/")[-2] if group.parent else group.parent
    if fmt == OutputFormat.TABULATED:
        title, details = "TITLE", "DETAILS"
        return [
            {title: "id", details: group.id},
            {title: "name", details: group.name},
            {title: "parent_id", details: parent},
            {title: "device_count", details: group.device_count},
            {title: "path", details: group.path},
            {title: "children_count", details: group.children_count},
        ]
    return {
        "id": group.id, "name": group.name, "parent": parent,
        "device_count": group.device_count, "path": group.path,
        "children_count": group.children_count,
    }


def _resolve_group_by_name_or_id(
    group_client, enterprise_id,
    group_name: Optional[str], group_id: Optional[str], tag: str
):
    """Return (group_id, resolved_group_object) by ID or name."""
    if group_id:
        try:
            response = group_client.get_group_by_id(group_id, enterprise_id)
            if group_name and group_name != response.name:
                render(f"Group does not exist with id {group_id} and name {group_name}")
                raise typer.Exit(1)
            return group_id, response
        except ApiException as e:
            state.log.error(f"[{tag}] Group does not exist with id {group_id}: {e}")
            render(f"ERROR: {parse_error_message(e)}")
            raise typer.Exit(1)
    elif group_name:
        try:
            search_response = group_client.get_all_groups(
                enterprise_id, limit=1, offset=0, name=group_name
            )
            for g in search_response.results:
                if g.name == group_name:
                    return g.id, g
            render(f"Group does not exist with name {group_name}")
            raise typer.Exit(1)
        except ApiException as e:
            state.log.error(f"[{tag}] Failed to list groups: {e}")
            render(f"ERROR: {parse_error_message(e)}")
            raise typer.Exit(1)
    else:
        return None, None


def _get_group_device_ids(device_client, enterprise_id, group_id) -> list:
    """Paginate through all devices in a group and return their IDs."""
    device_ids = []
    counter = 0
    limit = 100
    try:
        while True:
            offset = limit * counter
            response = device_client.get_all_devices(
                enterprise_id, group=group_id, limit=limit, offset=offset
            )
            if not response.results:
                break
            device_ids.extend([d.id for d in response.results])
            counter += 1
    except ApiException as e:
        state.log.error(f"[_get_group_device_ids] Failed to list devices: {e}")
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)
    return device_ids


# ---------------------------------------------------------------------------
# Commands
# ---------------------------------------------------------------------------

@app.command("list")
def group_list(
    name: Optional[str] = typer.Option(None, "--name", "-n", help="Filter groups by name"),
    limit: int = typer.Option(20, "--limit", "-l", help="Results per page"),
    offset: int = typer.Option(0, "--offset", "-i", help="Initial index"),
    all_results: bool = typer.Option(
        False, "--all", "-A",
        help="Fetch ALL groups, auto-paginating (ignores --limit / --offset).",
    ),
    json_output: bool = typer.Option(False, "--json", "-j", help="Render result in JSON format"),
):
    """List device groups."""
    validate_creds()
    db = DBWrapper(state.creds)
    group_client = APIClient(db.get_configure()).get_group_api_client()
    enterprise_id = db.get_enterprise_id()

    kwargs = {}
    if name:
        kwargs["name"] = name

    try:
        if all_results:
            from esper.cli.output import console as _console
            results = []
            page_size, cur_offset = 100, 0
            with _console.status("[dim]Fetching all groups…[/dim]", spinner="dots"):
                while True:
                    page = group_client.get_all_groups(
                        enterprise_id, limit=page_size, offset=cur_offset, **kwargs
                    )
                    results.extend(page.results)
                    if len(results) >= page.count or not page.results:
                        break
                    cur_offset += page_size

            class _FR:
                def __init__(self, r):
                    self.results = r
                    self.count = len(r)
            response = _FR(results)
        else:
            response = group_client.get_all_groups(enterprise_id, limit=limit, offset=offset, **kwargs)
    except ApiException as e:
        state.log.error(f"[group-list] Failed to list groups: {e}")
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)

    label = {"id": "ID", "name": "NAME", "parent_id": "PARENT ID", "device_count": "DEVICE COUNT"}
    groups = []
    for g in response.results:
        parent = g.parent.split("/")[-2] if g.parent else g.parent
        groups.append({
            label["id"]: g.id, label["name"]: g.name,
            label["device_count"]: g.device_count or 0, label["parent_id"]: parent,
        })

    render(f"Number of Groups: {response.count}")
    if not json_output:
        render(groups, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
    else:
        render(groups, format=OutputFormat.JSON.value)


@app.command("show")
def group_show(
    group_name: str = typer.Argument(..., help="Group name"),
    group_id: Optional[str] = typer.Option(None, "--groupid", "-id", help="Group ID"),
    active: bool = typer.Option(
        False, "--active", "-a", help="Set as the active group"
    ),
    json_output: bool = typer.Option(False, "--json", "-j", help="Render result in JSON format"),
):
    """Show group details (and optionally set as active)."""
    validate_creds()
    db = DBWrapper(state.creds)
    group_client = APIClient(db.get_configure()).get_group_api_client()
    enterprise_id = db.get_enterprise_id()

    gid, response = _resolve_group_by_name_or_id(
        group_client, enterprise_id, group_name, group_id, "group-show"
    )
    if response is None:
        render(f"Group does not exist with name {group_name}")
        raise typer.Exit(1)

    if active:
        db.set_group({"id": response.id, "name": response.name})

    if not json_output:
        render(_group_basic_response(response), format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
    else:
        render(_group_basic_response(response, OutputFormat.JSON), format=OutputFormat.JSON.value)


@app.command("set-active")
def group_set_active(
    name: Optional[str] = typer.Option(None, "--name", "-n", help="Group name"),
    group_id: Optional[str] = typer.Option(None, "--groupid", "-id", help="Group ID"),
    json_output: bool = typer.Option(False, "--json", "-j", help="Render result in JSON format"),
):
    """Set or show the active group."""
    validate_creds()
    db = DBWrapper(state.creds)
    group_client = APIClient(db.get_configure()).get_group_api_client()
    enterprise_id = db.get_enterprise_id()

    if group_id or name:
        gid, response = _resolve_group_by_name_or_id(
            group_client, enterprise_id, name, group_id, "group-active"
        )
        db.set_group({"id": response.id, "name": response.name})
    else:
        group = db.get_group()
        if group is None or group.get("name") is None:
            render("There is no active group.")
            raise typer.Exit(1)
        try:
            response = group_client.get_group_by_id(group.get("id"), enterprise_id)
        except ApiException as e:
            state.log.error(f"[group-active] Failed to show active group: {e}")
            render(f"ERROR: {parse_error_message(e)}")
            raise typer.Exit(1)

    if not json_output:
        render(_group_basic_response(response), format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
    else:
        render(_group_basic_response(response, OutputFormat.JSON), format=OutputFormat.JSON.value)


@app.command("unset-active")
def group_unset_active():
    """Unset the current active group."""
    validate_creds()
    db = DBWrapper(state.creds)

    group = db.get_group()
    if group is None or group.get("name") is None:
        render("There is no active group.")
        raise typer.Exit(1)

    db.unset_group()
    render(
        f"Unset the active group with id: {group.get('id')} and name: {group.get('name')}"
    )


@app.command("create")
def group_create(
    name: Optional[str] = typer.Option(None, "--name", "-n", help="Group name"),
    parent: Optional[str] = typer.Option(None, "--parentid", "-pid", help="Parent group ID"),
    json_output: bool = typer.Option(False, "--json", "-j", help="Render result in JSON format"),
):
    """Create a group."""
    validate_creds()
    db = DBWrapper(state.creds)
    group_client = APIClient(db.get_configure()).get_group_api_client()
    enterprise_id = db.get_enterprise_id()

    if not name:
        render("name cannot be empty.")
        raise typer.Exit(1)

    data = DeviceGroup(name=name, parent=parent) if parent else DeviceGroup(name=name)

    try:
        response = group_client.create_group(enterprise_id, data)
    except ApiException as e:
        state.log.error(f"[group-create] Failed to create group: {e}")
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)

    if not json_output:
        render(_group_basic_response(response), format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
    else:
        render(_group_basic_response(response, OutputFormat.JSON), format=OutputFormat.JSON.value)


@app.command("update")
def group_update(
    name: str = typer.Argument(..., help="Group current name"),
    group_id: Optional[str] = typer.Option(None, "--groupid", "-id", help="Group ID"),
    new_name: Optional[str] = typer.Option(None, "--name", "-n", help="New group name"),
    json_output: bool = typer.Option(False, "--json", "-j", help="Render result in JSON format"),
):
    """Update a group's name."""
    validate_creds()
    db = DBWrapper(state.creds)
    group_client = APIClient(db.get_configure()).get_group_api_client()
    enterprise_id = db.get_enterprise_id()

    gid, _ = _resolve_group_by_name_or_id(group_client, enterprise_id, name, group_id, "group-update")

    if not new_name:
        render("new name cannot be empty.")
        raise typer.Exit(1)

    data = DeviceGroupPartialUpdate()
    data.name = new_name

    try:
        response = group_client.partial_update_group(gid, enterprise_id, data)
    except ApiException as e:
        state.log.error(f"[group-update] Failed to update group: {e}")
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)

    if not json_output:
        render(_group_basic_response(response), format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
    else:
        render(_group_basic_response(response, OutputFormat.JSON), format=OutputFormat.JSON.value)


@app.command("delete")
def group_delete(
    name: str = typer.Argument(..., help="Group name"),
    group_id: Optional[str] = typer.Option(None, "--groupid", "-id", help="Group ID"),
    yes: bool = typer.Option(False, "--yes", "-y", help="Skip confirmation prompt"),
):
    """Delete a group."""
    if not yes:
        typer.confirm(f"Delete group '{name}'? This cannot be undone.", abort=True)
    validate_creds()
    db = DBWrapper(state.creds)
    group_client = APIClient(db.get_configure()).get_group_api_client()
    enterprise_id = db.get_enterprise_id()

    gid, _ = _resolve_group_by_name_or_id(group_client, enterprise_id, name, group_id, "group-delete")

    try:
        group_client.delete_group(gid, enterprise_id)
        render(f"Group with name {name} deleted successfully")
        group = db.get_group()
        if group and group.get("id") == gid:
            db.unset_group()
    except ApiException as e:
        state.log.error(f"[group-delete] Failed to delete group: {e}")
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)


@app.command("add")
def group_add(
    group: Optional[str] = typer.Option(None, "--group", "-g", help="Group name", shell_complete=group_name_complete),
    group_id: Optional[str] = typer.Option(None, "--groupid", "-id", help="Group ID"),
    devices: Optional[str] = typer.Option(
        None, "--devices", "-d",
        help='Device names (space-separated, e.g. "dev1 dev2 dev3")'
    ),
    json_output: bool = typer.Option(False, "--json", "-j", help="Render result in JSON format"),
):
    """Add devices to a group."""
    validate_creds()
    db = DBWrapper(state.creds)
    group_client = APIClient(db.get_configure()).get_group_api_client()
    device_client = APIClient(db.get_configure()).get_device_api_client()
    enterprise_id = db.get_enterprise_id()

    if group or group_id:
        gid, _ = _resolve_group_by_name_or_id(
            group_client, enterprise_id, group, group_id, "group-add"
        )
    else:
        active = db.get_group()
        if active is None or active.get("name") is None:
            render("There is no active group.")
            raise typer.Exit(1)
        gid = active.get("id")

    device_names = devices.split() if devices else []
    if not device_names:
        render("devices cannot be empty.")
        raise typer.Exit(1)
    if len(device_names) > 1000:
        render("Cannot add more than 1000 devices at a time.")
        raise typer.Exit(1)

    request_device_ids = []
    for device_name in device_names:
        try:
            search_response = device_client.get_all_devices(
                enterprise_id, limit=1, offset=0, name=device_name
            )
            if not search_response.results:
                render(f"Device does not exist with name {device_name}")
                raise typer.Exit(1)
            request_device_ids.append(search_response.results[0].id)
        except ApiException as e:
            state.log.error(f"[group-add] Failed to list devices: {e}")
            render(f"ERROR: {parse_error_message(e)}")
            raise typer.Exit(1)

    existing_ids = _get_group_device_ids(device_client, enterprise_id, gid)
    existing_ids.extend(request_device_ids)

    data = DeviceGroupPartialUpdate()
    data.device_ids = existing_ids

    try:
        response = group_client.partial_update_group(gid, enterprise_id, data, action="add")
    except ApiException as e:
        state.log.error(f"[group-add] Failed to add devices to group: {e}")
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)

    if not json_output:
        render(_group_basic_response(response), format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
    else:
        render(_group_basic_response(response, OutputFormat.JSON), format=OutputFormat.JSON.value)


@app.command("remove")
def group_remove(
    group: Optional[str] = typer.Option(None, "--group", "-g", help="Group name", shell_complete=group_name_complete),
    group_id: Optional[str] = typer.Option(None, "--groupid", "-id", help="Group ID"),
    devices: Optional[str] = typer.Option(
        None, "--devices", "-d",
        help='Device names (space-separated, e.g. "dev1 dev2 dev3")'
    ),
    json_output: bool = typer.Option(False, "--json", "-j", help="Render result in JSON format"),
):
    """Remove devices from a group."""
    validate_creds()
    db = DBWrapper(state.creds)
    group_client = APIClient(db.get_configure()).get_group_api_client()
    device_client = APIClient(db.get_configure()).get_device_api_client()
    enterprise_id = db.get_enterprise_id()

    if group or group_id:
        gid, _ = _resolve_group_by_name_or_id(
            group_client, enterprise_id, group, group_id, "group-remove"
        )
    else:
        active = db.get_group()
        if active is None or active.get("name") is None:
            render("There is no active group.")
            raise typer.Exit(1)
        gid = active.get("id")

    device_names = devices.split() if devices else []
    if not device_names:
        render("devices cannot be empty.")
        raise typer.Exit(1)
    if len(device_names) > 1000:
        render("Cannot remove more than 1000 devices at a time.")
        raise typer.Exit(1)

    request_device_ids = []
    for device_name in device_names:
        try:
            search_response = device_client.get_all_devices(
                enterprise_id, limit=1, offset=0, name=device_name
            )
            if not search_response.results:
                render(f"Device does not exist with name {device_name}")
                raise typer.Exit(1)
            request_device_ids.append(search_response.results[0].id)
        except ApiException as e:
            state.log.error(f"[group-remove] Failed to list devices: {e}")
            render(f"ERROR: {parse_error_message(e)}")
            raise typer.Exit(1)

    current_ids = _get_group_device_ids(device_client, enterprise_id, gid)
    extra = list(set(request_device_ids) - set(current_ids))
    if extra:
        render("The given devices are not present in the group.")
        raise typer.Exit(1)

    data = DeviceGroupPartialUpdate()
    data.device_ids = request_device_ids

    try:
        response = group_client.partial_update_group(gid, enterprise_id, data, action="remove")
    except ApiException as e:
        state.log.error(f"[group-remove] Failed to remove devices: {e}")
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)

    if not json_output:
        render(_group_basic_response(response), format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
    else:
        render(_group_basic_response(response, OutputFormat.JSON), format=OutputFormat.JSON.value)


@app.command("devices")
def group_devices(
    group: Optional[str] = typer.Option(None, "--group", "-g", help="Group name", shell_complete=group_name_complete),
    group_id: Optional[str] = typer.Option(None, "--groupid", "-id", help="Group ID"),
    limit: int = typer.Option(20, "--limit", "-l", help="Results per page"),
    offset: int = typer.Option(0, "--offset", "-i", help="Initial index"),
    json_output: bool = typer.Option(False, "--json", "-j", help="Render result in JSON format"),
):
    """List devices in a group."""
    validate_creds()
    db = DBWrapper(state.creds)
    group_client = APIClient(db.get_configure()).get_group_api_client()
    device_client = APIClient(db.get_configure()).get_device_api_client()
    enterprise_id = db.get_enterprise_id()

    if group or group_id:
        gid, _ = _resolve_group_by_name_or_id(
            group_client, enterprise_id, group, group_id, "group-devices"
        )
    else:
        active = db.get_group()
        if active is None or active.get("name") is None:
            render("There is no active group.")
            raise typer.Exit(1)
        gid = active.get("id")

    try:
        response = device_client.get_all_devices(enterprise_id, group=gid, limit=limit, offset=offset)
    except ApiException as e:
        state.log.error(f"[group-devices] Failed to list group devices: {e}")
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)

    render(f"Number of Devices: {response.count}")
    if not json_output:
        label = {"id": "ID", "name": "NAME", "model": "MODEL",
                 "state": "CURRENT STATE", "tags": "TAGS"}
        devices = []
        for d in response.results:
            dev_name = d.alias_name if d.alias_name else d.device_name
            tags = ", ".join(d.tags) if d.tags else ""
            devices.append({
                label["id"]: d.id, label["name"]: dev_name,
                label["model"]: d.hardware_info.get("manufacturer"),
                label["state"]: DeviceState(d.status).name,
                label["tags"]: tags,
            })
        render(devices, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
    else:
        devices = []
        for d in response.results:
            dev_name = d.alias_name if d.alias_name else d.device_name
            devices.append({
                "id": d.id, "device": dev_name,
                "model": d.hardware_info.get("manufacturer"),
                "state": DeviceState(d.status).name,
                "tags": d.tags,
            })
        render(devices, format=OutputFormat.JSON.value)


@app.command("move")
def group_move(
    group: Optional[str] = typer.Option(None, "--group", "-g", help="Group name", shell_complete=group_name_complete),
    group_id: Optional[str] = typer.Option(None, "--groupid", "-id", help="Group ID"),
    parent: Optional[str] = typer.Option(None, "--parent", "-p", help="New parent group name"),
    parent_id: Optional[str] = typer.Option(None, "--parentid", "-pid", help="New parent group ID"),
    json_output: bool = typer.Option(False, "--json", "-j", help="Render result in JSON format"),
):
    """Move a group to a different parent."""
    validate_creds()
    db = DBWrapper(state.creds)
    group_client = APIClient(db.get_configure()).get_group_api_client()
    enterprise_id = db.get_enterprise_id()

    if group or group_id:
        gid, group_obj = _resolve_group_by_name_or_id(
            group_client, enterprise_id, group, group_id, "group-move"
        )
        group_name = group_obj.name if group_obj else group
    else:
        active = db.get_group()
        if active is None or active.get("name") is None:
            render("There is no active group.")
            raise typer.Exit(1)
        gid = active.get("id")
        group_name = active.get("name")

    if not parent_id and not parent:
        render("Either parent name or parent id should be present.")
        raise typer.Exit(1)

    if parent_id:
        try:
            parent_obj = group_client.get_group_by_id(parent_id, enterprise_id)
            if parent and parent != parent_obj.name:
                render(f"Group does not exist with id {parent_id} and name {parent}")
                raise typer.Exit(1)
            data = DeviceGroupUpdate(name=group_name, parent=parent_id)
        except ApiException as e:
            state.log.error(f"[group-move] Group does not exist with id {parent_id}: {e}")
            render(f"ERROR: {parse_error_message(e)}")
            raise typer.Exit(1)
    else:
        try:
            search_response = group_client.get_all_groups(
                enterprise_id, limit=1, offset=0, name=parent
            )
            if not search_response.results:
                render(f"Group does not exist with name {parent}")
                raise typer.Exit(1)
            pid = search_response.results[0].id
            data = DeviceGroupUpdate(name=group_name, parent=pid)
        except ApiException as e:
            state.log.error(f"[group-move] Failed to list groups: {e}")
            render(f"ERROR: {parse_error_message(e)}")
            raise typer.Exit(1)

    try:
        response = group_client.partial_update_group(gid, enterprise_id, data, action="move")
    except ApiException as e:
        state.log.error(f"[group-move] Failed to move group: {e}")
        render(f"ERROR: {parse_error_message(e)}")
        raise typer.Exit(1)

    if not json_output:
        render(_group_basic_response(response), format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
    else:
        render(_group_basic_response(response, OutputFormat.JSON), format=OutputFormat.JSON.value)
