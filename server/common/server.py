import socket
import logging
import signal

import time

from common.utils import *
from common.communication import read_bet, send_confirmation

class Server:
    def __init__(self, port, listen_backlog):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        
        self._shutting_down = False
        self._active_client_sockets = []

        signal.signal(signal.SIGTERM, self._handle_sigterm)

    def run(self):
        """
        Dummy Server loop

        Server that accept a new connections and establishes a
        communication with a client. After client with communucation
        finishes, servers starts to accept new connections again
        """

        while not self._shutting_down:
            client_sock = self.__accept_new_connection()
            if client_sock is not None:
                self._active_client_sockets.append(client_sock)
                self.__handle_client_connection(client_sock)
        
        self._server_socket.close()
        logging.info(f'action: Shutdown | result: success')
        

    def __handle_client_connection(self, client_sock):
        """
        Read message from a specific client socket and closes the socket

        If a problem arises in the communication with the client, the
        client socket will also be closed
        """
        try:
            logging.info(f'action: receive_bet | result: in_progress')
            bets, batch_id = read_bet(client_sock)
            if bets is not None:           
                store_bets(bets)
                logging.info(f'action: bets_stored | result: success | clientID: {bets[0].agency} | batchID: {batch_id}')

                send_confirmation(bets[0].agency, batch_id, client_sock)
        except OSError as e:
            if not self._shutting_down:
                logging.error("action: receive_message | result: fail | error: {e}")
        finally:
            client_sock.close()
            self._active_client_sockets.remove(client_sock)

    def __accept_new_connection(self):
        """
        Accept new connections

        Function blocks until a connection to a client is made.
        Then connection created is printed and returned
        """

        logging.info('action: accept_connections | result: in_progress')
        try:
            c, addr = self._server_socket.accept()
            logging.info(f'action: accept_connections | result: success | ip: {addr[0]}')
            return c
        except OSError as e:
            if not self._shutting_down:
                logging.error("action: accept_connections | result: fail | error: {e}")
            return None


    def _handle_sigterm(self, signum, frame):
        logging.info(f'action: Shutdown | result: in_progress')
        self._shutting_down = True

        self._server_socket.shutdown(socket.SHUT_RDWR)
        for client_sock in self._active_client_sockets:
            client_sock.shutdown(socket.SHUT_RDWR)
