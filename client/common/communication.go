package common

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

// SendMessage sends a bet to the server, returns true if the bet was confirmed.
func SendBet(c *Client, bet Bet) (bool, error) {
	//socket creation and graceful shutdown handling

	err := c.createClientSocket()
	if err != nil {
		return false, err
	}
	if c.shuttingDown {
		c.conn.Close()
		return false, err
	}

	// Creating the message with its header from the bet received
	payload := fmt.Sprintf("%s,%d,%s,%s,%s,%s,%s", bet.ClientID, bet.BetID, bet.Name, bet.Surname, bet.DNI, bet.Birthdate, bet.Number)
	message := fmt.Sprintf("BET,%04d,%s", len(payload), payload)

	log.Infof("Bet message being sent: %v",
		message,
	)

	// Send message to the server
	log.Infof("action: send_bet | result: in_progress | client_id: %v | bet_id: %v",
		c.config.ID,
		bet.BetID,
	)
	message += "\n"

	sentBytes := 0
	for sentBytes < len(message) {
		n, err := fmt.Fprintf(c.conn, message[sentBytes:])
		if err != nil {
			return false, err
		}
		if n == 0 {
			return false, fmt.Errorf("connection closed by remote host")
		}
		sentBytes += n
	}

	log.Infof("action: send_bet | result: success | client_id: %v | bet_id: %v",
		c.config.ID,
		bet.BetID,
	)

	// Receive ACK from the server
	ackPayload, err := receiveACK(c.conn)
	if err != nil {
		return false, err
	}

	log.Infof("ACK Received: %s | Expected ACK: ,%s,%d", ackPayload, bet.ClientID, bet.BetID)

	// Check if the response is the expected ACK
	if ackPayload == fmt.Sprintf(",%s,%d", bet.ClientID, bet.BetID) {
		return true, nil
	}

	// If the response is not the correct ACK
	return false, fmt.Errorf("unexpected response from server: %s", ackPayload)
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
	message := fmt.Sprintf("BET%04d%02d%04d%s", len(payload), len(bets), batchID, payload)

	log.Infof("Bet message being sent: %v", message)

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

	log.Infof("ACK recibido: %s | ACK esperado: ,%s,%d", ackPayload, c.config.ID, batchID)

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
	header := make([]byte, 8) // Header size is fixed at 8 bytes
	n, err := io.ReadFull(conn, header)
	if err != nil || n < 8 {
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
