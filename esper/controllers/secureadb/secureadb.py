import signal
import socket
import ssl
import subprocess
import time

from cement import Controller, ex, CaughtSignal
from esperclient import V0CommandRequest
from esperclient.rest import ApiException

from esper.controllers.enums import OutputFormat
from esper.ext.api_client import APIClient
from esper.ext.certs import cleanup_certs, create_self_signed_cert, save_device_certificate, get_ssh_key
from esper.ext.db_wrapper import DBWrapper
from esper.ext.relay import Relay
from esper.ext.remoteadb_api import initiate_remoteadb_connection, fetch_device_certificate, fetch_relay_endpoint, \
    RemoteADBError, get_remoteadb_url, fetch_remotessh_ip
from esper.ext.utils import validate_creds_exists, parse_error_message


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
        db = DBWrapper(self.app.creds)
        device_name = name

        device_client = APIClient(db.get_configure()).get_device_api_client()
        enterprise_id = db.get_enterprise_id()

        kwargs = {'name': device_name}

        search_response = device_client.get_all_devices(enterprise_id, limit=1, offset=0, **kwargs)
        if not search_response.results or len(search_response.results) == 0:
            raise SecureADBWorkflowError(f'Device does not exist with name {device_name}')

        return search_response.results[0].id

    def _check_if_device_is_linux(self, device_id: str) -> bool:
        """
        Check if the device is a Linux device
        (Currently hardcoded as there isnt a reliable way to determine Linux OS for so any distros.)

        :param device_id: Device ID as UUID string
        :return: True if device is a Linux device, False otherwise
        """

        # db = DBWrapper(self.app.creds)
        # device_client = APIClient(db.get_configure()).get_device_api_client()
        # enterprise_id = db.get_enterprise_id()
        # linux_distros = ['ubuntu', 'debian', 'centos', 'fedora', 'redhat', 'rhel', 'suse', 'sles', 'opensuse']

        # device = device_client.get_device(enterprise_id, device_id)
        # # Strip os version string to get the OS type
        # os_type = device.software_info.os_version.lower().split(' ')[0]
        # self.app.log.debug(f"Device OS Type: {os_type}")
        # if os_type in linux_distros:
        #     self.app.log.debug(f"Device {device_id} is a Linux Device")
        #     return True

        # return False
        return True

    def _fire_set_adb_state_command(self, state: str, remoteadb_ip: str, remoteadb_device_port: str, device_id: str, remoteadb_id: str,
                                    api_key: str, log):
        """
        Fire the command to set the ADB state to connected
        """
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        commandsV2_client = APIClient(db.get_configure()).get_commandsV2_api_client()
        enterprise_id = db.get_enterprise_id()
        environment = db.get_configure().get("environment")
        command = "SET_ADB_STATE"

        command_args = {
            "adb_state": state,
            "remoteadb_ip": remoteadb_ip,
            "remoteadb_device_port": remoteadb_device_port,
            "remoteadb_url": get_remoteadb_url(environment, enterprise_id, device_id, remoteadb_id=remoteadb_id),
        }
        device_ids = [device_id]

        command_request = V0CommandRequest(
                                    command_type="DEVICE",
                                    devices=device_ids,
                                    device_type="all",
                                    command=command,
                                    command_args=command_args,
                                    schedule="IMMEDIATE"
        )
        self.app.log.info(f"[commandsV2-command] Firing command to set ADB state to {state}")
        try:
            response = commandsV2_client.create_command(enterprise_id, command_request)
        except ApiException as e:
            self.app.log.error(f"[commandsV2-command] Failed to fire the command {command}: {e}")
            self.app.render(f"ERROR: {parse_error_message(self.app, e)}\n")
            return
        except Exception as e:
            self.app.log.error(f"[commandsV2-command] Failed to fire the command {command}: {e}")
            self.app.render(f"ERROR: {e}\n")
            return


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
            (['-d', '--device'],
             {'help': "Device name",
              'action': 'store',
              'dest': 'device_name',
              'default': None})
        ])
    def connect(self):
        """Setup and connect securely via Remote ADB to device"""

        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)

        enterprise_id = db.get_enterprise_id()

        # Remove older certs
        cleanup_certs(self.app)

        linux_device = False

        # Create new certs
        create_self_signed_cert(local_cert=self.app.local_cert,
                                local_key=self.app.local_key)

        try:
            # Check if device is a linux device
            # Get device
            device_id = None
            if self.app.pargs.device_name:
                device_id = self._fetch_device_by_name(self.app.pargs.device_name)
                self.app.log.debug(f"Device Name: {self.app.pargs.device_name}. Device ID: {device_id}")
                linux_device = self._check_if_device_is_linux(device_id)

            elif db.get_device():
                device_id = db.get_device().get('id')
                linux_device = self._check_if_device_is_linux(device_id)

            else:
                self.app.log.error("[remoteadb-connect] Device not specified!")
                return

            self.app.render("\nDevice is a Linux Device\n" if linux_device else "\nDevice is not a Linux Device\n")

            if linux_device:
                self.app.log.debug("[remoteadb-connect] Starting Remote SSH Session")
                self.app.render("\nInitiating Remote SSH Session. This may take a few seconds...\n")
                # get ssh pub key path of current user
                pub_key_path = get_ssh_key(self.app)

                remoteadb_id = initiate_remoteadb_connection(environment=db.get_configure().get("environment"),
                                                                enterprise_id=enterprise_id,
                                                                device_id=device_id,
                                                                api_key=db.get_configure().get("api_key"),
                                                                client_cert_path="",
                                                                public_key_path=pub_key_path,
                                                                log=self.app.log)
                remoteadb_ip, remoteadb_port = fetch_relay_endpoint(environment=db.get_configure().get("environment"),
                                                        enterprise_id=enterprise_id,
                                                        device_id=device_id,
                                                        remoteadb_id=remoteadb_id,
                                                        api_key=db.get_configure().get("api_key"),
                                                        log=self.app.log)

                self._fire_set_adb_state_command(remoteadb_ip=remoteadb_ip,
                                             remoteadb_device_port=str(remoteadb_port),
                                             device_id=device_id,
                                             remoteadb_id=remoteadb_id,
                                             api_key=db.get_configure().get("api_key"),
                                             log=self.app.log,
                                             state="ENABLED")
                ipaddr = fetch_remotessh_ip(environment=db.get_configure().get("environment"),
                                            enterprise_id=enterprise_id,
                                            device_id=device_id,
                                            remoteadb_id=remoteadb_id,
                                            api_key=db.get_configure().get("api_key"),
                                            log=self.app.log)
                if ipaddr:
                    self.app.render(f"\nStarting ssh session for {ipaddr}\n")
                    command = ["ssh", f"device-user@{ipaddr}"]
                    process = subprocess.Popen(command)
                    while True:
                        if process.poll() is not None:
                            print("The process has terminated.")
                            self._fire_set_adb_state_command(remoteadb_ip=remoteadb_ip,
                                                remoteadb_device_port=str(remoteadb_port),
                                                device_id=device_id,
                                                remoteadb_id=remoteadb_id,
                                                api_key=db.get_configure().get("api_key"),
                                                log=self.app.log,
                                                state="DISABLED")
                            break
                        time.sleep(1)
                else:
                    self.app.render(f"\nFailed to fetch Remote SSH endpoint\n")


            else:
                self.app.render("\nInitiating Remote ADB Session. This may take a few seconds...\n")



                # Call SCAPI for establish remote adb connection with device
                remoteadb_id = initiate_remoteadb_connection(environment=db.get_configure().get("environment"),
                                                            enterprise_id=enterprise_id,
                                                            device_id=device_id,
                                                            api_key=db.get_configure().get("api_key"),
                                                            client_cert_path=self.app.local_cert,
                                                            pub_key_path="",
                                                            log=self.app.log)

                # Poll and fetch the TCP relay's endpoint
                relay_ip, relay_port = fetch_relay_endpoint(environment=db.get_configure().get("environment"),
                                                            enterprise_id=enterprise_id,
                                                            device_id=device_id,
                                                            remoteadb_id=remoteadb_id,
                                                            api_key=db.get_configure().get("api_key"),
                                                            log=self.app.log)

                # Poll and fetch the Device's Certificate String
                device_cert = fetch_device_certificate(environment=db.get_configure().get("environment"),
                                                    enterprise_id=enterprise_id,
                                                    device_id=device_id,
                                                    remoteadb_id=remoteadb_id,
                                                    api_key=db.get_configure().get("api_key"),
                                                    log=self.app.log)

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

                title = "Secure ADB Client"
                table = [
                    {title: f"Please connect ADB client to the following endpoint: {listener_ip} : {listener_port}"},
                    {
                        title: f"If adb-tools is installed, please run the command below:\n adb connect {listener_ip}:{listener_port}"},
                    {title: "Press Ctrl+C to quit! "},
                ]

                self.app.render(table, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")

                self.app.log.debug("[remoteadb-connect] Starting Client Mediator")

                relay.accept_connection()
                relay.start_relay()

        except (SecureADBWorkflowError, RemoteADBError) as timeout_exc:
            self.app.log.error(f"[remoteadb-connect] {str(timeout_exc)}")
            self.app.render(f"[ERROR] Issue in reaching Esper API Service for connection negotiation!\n")

        except CaughtSignal as sig:
            self.app.log.debug(f"Recieved Signal: {signal.Signals(sig.signum).name}")
            if sig.signum == signal.SIGINT:
                self.app.render("Quitting application...\n")

        except Exception as exc:
            self.app.log.error(f"Failed to establish Secure ADB connection to device: {self.app.pargs.device_name}")
            self.app.log.error(f"Exception Encountered -> {exc}")

        finally:
            if not linux_device:
                if "relay" in locals():
                    relay = locals().get("relay")
                    relay.stop_relay()
                    metrics = relay.gather_metrics()
                    relay.cleanup_connections()

                    if metrics.get('started') and metrics.get('stopped'):
                        self.app.render(f"\nSession Duration: {metrics.get('stopped') - metrics.get('started')}\n")

                    if metrics.get('bytes'):
                        self.app.render(f"\nTotal Data streamed: {metrics.get('bytes')/1024.0} KB\n")
            self.app.render("\nCleaning up...\n")
