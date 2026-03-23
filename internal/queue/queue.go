package queue

import (
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	ParkingCreateQueue = "parking.create"
)

func Connect(url string) (*amqp.Connection, error) {
	var (
		conn *amqp.Connection
		err  error
	)
	delays := []time.Duration{2, 4, 8, 16, 30}
	for i, d := range delays {
		conn, err = amqp.Dial(url)
		if err == nil {
			log.Printf("rabbitmq: connected on attempt %d", i+1)
			return conn, nil
		}
		log.Printf("rabbitmq: attempt %d failed, retrying in %s: %v", i+1, d*time.Second, err)
		time.Sleep(d * time.Second)
	}
	return nil, fmt.Errorf("rabbitmq: all connection attempts failed: %w", err)
}

func declareQueue(ch *amqp.Channel, name string) error {
	_, err := ch.QueueDeclare(
		name,
		true,
		false,
		false,
		false,
		nil,
	)
	return err
}
