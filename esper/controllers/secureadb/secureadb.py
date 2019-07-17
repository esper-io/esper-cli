import signal
import socket
import ssl
import time
from typing import Tuple

import requests
from cement import Controller, ex, CaughtSignal

from esper.ext.api_client import APIClient
from esper.ext.certs import cleanup_certs, create_self_signed_cert, save_device_certificate
from esper.ext.db_wrapper import DBWrapper
from esper.ext.mediator import Relay
from esper.ext.utils import validate_creds_exists


class SecureADBWorkflowError(Exception):
    pass


class SecureADB(Controller):
    class Meta:
        label = 'secureadb'

        # text displayed at the top of --help output
        description = 'Setup Secure ADB interface'

        # text displayed at the bottom of --help output
        epilog = 'Usage: espercli secureadb'

        help = 'Setup Secure ADB connection to Device'

        stacked_type = 'nested'
        stacked_on = 'base'

    def _fetch_device_by_name(self, name: str) -> str:
        """
        Fetch the device entry by its device-name

        :param name: Device Name
        :return: uuid str - Device ID as UUID string
        """
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        device_name = name

        device_client = APIClient(db.get_configure()).get_device_api_client()
        enterprise_id = db.get_enterprise_id()

        kwargs = {'name': device_name}

        search_response = device_client.get_all_devices(enterprise_id, limit=1, offset=0, **kwargs)
        if not search_response.results or len(search_response.results) == 0:
            raise SecureADBWorkflowError(f'Device does not exist with name {device_name}')

        return search_response.results[0].id

    def initiate_remoteadb_connection(self,
                                      environment: str,
                                      api_key: str,
                                      enterprise_id: str,
                                      device_id: str) -> str:
        """
        Create a Remote ADB session for given enterprise and device, and return its id.

        :param environment: The client/tenant's environment
        :param api_key: API access key for the above environments
        :param enterprise_id: UUID string representing user's enterprise
        :param device_id: UUID string representing user's device, against which remote-adb connection should be established
        :return: uuid-string - ID for the remote adb connection
        """

        host = f'https://{environment}-api.shoonyacloud.com'
        url = f'{host}/api/v0/enterprise/{enterprise_id}/device/{device_id}/remoteadb/'

        client_cert = ""
        with open(self.app.local_cert, 'rb') as f:
            client_cert = f.read()

        # Convert byte stream to utf-8
        client_cert = client_cert.decode('utf-8')

        self.app.log.info("Initiating RemoteADB connection...")
        self.app.log.debug(f"Creating RemoteADB session at {url}")

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
            self.app.log.debug(
                f"[remoteadb-connect] Error in Remote ADB connection. [{response.status_code}] -> {response.content}")
            raise SecureADBWorkflowError("Failed to create Remote ADB Connection")

        return response.json().get('id')

    def fetch_relay_endpoint(self,
                             environment: str,
                             api_key: str,
                             enterprise_id: str,
                             device_id: str,
                             remoteadb_id: str) -> Tuple[str, int]:
        """
        Poll the remoteadb-connection API and Fetch the TCP relay's IP:port

        :param environment: The client/tenant's environment
        :param api_key: API access key for the above environments
        :param enterprise_id: UUID string representing user's enterprise
        :param device_id: UUID string representing user's device, against which remote-adb connection should be established
        :param remoteadb_id: UUID string for the remote adb connection
        :return: (Relay IP, Relay Port) as a Tuple
        """

        host = f'https://{environment}-api.shoonyacloud.com'
        url = f'{host}/api/v0/enterprise/{enterprise_id}/device/{device_id}/remoteadb/{remoteadb_id}'
        timeout = 160.0

        self.app.log.info("[remoteadb-connect] Acquiring TCP relay's IP and port... [attempting for 20s]...")
        start = time.time()

        while time.time() - start < timeout:
            response = requests.get(
                url,
                headers={
                    'Authorization': f'Bearer {api_key}'
                }
            )

            if response.ok:
                host = response.json().get("ip")
                port = response.json().get("client_port")

                if host and port:
                    port = int(port)

                    self.app.log.debug(f"[remoteadb-connect] Recieved IP:Port -> {host}:{port}")
                    return host, port

            # Wait 1s before repolling the same endpoint
            time.sleep(1)

        # If the method didnt return, then it failed to fetch the details from SCAPI endpoint
        raise SecureADBWorkflowError(f"Failed to acquire TCP Relay's IP:port in {timeout}  secs")

    def fetch_device_cert(self,
                          environment: str,
                          api_key: str,
                          enterprise_id: str,
                          device_id: str,
                          remoteadb_id: str) -> str:
        """
        Poll the remoteadb-connection API and fetch the Device's SSL certificate (public key)

        :param environment: The client/tenant's environment
        :param api_key: API access key for the above environments
        :param enterprise_id: UUID string representing user's enterprise
        :param device_id: UUID string representing user's device, against which remote-adb connection should be established
        :param remoteadb_id: UUID string for the remote adb connection
        :return: Device's Public Key to be used for SSL connection. The data is in PEM format (base-64 encoded)
        """

        host = f'https://{environment}-api.shoonyacloud.com'
        url = f'{host}/api/v0/enterprise/{enterprise_id}/device/{device_id}/remoteadb/{remoteadb_id}'
        timeout = 160.0

        self.app.log.info("[remoteadb-connect] Acquiring Device's Certificate... [attempting for 20s]...")
        start = time.time()

        while time.time() - start < timeout:
            response = requests.get(
                url,
                headers={
                    'Authorization': f'Bearer {api_key}'
                }
            )

            if response.ok and response.json().get("device_certificate"):
                self.app.log.debug("[remoteadb-connect] Recieved Device Certificate")
                return response.json().get("device_certificate")

            time.sleep(1)

        raise SecureADBWorkflowError(f"Failed to acquire Device's certificate in {timeout} secs")

    def setup_ssl_connection(self,
                             host: str,
                             port: int,
                             client_cert: str,
                             client_key: str,
                             device_cert: str) -> ssl.SSLSocket:
        """
        Create a SSL connection to given host/port endpoint using Mutual TLS, ie,
        by using device (public-key) certificate as CA certs and Client Certificate
        and private key as client-side certificate

        :param host: IP or FQDN string of the TCP relay's endpoint
        :param port: Port on which the TCP relay is listening
        :param client_cert: File path to Client's Public Key (PEM Formatted)
        :param client_key: File path to Client's Private Key (PEM Formatted)
        :param device_cert: File path to Device's Public Key (PEM Formatted)
        :return: A Secure TCP Socket, wrapped in SSL Context
        """

        self.app.log.debug("[remoteadb-connect] Starting SSL Connection setup")
        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        sock.connect((host, port))

        self.app.log.debug(f"[remoteadb-connect] Connected to TCP endpoint")

        self.app.log.debug("[remoteadb-connect] Setting up SSL context")
        context = ssl.SSLContext(ssl.PROTOCOL_TLS_CLIENT)
        context.verify_mode = ssl.CERT_REQUIRED
        context.check_hostname = False

        self.app.log.debug("[remoteadb-connect] Loading Device Certificate")
        context.load_verify_locations(cafile=device_cert)
        self.app.log.debug("[remoteadb-connect] Loading Client Certificates")
        context.load_cert_chain(certfile=client_cert, keyfile=client_key)

        self.app.log.debug("[remoteadb-connect] Wrapping SSL Context")
        secure_sock = context.wrap_socket(sock, server_side=False)

        cert = secure_sock.getpeercert()
        self.app.log.debug(f"[remoteadb-connect] Peer Certificate -> {cert}")

        # TODO: verify device via cert params

        return secure_sock

    @ex(help='Setup and connect securely via Remote ADB to device',
        arguments=[
            (['-d', '--device-name'],
             {'help': "Device name",
              'action': 'store',
              'dest': 'device_name',
              'default': None})
        ])
    def connect(self):
        """Setup and connect securely via Remote ADB to device"""

        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)

        # client = APIClient(db.get_configure()).get_device_api_client()
        enterprise_id = db.get_enterprise_id()

        # Remove older certs
        cleanup_certs(self.app)

        # Create new certs
        create_self_signed_cert(local_cert=self.app.local_cert,
                                local_key=self.app.local_key)

        try:
            # Get device
            device_id = None
            if self.app.pargs.device_name:
                device_id = self._fetch_device_by_name(self.app.pargs.device_name)
                self.app.log.debug(f"Device Name: {self.app.pargs.device_name}. Device ID: {device_id}")

            elif db.get_device():
                device_id = db.get_device().get('id')

            else:
                self.app.log.error("[remoteadb-connect] Device not specified!")
                return

            # Call SCAPI for establish remote adb connection with device
            remoteadb_id = self.initiate_remoteadb_connection(environment=db.get_configure().get("environment"),
                                                              api_key=db.get_configure().get("api_key"),
                                                              enterprise_id=enterprise_id,
                                                              device_id=device_id)

            # Poll and fetch the TCP relay's endpoint
            relay_ip, relay_port = self.fetch_relay_endpoint(environment=db.get_configure().get("environment"),
                                                             api_key=db.get_configure().get("api_key"),
                                                             enterprise_id=enterprise_id,
                                                             device_id=device_id,
                                                             remoteadb_id=remoteadb_id)

            # Poll and fetch the Device's Certificate String
            device_cert = self.fetch_device_cert(environment=db.get_configure().get("environment"),
                                                 api_key=db.get_configure().get("api_key"),
                                                 enterprise_id=enterprise_id,
                                                 device_id=device_id,
                                                 remoteadb_id=remoteadb_id)

            # Save Device certificate to disk
            save_device_certificate(self.app.device_cert, device_cert)

            # Setup an SSL connection to TCP relay
            secure_sock = self.setup_ssl_connection(host=relay_ip,
                                                    port=relay_port,
                                                    client_cert=self.app.local_cert,
                                                    client_key=self.app.local_key,
                                                    device_cert=self.app.device_cert)

            relay = Relay(relay_conn=secure_sock, relay_addr=secure_sock.getsockname(), log=self.app.log)

            listener_ip, listener_port = relay.get_listener_address()

            self.app.log.info(f"|------------------------------------------------------------------------|")
            self.app.log.info(
                f"| Please connect ADB client to the following endpoint: {listener_ip} : {listener_port} |")
            self.app.log.info(f"| If adb-tools is installed, please run the command below:               |")
            self.app.log.info(
                f"|        adb connect {listener_ip}:{listener_port}                                     |")
            self.app.log.info(f"|                                                                        |")
            self.app.log.info(f"| Press Ctrl+C to quit!                                                  |")
            self.app.log.info(f"|------------------------------------------------------------------------|")

            self.app.log.debug("[remoteadb-connect] Starting Client Mediator")

            relay.accept_connection()
            relay.start_relay()

        except SecureADBWorkflowError as timeout_exc:
            self.app.log.error(f"[remoteadb-connect] {str(timeout_exc)}")

        except CaughtSignal as sig:
            self.app.log.debug(f"Recieved Signal: {signal.Signals(sig.signum).name}")
            if sig.signum == signal.SIGINT:
                self.app.log.info("Quitting application...")

        except Exception as exc:
            self.app.log.error(f"Failed to establish Secure ADB connection to device: {self.app.pargs.device_name}")
            self.app.log.debug(f"Exception Encountered -> {exc}")

        finally:
            if "relay" in locals():
                relay = locals().get("relay")
                relay.stop_relay()
                relay.gather_metrics()
                relay.cleanup_connections()
    #
    # @ex(help='Server')
    # def test_server(self):
    #     import socket
    #     import ssl
    #     import pprint
    #
    #     HOST = '127.0.0.1'
    #     PORT = 1234
    #
    #     server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    #     server_socket.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    #     server_socket.bind((HOST, PORT))
    #     server_socket.listen(10)
    #
    #     print(f"Listening on {HOST}:{PORT}...")
    #     client, fromaddr = server_socket.accept()
    #     print(f"Connection accepted from {fromaddr}")
    #
    #     secure_sock = ssl.wrap_socket(client, server_side=True,
    #                                   ca_certs="/Users/jeryn/.esper/certs1/local.pem",
    #                                   certfile="/Users/jeryn/.esper/certs2/local.pem",
    #                                   keyfile="/Users/jeryn/.esper/certs2/local.key",
    #                                   cert_reqs=ssl.CERT_REQUIRED,
    #                                   ssl_version=ssl.PROTOCOL_TLSv1_2)
    #
    #     print(repr(secure_sock.getpeername()))
    #     print(secure_sock.cipher())
    #
    #     pprint.pformat(secure_sock.getpeercert())
    #     cert = secure_sock.getpeercert()
    #     print(cert)
    #
    #     # verify client
    #     # if not cert or ('commonName', 'test') not in cert['subject'][3]:
    #     #     raise Exception("ERROR")
    #
    #     try:
    #         data = secure_sock.read(1024)
    #         secure_sock.write(data)
    #     finally:
    #         secure_sock.close()
    #         server_socket.close()
    #
    # @ex(help="Test Client",
    #     arguments=[
    #         (['-p', '--port'],
    #          {'help': "Port",
    #           'action': 'store',
    #           'type': int,
    #           'dest': 'port',
    #           'default': 1234})
    #     ]
    #     )
    # def test_client(self, host='127.0.0.1'):
    #     import socket
    #     import ssl
    #
    #     HOST = host
    #     PORT = self.app.pargs.port
    #
    #     sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    #     sock.setblocking(1)
    #     sock.connect((HOST, PORT))
    #
    #     context = ssl.SSLContext(ssl.PROTOCOL_TLSv1_2)
    #     context.verify_mode = ssl.CERT_REQUIRED
    #     context.load_verify_locations(cafile='/Users/jeryn/.esper/certs2/local.pem')
    #     context.load_cert_chain(certfile="/Users/jeryn/.esper/certs1/local.pem",
    #                             keyfile="/Users/jeryn/.esper/certs1/local.key")
    #
    #     if ssl.HAS_SNI:
    #         secure_sock = context.wrap_socket(sock, server_side=False, server_hostname=HOST)
    #     else:
    #         secure_sock = context.wrap_socket(sock, server_side=False)
    #
    #     cert = secure_sock.getpeercert()
    #     print(cert)
    #
    #     # verify server
    #     # if not cert or ('commonName', 'test') not in cert['subject'][3]:
    #     #     raise Exception("ERROR")
    #
    #     secure_sock.write(b'hello')
    #     print(secure_sock.read(1024))
    #
    #     secure_sock.close()
    #     sock.close()
    #
    # @ex(help="Test Mediator")
    # def test_mediator(self):
    #
    #     sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    #     sock.setblocking(1)
    #     sock.connect(('127.0.0.1', 1234))
    #
    #     mediator = Mediator(secure_conn=sock, secure_addr=('127.0.0.1', 6789), log=self.app.log)
    #     listener_ip, listener_port = mediator.setup_listener()
    #
    #     self.app.log.info(f"|------------------------------------------------------------------------|")
    #     self.app.log.info(f"| Please connect ADB client to the following endpoint: {listener_ip} : {listener_port} |")
    #     self.app.log.info(f"|                                                                        |")
    #     self.app.log.info(f"| Note: Press Ctrl+C to quit!                                            |")
    #     self.app.log.info(f"|------------------------------------------------------------------------|")
    #
    #     self.app.log.debug("[remoteadb-connect] Starting Client Mediator")
    #     mediator.run_forever()
