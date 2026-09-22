"""
Lightweight replacement for Cement's App object.
All CLI modules import `state` from here instead of using `self.app`.
"""
import json
import logging
import os
import stat
import sys
from pathlib import Path

import typer
from tinydb import TinyDB

from esper.ext.db_wrapper import DBWrapper

# Paths can be overridden via environment variables so that test suites and
# non-standard deployments can point to a different credential store without
# editing source code.
CREDS_FILE = os.environ.get(
    "ESPER_CREDS_FILE",
    os.path.expanduser("~/.esper/db/creds.json"),
)
CERTS_FOLDER = os.environ.get(
    "ESPER_CERTS_DIR",
    os.path.expanduser("~/.esper/certs"),
)


class EsperState:
    """Module-level singleton that replaces the Cement App context."""

    def __init__(self):
        self._creds = None
        self.debug = False

        # Cert paths (used by secureadb)
        self.local_key = os.path.join(CERTS_FOLDER, "local.key")
        self.local_cert = os.path.join(CERTS_FOLDER, "local.pem")
        self.device_cert = os.path.join(CERTS_FOLDER, "device.pem")
        self.certs_path = CERTS_FOLDER

        # Logger
        logging.basicConfig(
            level=logging.WARNING,
            format="%(levelname)s %(name)s: %(message)s",
        )
        self.log = logging.getLogger("espercli")

    @property
    def creds(self):
        """Lazy-initialise TinyDB, creating parent dirs as needed.

        The credentials directory is created with mode 0o700 (owner-only) and
        the database file is locked down to 0o600 after first write so that
        the API key is never readable by other users on a shared system.
        """
        if self._creds is None:
            creds_path = Path(CREDS_FILE)
            creds_dir = creds_path.parent

            # Create the directory with restrictive permissions.
            creds_dir.mkdir(parents=True, exist_ok=True)
            try:
                os.chmod(creds_dir, stat.S_IRWXU)  # 0o700
            except OSError:
                pass  # best-effort; may fail on some filesystems

            self._creds = TinyDB(str(creds_path))

            # Lock down the file itself after TinyDB creates/opens it.
            try:
                os.chmod(str(creds_path), stat.S_IRUSR | stat.S_IWUSR)  # 0o600
            except OSError:
                pass  # best-effort
        return self._creds

    def set_debug(self, debug: bool):
        self.debug = debug
        level = logging.DEBUG if debug else logging.WARNING
        logging.getLogger("espercli").setLevel(level)


# Single shared instance used by all command modules
state = EsperState()


# ---------------------------------------------------------------------------
# Helpers that replace the Cement utility functions
# ---------------------------------------------------------------------------

def validate_creds():
    """Exit 1 with a Rich error panel if credentials are not configured."""
    db = DBWrapper(state.creds)
    if not db.get_configure():
        from rich.console import Console
        from rich.panel import Panel
        Console(stderr=True).print(
            Panel(
                "[bold red]✗[/bold red]  No credentials found.\n\n"
                "[dim]Run [cyan]espercli configure[/cyan] to set your environment, "
                "enterprise ID and API token.[/dim]",
                title="[bold red]Not Configured[/bold red]",
                border_style="red",
                expand=False,
                padding=(0, 1),
            )
        )
        raise typer.Exit(1)


def parse_error_message(exception) -> str:
    """Extract a human-readable message from an ApiException."""
    try:
        body = json.loads(exception.body) if exception.body else {}
        return body.get("message") or exception.reason
    except (ValueError, AttributeError):
        return getattr(exception, "reason", str(exception))
