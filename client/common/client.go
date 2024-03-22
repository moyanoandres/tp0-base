package common

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

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

// CreateClientSocket Initializes client socket. In case of
// failure, error is printed in stdout/stderr and exit 1
// is returned
func (c *Client) createClientSocket() error {
	conn, err := net.Dial("tcp", c.config.ServerAddress)
	if err != nil {
		log.Fatalf(
			"action: connect | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
	}
	if !c.shuttingDown {
		c.conn = conn
	}
	return nil
}

// StartClientLoop Send messages to the client until some time threshold is met
func (c *Client) StartClientLoop() {
	// autoincremental msgID to identify every message sent
	msgID := 1

	sigchnl := make(chan os.Signal, 1)
	signal.Notify(sigchnl, syscall.SIGTERM)

	go func() {
		<-sigchnl
		log.Infof("action: Shutdown | result: in_progress | client_id: %v",
			c.config.ID,
		)
		c.shuttingDown = true
		if c.conn != nil {
			c.conn.Close()
		}
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

		// Create the connection the server in every loop iteration. Send an
		c.createClientSocket()

		if c.shuttingDown {
			if c.conn != nil {
				c.conn.Close()
			}
			return
		}
		// TODO: Modify the send to avoid short-write
		_, err := fmt.Fprintf(
			c.conn,
			"[CLIENT %v] Message N°%v\n",
			c.config.ID,
			msgID,
		)
		if err != nil {
			if !c.shuttingDown { //The case in which the shutdown was triggered while reading
				log.Errorf("action: send_message | result: fail | client_id: %v | error: %v",
					c.config.ID,
					err,
				)
			}
			return
		}

		msg, err := bufio.NewReader(c.conn).ReadString('\n')
		msgID++

		if c.conn != nil {
			c.conn.Close()
		}

		if err != nil {
			if !c.shuttingDown {
				log.Errorf("action: receive_message | result: fail | client_id: %v | error: %v",
					c.config.ID,
					err,
				)
			}
			return
		}
		log.Infof("action: receive_message | result: success | client_id: %v | msg: %v",
			c.config.ID,
			msg,
		)

		// Wait a time between sending one message and the next one
		time.Sleep(c.config.LoopPeriod)
	}

	log.Infof("action: Shutdown | result: success | client_id: %v", c.config.ID)
}
