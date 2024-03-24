import logging
from common.utils import Bet

def read_bet(client_sock):
    try:
        #Read header (8bytes "BET," + 4 bytes for payload size)
        header = client_sock.recv(8)
        if len(header) != 8:
            logging.error("Failed to read header.")
            return None, None
        
        msg_type = header[:3].decode('utf-8')
        if msg_type != 'BET':
            logging.error("Incorrect message type received | expected: %s | received: %s", "BET", msg_type)
            return None, None

        payload_size = int(header[4:].decode('utf-8'))

        client_sock.recv(1).decode('utf-8') #read coma

        # Read the payload using the determined payload size
        payload = b''
        while len(payload) < payload_size:
            chunk = client_sock.recv(payload_size - len(payload))
            if not chunk:
                logging.error("Failed to read payload.")
                return None, None
            payload += chunk

        msg = payload.decode('utf-8')
        logging.info(f'action: receive_bet | result: success | message_received: %s,%s',header.decode('utf-8'),msg)
        parts = msg.split(',')

        if len(parts) != 7:
            logging.error("Incorrect message format received |")
            return None, None
        
        

        bet_id = parts[1]
        
        agency = parts[0]
        first_name = parts[2]
        last_name = parts[3]
        document = parts[4]
        birthdate = parts [5]
        number = parts[6]

        bet = Bet(agency, first_name, last_name, document, birthdate, number)
        return [bet], bet_id

    except Exception as e: 
        logging.error(f"Error reading bet: {e}")
        return None, None


def send_confirmation(bet, bet_id, client_sock):
    try:
        payload = f'{bet.agency},{bet_id}\n'.encode('utf-8')
        payload_size = len(payload)
        payload_size_str = f"{payload_size:04d}"
        message = f"ACK,{payload_size_str},".encode('utf-8') + payload

        total_sent = 0
        while total_sent < len(message):
            sent = client_sock.send(message[total_sent:])
            if sent == 0:
                raise RuntimeError("Socket connection broken")
            total_sent += sent
    except Exception as e:
        logging.error(f"Error sending confirmation: {e}")
        return None, None