"""
Tab-completion helpers for espercli.

Static completions use known enum values — always available even before
the CLI is configured.

Dynamic completions query the Esper API at completion time (i.e. when the
user presses <TAB>).  They fail silently — if credentials are missing or
the API is unreachable the shell simply shows no suggestions instead of
printing an error.

Usage in command files::

    from esper.cli.completions import device_name_complete, device_state_complete

    @app.command("list")
    def device_list(
        state_filter: Optional[str] = typer.Option(
            None, "--state", "-s",
            shell_complete=device_state_complete,
        ),
        ...
    ):
        ...
"""
from __future__ import annotations

# ── Static completion factories ───────────────────────────────────────────────

def _static(values: list[str]):
    """Return a Click shell-completion callback for a fixed *values* list."""
    def _complete(ctx, param, incomplete: str):
        from click.shell_completion import CompletionItem
        low = incomplete.lower()
        return [CompletionItem(v) for v in values if v.lower().startswith(low)]
    return _complete


# ── Static completions ────────────────────────────────────────────────────────

#: Device lifecycle states (lowercase, as accepted by --state)
device_state_complete = _static([
    "active", "inactive", "disabled", "onboarded", "registered",
    "provisioning_begin", "google_play_configuration",
    "policy_application_in_progress", "onboarding_in_progress",
    "onboarding_failed", "wipe_in_progress", "apps_installed",
    "branding_processed", "permission_policy_processed",
    "device_policy_processed", "device_settings_processed",
    "security_policy_processed", "phone_policy_processed",
    "custom_settings_processed",
])

#: V2 command names (lowercase)
command_enum_complete = _static([
    "reboot", "set_new_policy", "update_heartbeat", "wipe",
    "install", "uninstall", "update_latest_dpc", "set_kiosk_app",
    "set_device_lockdown_state", "set_app_state", "add_wifi_ap",
    "remove_wifi_ap", "update_device_config",
])

#: Command request types
command_type_complete = _static(["device", "group", "dynamic"])

#: V2 command lifecycle states
command_state_complete = _static([
    "queued", "initiate", "acknowledge", "in_progress",
    "timeout", "success", "failure", "scheduled", "cancelled",
])

#: Device type filter for V2 commands
device_type_complete = _static(["active", "inactive", "all"])

#: Schedule types for V2 commands
schedule_type_complete = _static(["immediate", "window", "recurring"])

#: Schedule days
days_complete = _static([
    "all", "monday", "tuesday", "wednesday",
    "thursday", "friday", "saturday", "sunday",
])

#: Schedule time type
time_type_complete = _static(["console", "device"])

#: legacy-format flag values
legacy_format_complete = _static(["true", "false"])

#: GMS flag values
gms_complete = _static(["true", "false"])

#: Telemetry period values
period_complete = _static(["custom", "today", "yesterday", "weekly", "monthly"])

#: Telemetry statistic values
statistic_complete = _static(["avg", "max", "min", "sum"])


# ── Dynamic completion helpers ────────────────────────────────────────────────

def _load_db():
    """Return a configured DBWrapper, or *None* if credentials aren't set up."""
    try:
        import os
        from tinydb import TinyDB
        from esper.ext.db_wrapper import DBWrapper
        from esper.cli.state import CREDS_FILE

        if not os.path.exists(CREDS_FILE):
            return None
        db = DBWrapper(TinyDB(CREDS_FILE))
        if not db.get_configure():
            return None
        return db
    except Exception:
        return None


def device_name_complete(ctx, param, incomplete: str):
    """Complete device names (alias or hardware name) from the API."""
    from click.shell_completion import CompletionItem
    try:
        db = _load_db()
        if db is None:
            return []
        from esper.ext.api_client import APIClient
        enterprise_id = db.get_enterprise_id()
        client = APIClient(db.get_configure()).get_device_api_client()
        kwargs = {"name": incomplete} if incomplete else {}
        resp = client.get_all_devices(enterprise_id, limit=30, offset=0, **kwargs)
        items = []
        for d in resp.results:
            label = d.alias_name or d.device_name
            hint = d.device_name if d.alias_name else None
            items.append(CompletionItem(label, help=hint))
        return items
    except Exception:
        return []


def group_name_complete(ctx, param, incomplete: str):
    """Complete group names from the API."""
    from click.shell_completion import CompletionItem
    try:
        db = _load_db()
        if db is None:
            return []
        from esper.ext.api_client import APIClient
        enterprise_id = db.get_enterprise_id()
        client = APIClient(db.get_configure()).get_group_api_client()
        kwargs = {"name": incomplete} if incomplete else {}
        resp = client.get_all_groups(enterprise_id, limit=30, offset=0, **kwargs)
        return [CompletionItem(g.name) for g in resp.results]
    except Exception:
        return []


def app_name_complete(ctx, param, incomplete: str):
    """Complete application names (returns app ID, shows name as hint) from the API."""
    from click.shell_completion import CompletionItem
    try:
        db = _load_db()
        if db is None:
            return []
        from esper.ext.api_client import APIClient
        enterprise_id = db.get_enterprise_id()
        client = APIClient(db.get_configure()).get_application_api_client()
        kwargs = {"application_name": incomplete, "is_hidden": False} if incomplete else {"is_hidden": False}
        resp = client.get_all_applications(enterprise_id, limit=30, offset=0, **kwargs)
        return [
            CompletionItem(a.id, help=a.application_name)
            for a in resp.results
        ]
    except Exception:
        return []
