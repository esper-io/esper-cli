"""
Secure ADB commands — replaces esper/controllers/secureadb/secureadb.py.
`espercli secureadb connect`
"""
import signal
import socket
import ssl
from typing import Optional

import typer
from esperclient.rest import ApiException

from esper.cli.output import render
from esper.cli.state import state, validate_creds
from esper.controllers.enums import OutputFormat
from esper.ext.api_client import APIClient
from esper.ext.certs import cleanup_certs, create_self_signed_cert, save_device_certificate
from esper.ext.db_wrapper import DBWrapper
from esper.ext.relay import Relay
from esper.ext.remoteadb_api import (
    initiate_remoteadb_connection, fetch_device_certificate, fetch_relay_endpoint,
    RemoteADBError,
)

app = typer.Typer(help="Setup Secure ADB connection to device")


class SecureADBWorkflowError(Exception):
    pass


def _fetch_device_by_name(db: DBWrapper, name: str) -> str:
    """Fetch device ID by device name."""
    device_client = APIClient(db.get_configure()).get_device_api_client()
    enterprise_id = db.get_enterprise_id()
    search_response = device_client.get_all_devices(enterprise_id, limit=1, offset=0, name=name)
    if not search_response.results:
        raise SecureADBWorkflowError(f"Device does not exist with name {name}")
    return search_response.results[0].id


def _setup_ssl_connection(host: str, port: int,
                          client_cert: str, client_key: str,
                          device_cert: str) -> ssl.SSLSocket:
    """Create a mutual-TLS SSL connection to the relay endpoint."""
    state.log.debug("[remoteadb-connect] Starting SSL Connection setup")
    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    sock.connect((host, port))
    state.log.debug("[remoteadb-connect] Connected to TCP endpoint")

    context = ssl.SSLContext(ssl.PROTOCOL_TLS_CLIENT)
    context.verify_mode = ssl.CERT_REQUIRED
    context.check_hostname = False
    context.load_verify_locations(cafile=device_cert)
    context.load_cert_chain(certfile=client_cert, keyfile=client_key)

    secure_sock = context.wrap_socket(sock, server_side=False)
    state.log.debug(f"[remoteadb-connect] Peer Certificate -> {secure_sock.getpeercert()}")
    return secure_sock


# ---------------------------------------------------------------------------
# Commands
# ---------------------------------------------------------------------------

@app.command("connect")
def secureadb_connect(
    device_name: Optional[str] = typer.Option(
        None, "--device", "-d", help="Device name"
    ),
):
    """Setup and connect securely via Remote ADB to a device."""
    validate_creds()
    db = DBWrapper(state.creds)
    environment = db.get_configure().get("environment")
    enterprise_id = db.get_enterprise_id()
    api_key = db.get_configure().get("api_key")

    # Remove older certs and create new ones
    cleanup_certs(state)
    create_self_signed_cert(local_cert=state.local_cert, local_key=state.local_key)

    relay = None
    try:
        # Resolve device ID
        if device_name:
            device_id = _fetch_device_by_name(db, device_name)
            state.log.debug(f"Device Name: {device_name}. Device ID: {device_id}")
        elif db.get_device():
            device_id = db.get_device().get("id")
        else:
            state.log.error("[remoteadb-connect] Device not specified!")
            render("ERROR: No device specified. Use --device or set an active device.")
            raise typer.Exit(1)

        render("\nInitiating Remote ADB Session. This may take a few seconds...\n")

        remoteadb_id = initiate_remoteadb_connection(
            environment=environment,
            enterprise_id=enterprise_id,
            device_id=device_id,
            api_key=api_key,
            client_cert_path=state.local_cert,
            log=state.log,
        )

        relay_ip, relay_port = fetch_relay_endpoint(
            environment=environment,
            enterprise_id=enterprise_id,
            device_id=device_id,
            remoteadb_id=remoteadb_id,
            api_key=api_key,
            log=state.log,
        )

        device_cert_contents = fetch_device_certificate(
            environment=environment,
            enterprise_id=enterprise_id,
            device_id=device_id,
            remoteadb_id=remoteadb_id,
            api_key=api_key,
            log=state.log,
        )

        save_device_certificate(state.device_cert, device_cert_contents)

        secure_sock = _setup_ssl_connection(
            host=relay_ip,
            port=relay_port,
            client_cert=state.local_cert,
            client_key=state.local_key,
            device_cert=state.device_cert,
        )

        relay = Relay(relay_conn=secure_sock, relay_addr=secure_sock.getsockname(), log=state.log)
        listener_ip, listener_port = relay.get_listener_address()

        title = "Secure ADB Client"
        table = [
            {title: f"Please connect ADB client to the following endpoint: {listener_ip} : {listener_port}"},
            {title: f"If adb-tools is installed, please run:\n adb connect {listener_ip}:{listener_port}"},
            {title: "Press Ctrl+C to quit!"},
        ]
        render(table, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")

        state.log.debug("[remoteadb-connect] Starting Client Mediator")
        relay.accept_connection()
        relay.start_relay()

    except (SecureADBWorkflowError, RemoteADBError) as exc:
        state.log.error(f"[remoteadb-connect] {exc}")
        render("[ERROR] Issue in reaching Esper API Service for connection negotiation!")

    except KeyboardInterrupt:
        render("Quitting application...")

    except Exception as exc:
        state.log.error(f"Failed to establish Secure ADB connection to device: {device_name}")
        state.log.debug(f"Exception: {exc}")

    finally:
        if relay is not None:
            relay.stop_relay()
            metrics = relay.gather_metrics()
            relay.cleanup_connections()

            if metrics.get("started") and metrics.get("stopped"):
                render(f"\nSession Duration: {metrics['stopped'] - metrics['started']}\n")
            if metrics.get("bytes"):
                render(f"\nTotal Data streamed: {metrics['bytes'] / 1024.0} KB\n")
