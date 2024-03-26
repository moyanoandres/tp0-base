package common

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	HEADER_SIZE = 12
)

func receiveWIN(conn net.Conn) (int, error) {
	// Read the header (WIN message) from the server
	header := make([]byte, HEADER_SIZE) // Header size is fixed at 12 bytes: 'WIN,' + 8 bytes of payload size
	n, err := io.ReadFull(conn, header)
	if err != nil || n < HEADER_SIZE {
		return -1, err
	}

	// Parse the header to get the payload size
	headerStr := string(header)
	parts := strings.Split(headerStr, ",")
	if len(parts) != 2 || parts[0] != "WIN" {
		return -1, fmt.Errorf("invalid WIN header format")
	}

	payloadSize, err := strconv.Atoi(parts[1])
	if err != nil {
		return -1, err
	}

	// Read the payload using the determined payload size
	winPayload := make([]byte, payloadSize)
	totalBytesRead := 0
	for totalBytesRead < payloadSize {
		bytesRead, err := conn.Read(winPayload[totalBytesRead:])
		if err != nil {
			return -1, err
		}
		totalBytesRead += bytesRead
	}

	if len(winPayload) == 0 { //no hubieron ganadores
		return 0, nil
	}

	DNIs := strings.Split(string(winPayload), ",")
	return len(DNIs), nil

}

func sendEndNotification(c *Client) (int, error) {
	//socket creation and graceful shutdown handling
	err := c.createClientSocket()
	if err != nil {
		return -1, err
	}
	if c.shuttingDown {
		c.conn.Close()
		return -1, err
	}

	clientID, err := strconv.Atoi(c.config.ID)
	message := fmt.Sprintf("FIN%20d\n", clientID)
	log.Infof("action: send_fin | result: in_progress | client_id: %v",
		c.config.ID,
	)
	sentBytes := 0
	for sentBytes < len(message) {
		n, err := fmt.Fprintf(c.conn, message[sentBytes:])
		if err != nil {
			log.Infof("action: send_fin | result: fail | client_id: %v",
				c.config.ID,
			)
			return -1, err
		}
		if n == 0 {
			return -1, fmt.Errorf("conexión cerrada por el host remoto")
		}
		sentBytes += n
	}

	log.Infof("action: send_fin | result: success | client_id: %v",
		c.config.ID,
	)

	// Receive Winners
	winners, err := receiveWIN(c.conn)
	if err != nil {
		return -1, err
	}

	return winners, nil
}

// SendBatch envía un lote de apuestas al servidor y devuelve true si todas las apuestas del lote fueron confirmadas correctamente.
func SendBatch(c *Client, bets []*Bet, batchID int) (bool, error) {
	//socket creation and graceful shutdown handling
	err := c.createClientSocket()
	if err != nil {
		return false, err
	}
	if c.shuttingDown {
		c.conn.Close()
		return false, err
	}
	// Crear el mensaje con su encabezado a partir de las apuestas recibidas
	payload := ""
	for _, bet := range bets {
		payload += fmt.Sprintf(";%s,%d,%s,%s,%s,%s,%s", bet.ClientID, bet.BetID, bet.Name, bet.Surname, bet.DNI, bet.Birthdate, bet.Number)
	}

	// BET PAYLOAD_SIZE BATCHSIZE BatchID PAYLOAD
	message := fmt.Sprintf("BET%08d%04d%08d%s", len(payload), len(bets), batchID, payload)
	//log.Infof("Bet message being sent: %v", message)

	// Enviar el mensaje al servidor
	log.Infof("action: send_batch | result: in_progress | client_id: %v | batch_id: %v",
		c.config.ID,
		batchID,
	)

	message += "\n"
	sentBytes := 0
	for sentBytes < len(message) {
		n, err := fmt.Fprintf(c.conn, message[sentBytes:])
		if err != nil {
			return false, err
		}
		if n == 0 {
			return false, fmt.Errorf("conexión cerrada por el host remoto")
		}
		sentBytes += n
	}

	log.Infof("action: send_batch | result: success | client_id: %v | batch_id: %v",
		c.config.ID,
		batchID,
	)

	// Recibir ACK del servidor
	ackPayload, err := receiveACK(c.conn)
	if err != nil {
		return false, err
	}

	//log.Infof("ACK recibido: %s | ACK esperado: ,%s,%d", ackPayload, c.config.ID, batchID)

	// Verificar si la respuesta es el ACK esperado
	if ackPayload == fmt.Sprintf(",%s,%d", c.config.ID, batchID) {
		return true, nil
	}

	// Si la respuesta no es el ACK correcto
	return false, fmt.Errorf("respuesta inesperada del servidor: %s", ackPayload)
}

// receiveACK receives and parses the ACK message from the server.
func receiveACK(conn net.Conn) (string, error) {
	// Read the header (ACK message) from the server
	header := make([]byte, HEADER_SIZE) // Header size is fixed at 12 bytes: 'ACK,' + 8 bytes of payload size
	n, err := io.ReadFull(conn, header)
	if err != nil || n < HEADER_SIZE {
		return "", err
	}

	// Parse the header to get the payload size
	headerStr := string(header)
	parts := strings.Split(headerStr, ",")
	if len(parts) != 2 || parts[0] != "ACK" {
		return "", fmt.Errorf("invalid ACK header format")
	}

	payloadSize, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", err
	}

	// Read the payload using the determined payload size
	ackPayload := make([]byte, payloadSize)
	totalBytesRead := 0
	for totalBytesRead < payloadSize {
		bytesRead, err := conn.Read(ackPayload[totalBytesRead:])
		if err != nil {
			return "", err
		}
		totalBytesRead += bytesRead
	}

	return string(ackPayload), nil
}

// CreateClientSocket initializes client socket. In case of
// failure, error is printed in stdout/stderr and exit 1
// is returned.
func (c *Client) createClientSocket() error {
	conn, err := net.Dial("tcp", c.config.ServerAddress)
	if err != nil {
		log.Fatalf(
			"action: connect | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
	}
	c.conn = conn
	return nil
}
