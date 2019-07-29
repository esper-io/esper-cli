import logging
import socket
import threading
from datetime import datetime
from typing import Tuple, ByteString

BUFFER_SIZE = 1024


class ClientConnection(object):
    '''
    Client Connection Management object
    '''

    def __init__(self, connection: socket.socket, addr: Tuple[str, int]):
        self.connection = connection
        self.ip, self.port = addr

    def __repr__(self) -> str:
        return ":".join([self.ip, str(self.port)])

    def send(self, data: bytes) -> None:
        return self.connection.sendall(data)

    def recv(self, bufsize: int = BUFFER_SIZE) -> ByteString:
        return self.connection.recv(bufsize)

    def close(self) -> None:
        if self.connection:
            try:
                self.connection.shutdown(socket.SHUT_RDWR)
            except OSError:
                pass
            self.connection.close()


class Forwarder(threading.Thread):
    '''
    Multithreaded Socket Forwarder. Accepts a pair of socketserver connections, and forwards
    the data from one to the other
    '''

    # Thread class with a _stop() method.
    # The thread itself has to check
    # regularly for the stopped() condition.

    def __init__(self,
                 inbound_conn: socket.socket = None,
                 inbound_address: Tuple[str, int] = (None, None),
                 outbound_conn: socket.socket = None,
                 outbound_address: Tuple[str, int] = (None, None),
                 log: logging.Logger = None,
                 *args,
                 **kwargs):
        super(Forwarder, self).__init__(*args, **kwargs)

        self.args = kwargs.get('args')
        self.kwargs = kwargs.get('kwargs')
        self.log = log

        self.inbound_connection = ClientConnection(inbound_conn, inbound_address)
        self.outbound_connection = ClientConnection(outbound_conn, outbound_address)

        self.name = f"Forwarder [{self.inbound_connection}] -> [{self.outbound_connection}]"

        self._mystop = threading.Event()

        # Metrics
        self._bytes_transferred = 0
        self._connection_started = None
        self._connection_stopped = None

    @property
    def bytes(self):
        return self._bytes_transferred

    @bytes.setter
    def bytes(self, value):
        self._bytes_transferred += value

    def start_timer(self) -> None:
        self._connection_started = datetime.utcnow()

    def stop_timer(self) -> None:
        self._connection_stopped = datetime.utcnow()

    @property
    def duration(self) -> float:
        return (self._connection_stopped - self._connection_started).total_seconds()

    @property
    def connection_started(self):
        return self._connection_started

    @property
    def connection_stopped(self):
        return self._connection_stopped

    # function using _stop function
    def stop(self):
        self._mystop.set()

    def stopped(self) -> bool:
        return self._mystop.isSet()

    def cleanup(self) -> None:
        # Cleanup sockets
        self.inbound_connection.close()
        self.outbound_connection.close()

    def run(self):
        return NotImplemented


class TCPForwarder(Forwarder):

    def run(self):
        self.log.debug(f"Starting Forwarder Thread: {self.name}")

        self.start_timer()
        while True:
            if self.stopped():
                self.log.debug(f"[{self.name}] Received Thread stop event.")
                break

            try:
                data = self.inbound_connection.recv(BUFFER_SIZE)
                if len(data) <= 0:
                    self.log.debug(f"[{self.name}] Zero Data! Connection closed by client: {self.inbound_connection}!")
                    break
            except socket.error:
                self.log.debug(f"[{self.name}] Read Error! Connection closed by client: {self.inbound_connection}!")
                break
            except:
                self.log.debug(f"[{self.name}] Connection Error while relaying!")
                break

            try:
                self.outbound_connection.send(data)
                self.bytes = len(data)
            except socket.error:
                self.log.debug(f"[{self.name}] Write Error! Connection closed by client {self.outbound_connection}")
                break
            except:
                self.log.debug(f"[{self.name}] Connection Error while relaying!")
                break

        self.stop_timer()

        # Cleanup sockets
        self.cleanup()

        self.log.debug(f"[{self.name}] [METRICS] Bytes Txfered: {self.bytes} bytes, Connection Duration: {self.duration}s")
        return
