package common

import (
	"encoding/csv"
	"io"
	"net"
	"os"
	"os/signal"
	"strconv"
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
	BatchSize     string
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
	file, err := os.Open("/data/agency.csv")
	if err != nil {
		log.Fatalf("error opening agency file: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// autoincremental msgID to identify every message sent
	batchID := 1
	betID := 1

	// SIGTERM handling
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
	// Send messages until EOF
	for {
		if c.shuttingDown {
			break loop
		}

		// batch reading from file
		batchsize, err := strconv.Atoi(c.config.BatchSize)
		if err != nil {
			log.Errorf("Error converting batch size to integer: %v", err)
			break
		}

		bets := make([]*Bet, 0)
		for i := 0; i < batchsize; i++ {
			record, err := reader.Read()
			if err != nil {
				if err == io.EOF {
					break
				}
				log.Fatalf("error reading record from CSV: %v", err)
			}

			bet := &Bet{
				ClientID:  c.config.ID,
				BetID:     betID,
				Name:      record[0],
				Surname:   record[1],
				DNI:       record[2],
				Birthdate: record[3],
				Number:    record[4],
			}
			bets = append(bets, bet)
			betID++
		}

		//If no more bets, send FIN msg to server, receive winners and exit the loop
		if len(bets) == 0 {
			log.Infof("action: finished_loading_file | result: success | client_id: %v | final_batch_id: %v",
				c.config.ID,
				batchID-1,
			)
			winners, err := sendEndNotification(c)
			c.conn.Close()

			if err == nil {
				log.Infof("action: consulta_ganadores | result: success | cant_ganadores: %v",
					winners,
				)
			} else {
				log.Errorf("action: consulta_ganadores | result: fail | client_id: %v | error: %v",
					c.config.ID,
					err,
				)
			}
			break loop
		}

		//send batch of bets to server
		result, err := SendBatch(c, bets, batchID)
		if c.shuttingDown {
			c.conn.Close()
			return
		}
		if err != nil {
			log.Errorf("action: send_batch | result: fail | client_id: %v | batch_id: %v | error: %v",
				c.config.ID,
				batchID,
				err,
			)
			break loop
		}

		if result {
			log.Infof("action: receive_confirmation | result: success | client_id: %v | batch_id: %v",
				c.config.ID,
				batchID,
			)
		} else {
			log.Infof("action: receive_confirmation | result: fail | client_id: %v | batch_id: %v",
				c.config.ID,
				batchID,
			)
		}

		batchID++
		c.conn.Close()

		// Wait a time between sending one message and the next one
		//time.Sleep(c.config.LoopPeriod)
	}

	log.Infof("action: Shutdown | result: success | client_id: %v", c.config.ID)
}
