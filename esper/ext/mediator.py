import selectors
import socket
import types
from typing import Tuple
import random
import sys
import traceback
from logging import Logger
from datetime import datetime

from esper.ext.forwarder import TCPForwarder


class MediatorShutdown(BaseException):
    pass


class Mediator(object):
    selector = None
    listener = None
    log = None
    _host = None
    _port = None

    _insecure_connection = None
    _secure_connection = None
    _secure_addr = None

    _outbound_data = b''
    _inbound_data = b''

    @property
    def host(self) -> str:
        return self._host

    @property
    def port(self) -> int:
        return self._port

    def __init__(self, secure_conn=None, secure_addr=None, log=None):
        self.selector = selectors.DefaultSelector()
        self.log = log
        self._host = '127.0.0.1'
        self._port = self._get_random_unused_port()

        self._secure_connection = secure_conn
        self._secure_addr = secure_addr

    def _get_random_unused_port(self, min_port=47000, max_port=57000) -> int:
        '''Iterate over a range of ports and pick the first free port and return it '''

        for count in range((max_port - min_port)):
            port = random.randrange(min_port, max_port)
            with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as probe:
                resp = probe.connect_ex((socket.gethostname(), port))
                if resp != 0:
                    return port

    def setup_listener(self) -> Tuple[str, int]:
        self.listener = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self.listener.bind((self.host, self.port))
        self.listener.listen(1)

        if self.log:
            self.log.info(f"Starting TCP Mediator on {self.host}:{self.port}")

        self.listener.setblocking(False)
        self.selector.register(fileobj=self.listener, events=selectors.EVENT_READ, data=None)

        return self.host, self.port

    def accept_wrapper(self, sock) -> None:
        conn, addr = sock.accept()  # Should be ready to read

        if self.log:
            self.log.debug(f"Connection accepted connection from {addr[0]}:{addr[1]}")

        # Accept only one connection
        self.selector.unregister(fileobj=self.listener)
        try:
            self.listener.shutdown(socket.SHUT_RDWR)
        except OSError:
            pass
        self.listener.close()

        # Make the accepted client connection Non blocking
        conn.setblocking(False)
        self._insecure_connection = conn

        # Setup Data Object for Outbound traffic (ADB client to SSL endpoint)
        outbound_data = types.SimpleNamespace(is_secure=False,
                                              bytes_transferred=0,
                                              addr=addr)

        events = selectors.EVENT_READ | selectors.EVENT_WRITE

        # Register Outbound connection
        self.selector.register(fileobj=conn, events=events, data=outbound_data)

        # Setup Data Object for Inbound traffic (SSL endpoint to ADB client)
        inbound_data = types.SimpleNamespace(is_secure=True,
                                             bytes_transferred=0,
                                             addr=self._secure_addr)

        # Register Inbound connection
        self.selector.register(fileobj=self._secure_connection, events=events, data=inbound_data)

    def service_connection(self, key, mask) -> None:
        sock = key.fileobj
        data = key.data

        if mask & selectors.EVENT_READ:
            recv_data = sock.recv(1024)  # Should be ready to read

            if recv_data:
                # Inbound data: Receive on Local unsecure endpoint and send to Remote Secure endpoint
                # So Recv first and store in outbound buffer
                if not data.is_secure:
                    self.log.debug("Reading from ADB-Client...")
                    self._outbound_data += recv_data

                # Outbound data: Receive on Remote Secure endpoint and send to Local unsecure endpoint
                # So Recv first and store in inbound buffer
                else:
                    self.log.debug("Reading from TCP Relay...")
                    self._inbound_data += recv_data

                data.bytes_transferred += len(recv_data)

            else:
                if self.log:
                    self.log.debug(f"Closing connection to {data.addr[0]}:{data.addr[1]}")
                # self.selector.unregister(sock)
                # try:
                #     sock.shutdown(socket.SHUT_RDWR)
                # except OSError:
                #     pass
                # sock.close()
                raise MediatorShutdown("Shutdown initiated from EVENT_READ")

        if mask & selectors.EVENT_WRITE:
            try:
                if not data.is_secure:
                    self.log.debug("Sending to ADB-Client...")
                    sent = sock.send(self._inbound_data)  # Should be ready to write
                    self._inbound_data = self._inbound_data[sent:]

                else:
                    self.log.debug("Sending to TCP relay...")
                    sent = sock.send(self._outbound_data)  # Should be ready to write
                    self._outbound_data = self._outbound_data[sent:]
            except OSError:
                traceback.print_exception(*sys.exc_info())
                # try:
                #     sock.shutdown(socket.SHUT_RDWR)
                # except OSError:
                #     pass
                # sock.close()
                raise MediatorShutdown("Shutdown initiated from EVENT_WRITE")

    def run_forever(self):
        try:
            while True:
                # Block Selector till there are sockets ready for I/O
                events = self.selector.select(timeout=None)

                for key, mask in events:
                    if key.data is None:
                        self.accept_wrapper(key.fileobj)
                    else:
                        self.service_connection(key, mask)

        except KeyboardInterrupt:
            if self.log:
                self.log.info("Caught keyboard interrupt, exiting...")

        except MediatorShutdown as mexc:
            for index, sock in enumerate([self._insecure_connection, self._secure_connection]):
                try:
                    self.log.debug(f"Closing socket #{index + 1}...")
                    self.selector.unregister(sock)
                    sock.shutdown(socket.SHUT_RDWR)
                except OSError:
                    pass
                finally:
                    sock.close()

            if self.log:
                self.log.debug(f"{mexc}")
                self.log.info(f"Connections terminated from source. Exiting...")

        finally:
            self.selector.close()


