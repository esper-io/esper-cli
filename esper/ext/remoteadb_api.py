import time
from logging import Logger
from typing import Tuple

import requests


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

    host = f'https://{environment}-api.shoonyacloud.com'
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


def initiate_remoteadb_connection(environment: str,
                                  enterprise_id: str,
                                  device_id: str,
                                  api_key: str,
                                  client_cert_path: str,
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

    log.debug("Initiating RemoteADB connection...")
    log.debug(f"Creating RemoteADB session at {url}")

    response = requests.post(
        url,
        json={
            'client_certificate': client_cert
        },
        headers={
            'Authorization': f'Bearer {api_key}'
        }
    )

    if not response.ok:
        log.debug(f"[remoteadb-connect] Error in Remote ADB connection. [{response.status_code}] -> {response.content}")
        raise RemoteADBError("Failed to create Remote ADB Connection")

    return response.json().get('id')
