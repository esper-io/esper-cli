import os
import subprocess
import time
from logging import Logger
from typing import Tuple
import socket
from pathlib import Path
import requests

_DEFAULT_ADB_PUB_KEY = os.path.expanduser("~/.android/adbkey.pub")


class RemoteADBError(Exception):
    '''Exceptions related to calling RemoteADB API'''
    pass


def get_remoteadb_url(environment: str,
                      enterprise_id: str,
                      device_id: str,
                      remoteadb_id: str = None) -> str:
    """
    Build and return remoteadb url for scapi endpoint
    :param environment:
    :param enterprise_id:
    :param device_id:
    :param remoteadb_id:
    :return:
    """

    host = f'https://{environment}-api.esper.cloud'
    url = f'{host}/api/v0/enterprise/{enterprise_id}/device/{device_id}/remoteadb/'

    if remoteadb_id:
        url = f'{url}{remoteadb_id}'

    return url


def exponential_sleep():
    """
    Simple generator to sleep with exponential backoff
    Limited to maximum sleep window of 8 secs
    :return:
    """
    count = 1.0

    while True:
        time.sleep(count)
        yield

        count = min(count * 2, 8.0)


def get_remoteadb_connection_details(environment: str,
                                     enterprise_id: str,
                                     device_id: str,
                                     remoteadb_id: str,
                                     api_key: str,
                                     log: Logger = None) -> Tuple[bool, dict]:
    url = get_remoteadb_url(environment, enterprise_id, device_id, remoteadb_id)

    if log:
        log.debug("[remoteadb-connect] Fetching remoteadb session details...")

    response = requests.get(
        url,
        headers={
            'Authorization': f'Bearer {api_key}'
        }
    )

    return response.ok, response.json()


def fetch_relay_endpoint(environment: str,
                         enterprise_id: str,
                         device_id: str,
                         remoteadb_id: str,
                         api_key: str,
                         log: Logger) -> Tuple[str, int]:
    """
    Poll the remoteadb-connection API and Fetch the TCP relay's IP:port

    :param environment: The client/tenant's environment
    :param api_key: API access key for the above environments
    :param enterprise_id: UUID string representing user's enterprise
    :param device_id: UUID string representing user's device, against which remote-adb connection should be established
    :param remoteadb_id: UUID string for the remote adb connection
    :return: (Relay IP, Relay Port) as a Tuple
    """

    timeout = 160.0
    sleeper = exponential_sleep()

    if log:
        log.debug(f"[remoteadb-connect] Acquiring TCP relay's IP and port... [attempting for {timeout}s]...")

    # Start the timer
    start = time.time()

    # Iterate for given duration
    while time.time() - start < timeout:

        is_ok, remoteadb_session = get_remoteadb_connection_details(
            environment, enterprise_id, device_id, remoteadb_id, api_key, log
        )

        if is_ok:
            host = remoteadb_session.get("ip")
            port = remoteadb_session.get("client_port")
            remoteadb_host = remoteadb_session.get("remoteadb_host")

            if remoteadb_host:
                try:
                    ip_address = socket.gethostbyname(remoteadb_host)
                    host = ip_address
                except socket.error as e:
                    log.error(f"[remoteadb-connect] Could not resolve ip address of '{remoteadb_host}'. Error was: {e}")
                    raise Exception('Could not resolve ip address of remoteadb host. Please try again later.')

            if host and port:
                port = int(port)

                log.debug(f"[remoteadb-connect] Recieved IP:Port -> {host}:{port}")
                return host, port

        # Retry with exponential backoff
        next(sleeper)

    # If the method didnt return, then it failed to fetch the details from SCAPI endpoint
    raise RemoteADBError(f"Failed to acquire TCP Relay's IP:port in {timeout}  secs")


def fetch_device_certificate(environment: str,
                             enterprise_id: str,
                             device_id: str,
                             remoteadb_id: str,
                             api_key: str,
                             log: Logger) -> str:
    """
    Poll the remoteadb-connection API and Fetch the TCP relay's IP:port

    :param environment: The client/tenant's environment
    :param api_key: API access key for the above environments
    :param enterprise_id: UUID string representing user's enterprise
    :param device_id: UUID string representing user's device, against which remote-adb connection should be established
    :param remoteadb_id: UUID string for the remote adb connection
    :return: (Relay IP, Relay Port) as a Tuple
    """

    timeout = 120.0
    sleeper = exponential_sleep()

    if log:
        log.debug(f"[remoteadb-connect] Acquiring Device's Certificate... [attempting for {timeout}s]...")

    # Start the timer
    start = time.time()

    # Iterate for given duration
    while time.time() - start < timeout:

        is_ok, remoteadb_session = get_remoteadb_connection_details(
            environment, enterprise_id, device_id, remoteadb_id, api_key, log
        )

        if is_ok and remoteadb_session.get("device_certificate"):
            log.debug("[remoteadb-connect] Recieved Device Certificate")
            return remoteadb_session.get("device_certificate")

        # Retry with exponential backoff
        next(sleeper)

    # If the method didnt return, then it failed to fetch the details from SCAPI endpoint
    raise RemoteADBError(f"Failed to acquire Device certificate in {timeout} secs")


