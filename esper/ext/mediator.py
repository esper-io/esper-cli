import random
import selectors
import socket
import types

from typing import Tuple


class MediatorShutdown(BaseException):
    pass


class Mediator(object):
    """
    A simple TCP relay based on Non-blocking sockets. This implementation uses `selectors` to eliminate the need for
    threads to manage the various data streams.
    """

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
