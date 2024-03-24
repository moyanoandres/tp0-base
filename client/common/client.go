package common

import (
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

// Bet representa una apuesta.
type Bet struct {
	ClientID  string
	BetID     int
	Name      string
	Surname   string
	DNI       string
	Birthdate string
	Number    string
}

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID            string
	ServerAddress string
	LoopLapse     time.Duration
	LoopPeriod    time.Duration
}

// Client Entity that encapsulates how
type Client struct {
	config       ClientConfig
	conn         net.Conn
	shuttingDown bool
}

// NewClient Initializes a new client receiving the configuration
// as a parameter
func NewClient(config ClientConfig) *Client {
	client := &Client{
		config: config,
	}
	return client
}

// StartClientLoop Send messages to the client until some time threshold is met
func (c *Client) StartClientLoop() {
	// autoincremental msgID to identify every message sent
	betID := 1

	sigchnl := make(chan os.Signal, 1)
	signal.Notify(sigchnl, syscall.SIGTERM)

	go func() {
		<-sigchnl
		log.Infof("action: Shutdown | result: in_progress | client_id: %v",
			c.config.ID,
		)
		c.shuttingDown = true
		c.conn.Close()
	}()

loop:
	// Send messages if the loopLapse threshold has not been surpassed
	for timeout := time.After(c.config.LoopLapse); ; {
		select {
		case <-timeout:
			log.Infof("action: timeout_detected | result: success | client_id: %v",
				c.config.ID,
			)
			break loop
		default:
		}
		if c.shuttingDown {
			break loop
		}

		bet := Bet{
			ClientID:  c.config.ID,
			BetID:     betID,
			Name:      os.Getenv("Name"),
			Surname:   os.Getenv("Surname"),
			DNI:       os.Getenv("DNI"),
			Birthdate: os.Getenv("Birthdate"),
			Number:    os.Getenv("Number"),
		}

		result, err := SendBet(c, bet)
		if c.shuttingDown {
			c.conn.Close()
			return
		}
		if err != nil {
			log.Errorf("action: send_bet | result: fail | client_id: %v | bet_id: %v | error: %v",
				c.config.ID,
				betID,
				err,
			)
			break loop
		}

		if result {
			log.Infof("action: receive_confirmation | result: success | client_id: %v | bet_id: %v",
				c.config.ID,
				betID,
			)
		} else {
			log.Infof("action: receive_confirmation | result: fail | client_id: %v | bet_id: %v",
				c.config.ID,
				betID,
			)
		}

		betID++
		c.conn.Close()

		// Wait a time between sending one message and the next one
		time.Sleep(c.config.LoopPeriod)
	}

	log.Infof("action: Shutdown | result: success | client_id: %v", c.config.ID)
}