def _load_adb_pub_key(client_adb_pub_key_path: str, log: Logger) -> str:
    """
    Return the ADB public key contents, trying 'adb start-server' once if the
    key file is absent (fresh host where the local adb server has never run).

    Raises RemoteADBError with actionable instructions when the key cannot be
    obtained so the caller never sends an empty key to the API.
    """
    path = Path(client_adb_pub_key_path) if client_adb_pub_key_path else Path(_DEFAULT_ADB_PUB_KEY)

    if not path.exists():
        if log:
            log.debug(
                f"[remoteadb-connect] ADB public key not found at '{path}'. "
                "Trying 'adb start-server' to generate the key pair..."
            )
        try:
            result = subprocess.run(
                ["adb", "start-server"],
                capture_output=True,
                timeout=30,
            )
            if log:
                log.debug(f"[remoteadb-connect] 'adb start-server' exited with code {result.returncode}")
        except FileNotFoundError:
            raise RemoteADBError(
                f"ADB public key not found at '{path}' and 'adb' is not installed or not on PATH.\n"
                "Install Android Platform Tools, run 'adb start-server', then retry.\n"
                "Alternatively, set the ESPER_ADB_PUB_KEY environment variable to point to an "
                "existing adbkey.pub file."
            )
        except subprocess.TimeoutExpired:
            raise RemoteADBError(
                "'adb start-server' timed out. Run it manually, then retry."
            )

    if not path.exists():
        raise RemoteADBError(
            f"ADB public key not found at '{path}' even after running 'adb start-server'.\n"
            "Run 'adb start-server' manually to generate the key pair, then retry.\n"
            "Alternatively, set the ESPER_ADB_PUB_KEY environment variable to point to an "
            "existing adbkey.pub file."
        )

    # Warn when ADB_VENDOR_KEYS is set but the resolved key is the default one.
    # This means adb may authenticate with a vendor key that the device won't
    # recognise, causing 'adb connect' to prompt for manual authorisation.
    vendor_keys_env = os.environ.get("ADB_VENDOR_KEYS", "")
    if vendor_keys_env and log:
        vendor_paths = {
            (entry.strip() + ".pub")
            for entry in vendor_keys_env.split(os.pathsep)
            if entry.strip()
        }
        if str(path) not in vendor_paths:
            log.warning(
                f"[remoteadb-connect] ADB_VENDOR_KEYS is set but the ADB public key being "
                f"sent to Esper is '{path}', which does not match any vendor key. "
                "If adb authenticates with a different identity the device will still prompt "
                "for manual authorisation. Set the ESPER_ADB_PUB_KEY environment variable to "
                "the .pub file that corresponds to the key adb will actually use."
            )

    with open(path, 'rb') as f:
        return f.read().decode('utf-8')


def initiate_remoteadb_connection(environment: str,
                                  enterprise_id: str,
                                  device_id: str,
                                  api_key: str,
                                  client_cert_path: str,
                                  client_adb_pub_key_path: str,
                                  log: Logger) -> str:
    """
    Create a Remote ADB session for given enterprise and device, and return its id.

    :param environment: The client/tenant's environment
    :param api_key: API access key for the above environments
    :param enterprise_id: UUID string representing user's enterprise
    :param device_id: UUID string representing user's device, against which remote-adb connection should be established
    :return: uuid-string - ID for the remote adb connection
    """

    url = get_remoteadb_url(environment, enterprise_id, device_id)

    client_cert = ""
    with open(client_cert_path, 'rb') as f:
        client_cert = f.read()

    # Convert byte stream to utf-8
    client_cert = client_cert.decode('utf-8')

    adb_pub_key = _load_adb_pub_key(client_adb_pub_key_path, log)
    log.debug(f"[remoteadb-connect] ADB public key loaded ({len(adb_pub_key)} chars)")

    log.debug("Initiating RemoteADB connection...")
    log.debug(f"Creating RemoteADB session at {url}")

    response = requests.post(
        url,
        json={
            'client_certificate': client_cert,
            'adb_pub_key': adb_pub_key
        },
        headers={
            'Authorization': f'Bearer {api_key}'
        }
    )

    if not response.ok:
        log.debug(f"[remoteadb-connect] Error in Remote ADB connection. [{response.status_code}] -> {response.content}")
        raise RemoteADBError("Failed to create Remote ADB Connection")

    return response.json().get('id')