class Relay(object):

    listener_host = '127.0.0.1'
    listener_port = None
    _listener_server = None
    listener_timeout = 5 * 60

    forward = None
    reverse = None

    inbound_conn = None
    inbound_addr = None
    outbound_conn = None
    outbound_addr = None

    started = None
    stopped = None

    timeout = 30 * 60  # 30 mins, same as TCP relay

    log = None

    def __init__(self,
                 relay_conn: socket.socket = None,
                 relay_addr: Tuple[str, int] = (None, None),
                 log: Logger = None):

        self.log = log
        self.outbound_conn = relay_conn
        self.outbound_addr = relay_addr

        self.listener_port = self.get_random_port()

        self.setup_listener()

    def get_random_port(self, min_port=47000, max_port=57000) -> int:
        '''Iterate over a range of ports and pick the first free port and return it '''

        for count in range((max_port - min_port)):
            port = random.randrange(min_port, max_port)
            with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as probe:
                resp = probe.connect_ex((socket.gethostname(), port))
                if resp != 0:
                    return port

    def get_listener_address(self) -> Tuple[str, int]:
        return self.listener_host, self.listener_port

    def setup_listener(self) -> None:
        # Setup a TCP Socket Listener
        self._listener_server = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._listener_server.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        self._listener_server.bind(self.get_listener_address())
        self._listener_server.listen(1)

    def accept_connection(self) -> Tuple[socket.socket, Tuple[str, int]]:
        # Accept 2 Connections on the Listener;
        # one for device, and one for ADB client
        self._listener_server.settimeout(self.listener_timeout)
        conn, addr = self._listener_server.accept()

        self.log.debug(f"Connection accepted from ADB Client: {addr[0]}:{addr[1]}")

        self.log.debug(f"Closing Listener on Port : {self.listener_port}")
        self._listener_server.close()

        self.inbound_conn = conn
        self.inbound_addr = addr

        return conn, addr

    def gather_metrics(self):
        if self.forward.connection_started and self.reverse.connection_started:
            self.started = min(self.forward.connection_started, self.reverse.connection_started)
        else:
            self.started = self.forward.connection_started or self.reverse.connection_started

        if self.forward.connection_stopped and self.reverse.connection_stopped:
            self.stopped = max(self.forward.connection_stopped, self.reverse.connection_stopped)
        else:
            self.stopped = self.forward.connection_stopped or self.reverse.connection_stopped

        self.log.info(f"Relay started: {self.started.isoformat()}")
        self.log.info(f"Relay stopped: {self.stopped.isoformat()}")
        self.log.info(f"Relay Session duration: {str(self.stopped - self.started)}")

    def start_relay(self):

        self.log.debug("Checking variables...")
        self.log.debug(self.inbound_conn)
        self.log.debug(self.inbound_addr)
        self.log.debug(self.outbound_conn)
        self.log.debug(self.outbound_addr)

        self.forward = TCPForwarder(inbound_conn=self.inbound_conn, inbound_address=self.inbound_addr,
                                    outbound_conn=self.outbound_conn, outbound_address=self.outbound_addr, log=self.log)

        self.reverse = TCPForwarder(inbound_conn=self.outbound_conn, inbound_address=self.outbound_addr,
                                    outbound_conn=self.inbound_conn, outbound_address=self.inbound_addr, log=self.log)

        self.forward.start()
        self.reverse.start()

        self.forward.join(self.timeout)
        self.reverse.join(self.timeout)

    def stop_relay(self):

        self.log.info(f"Killing Forward Thread")
        self.forward.stop()
        self.forward.join(timeout=1)

        self.log.info(f"Killing Reverse Thread...")
        self.reverse.stop()
        self.reverse.join(timeout=1)

    def cleanup_connections(self):
        self.inbound_conn.close()
        self.outbound_conn.close()

    def run_forever(self):
        try:
            self.start_relay()
            self.log.info("Forwarder Threads Timed out!")

        except KeyboardInterrupt:
            self.log.info("Encountered KeyboardInterrupt! Quitting...")

        except:
            self.log.exception("Unhandled Exception!")

        finally:
            self.log.info("Closing remaining Sockets...")

            self.stop_relay()

            self.gather_metrics()

            self.cleanup_connections()
