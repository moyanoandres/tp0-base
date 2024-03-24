import logging
from common.utils import Bet

def read_bet(client_sock):
    try:
        #Read header (13bytes "BET" + 4 bytes for payload size + 2 bytes for batchsize + 4 bytes for batchID)
        header = client_sock.recv(13)
        if len(header) != 13:
            logging.error("Failed to read header.")
            return None, None
        header = header.decode('utf-8')

        msg_type = header[:3]
        if msg_type != 'BET':
            logging.error("Incorrect message type received | expected: %s | received: %s", "BET", msg_type)
            return None, None

        payload_size = int(header[3:7])
        batchsize = int(header[7:9])
        batchID = int(header[9:13])

        client_sock.recv(1).decode('utf-8') #read coma

        # Read the payload using the determined payload size
        payload = b''
        while len(payload) < payload_size:
            chunk = client_sock.recv(payload_size - len(payload))
            if not chunk:
                logging.error("Failed to read payload.")
                return None, batchID
            payload += chunk

        msg = payload.decode('utf-8')
        msg = msg[:-1] #Remove trailing \n
        logging.info(f'action: receive_bet | result: success | message_received: %s;%s', header,msg)

        bets_str = msg.split(';')
        if len(bets_str) != batchsize:
            logging.error("Incorrect message format| expected batches: %s | received: %s", batchsize, len(bets_str))
            return None, batchID

        bets = []
        for bet_str in bets_str:
            parts = bet_str.split(',')
            if len(parts) != 7:
                logging.error("Bet cannot be processed")
                return None, batchID
            
            bet_id = parts[1]
        
            agency = parts[0]
            first_name = parts[2]
            last_name = parts[3]
            document = parts[4]
            birthdate = parts[5]
            number = parts[6]
            bet = Bet(agency, first_name, last_name, document, birthdate, number)
            bets.append(bet)

        return bets, batchID

    except Exception as e: 
        logging.error(f"Error reading bet: {e}")
        return None, None


def send_confirmation(agency, batch_id, client_sock):
    try:
        payload = f'{agency},{batch_id}\n'.encode('utf-8')
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
