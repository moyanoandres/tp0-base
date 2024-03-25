import socket
import logging
import signal
import threading

import time

from common.utils import *
from common.communication import read_bet, send_confirmation, get_header, send_winners

NUMB_OF_AGENCIES = 5

class Server:
    def __init__(self, port, listen_backlog):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        
        self._shutting_down = False

        """
        Los "sockets de los clientes que esperan resultados" son un subset de los sockets
        activos, y se usan segun su ID para mandar los resultados del sorteo correspondiente
        una vez que todos estén listos.
        Los "sockets activos" son tanto los que esperan resultados, como los que estén abiertos para
        envío o recepción de mensajes.
        """
        self._active_client_sockets = []
        self._client_socks_awaiting_results = {}

        # Locks
        self._sotrage_lock = threading.Lock()
        self._active_client_lock = threading.Lock()
        self._awaiting_results_client_lock = threading.Lock()

        signal.signal(signal.SIGTERM, self._handle_sigterm)

    def run(self):
        """
        Server loop

        Server that accept a new connections and establishes a
        communication with a client. After client with communucation
        finishes, servers starts to accept new connections again.
        """

        while not self._shutting_down:
            client_sock = self.__accept_new_connection()
            if client_sock is not None:
                with self._active_client_lock:
                    self._active_client_sockets.append(client_sock)
                threading.Thread(target=self.__handle_client_connection, args=(client_sock,)).start()
        
        self._server_socket.close()
        logging.info(f'action: Shutdown | result: success')
        

    def __handle_client_connection(self, client_sock):
        """
        Read header from a specific client socket depending on the type of the
        message perform an action:
        BET -> load the bets received
        FIN -> register this client as finished, and if all clients are, conduct the draw

        The client socket will only remain open if the client sent a FIN, and is awaiting
        the results of the draw.
        """

        will_await_results = False
        try:
            msg_type, header = get_header(client_sock)

            if msg_type == 'BET':
                logging.info(f'action: receive_and_store_bet | result: in_progress')
                bets, batch_id = read_bet(client_sock, header)
                if bets is not None:
                    with self._sotrage_lock:    # bloquear hasta garantizar exclusion para el uso del almacenamiento      
                        store_bets(bets)
                        logging.info(f'action: receive_and_store_bet | result: success | clientID: {bets[0].agency} | batchID: {batch_id}')
                    send_confirmation(bets[0].agency, batch_id, client_sock)
            elif msg_type == 'FIN':
                will_await_results = True
                clientID = int(header[9:])
                logging.info(f'action: receive_fin | result: success | clientID: {clientID}')
                with self._awaiting_results_client_lock:
                    self._client_socks_awaiting_results[clientID] = client_sock
                    if len(self._client_socks_awaiting_results) >= NUMB_OF_AGENCIES: #Todos los clientes notificaron al sv
                        self._conduct_draw()
            else:
                logging.error(f"action: receive_message | result: fail | error: Unknown type message received")

        except Exception as e:
            if not self._shutting_down:
                logging.error(f"action: receive_message | result: fail | error: {e}")
        finally:
            if not will_await_results:
                with self._active_client_lock:
                    client_sock.close()
                    self._active_client_sockets.remove(client_sock)

    def __accept_new_connection(self):
        """
        Accept new connections

        Function blocks until a connection to a client is made.
        Then connection created is printed and returned
        """

        logging.info(f'action: accept_connections | result: in_progress')
        try:
            c, addr = self._server_socket.accept()
            logging.info(f'action: accept_connections | result: success | ip: {addr[0]}')
            return c
        except OSError as e:
            if not self._shutting_down:
                logging.error(f"action: accept_connections | result: fail | error: {e}")
            return None

    def _conduct_draw(self):
        #find and save winners by agency
        winners_by_agency = [[] for _ in range(NUMB_OF_AGENCIES)]
        with self._sotrage_lock:
            for bet in load_bets():
                if has_won(bet):
                    winners_by_agency[bet.agency - 1].append(bet.document)

        #send winners to their respective agency
        logging.info(f'action: send_winners | result: in_progress')
        for i in range(0, len(winners_by_agency)):
            send_winners(self._client_socks_awaiting_results[i + 1], winners_by_agency[i])
        logging.info(f'action: send_winners | result: success')

        #close sockets and cleanup
        with self._active_client_lock:
            for socket in self._active_client_sockets:
                socket.close()
            self._active_client_sockets = []

        #no hace falta cerrar en este caso porque son los mismos que _active_client_sockets
        #el lock correspondiente es tomado antes de la llamada a esta función
        self._client_socks_awaiting_results = {}
                

    def _handle_sigterm(self, signum, frame):
        logging.info(f'action: Shutdown | result: in_progress')
        self._shutting_down = True

        self._server_socket.shutdown(socket.SHUT_RDWR)
        with self._active_client_lock:
            for client_sock in self._active_client_sockets:
                client_sock.shutdown(socket.SHUT_RDWR)
