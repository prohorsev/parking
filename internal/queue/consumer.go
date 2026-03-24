package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/parking/api/internal/domain"
	"github.com/parking/api/internal/repository"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Consumer struct {
	ch   *amqp.Channel
	repo repository.Repository
}

func NewConsumer(conn *amqp.Connection, repo repository.Repository) (*Consumer, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("consumer: open channel: %w", err)
	}
	if err := declareQueue(ch, ParkingCreateQueue); err != nil {
		return nil, fmt.Errorf("consumer: declare queue: %w", err)
	}
	if err := ch.Qos(1, 0, false); err != nil {
		return nil, fmt.Errorf("consumer: set QoS: %w", err)
	}
	return &Consumer{ch: ch, repo: repo}, nil
}

func (c *Consumer) Start(ctx context.Context) error {
	msgs, err := c.ch.Consume(
		ParkingCreateQueue,
		"parking-consumer",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("consumer: start consume: %w", err)
	}

	log.Printf("queue consumer: listening on %q", ParkingCreateQueue)

	go func() {
		for {
			select {
			case msg, ok := <-msgs:
				if !ok {
					log.Println("queue consumer: channel closed")
					return
				}
				if err = c.handle(msg); err != nil {
					log.Printf("queue consumer: handle error: %v — requeueing", err)
					_ = msg.Nack(false, true)
				} else {
					_ = msg.Ack(false)
				}
			case <-ctx.Done():
				log.Println("queue consumer: context cancelled, stopping")
				return
			}
		}
	}()

	return nil
}

func (c *Consumer) handle(msg amqp.Delivery) error {
	var lot domain.ParkingLot
	if err := json.Unmarshal(msg.Body, &lot); err != nil {
		_ = msg.Ack(false)
		return fmt.Errorf("unmarshal (message discarded): %w", err)
	}
	if err := c.repo.Upsert([]domain.ParkingLot{lot}); err != nil {
		return fmt.Errorf("upsert %q: %w", lot.Name, err)
	}
	log.Printf("queue consumer: persisted parking lot %q (provider: %s, id: %s)",
		lot.Name, lot.Provider, lot.ExternalID)
	return nil
}

func (c *Consumer) Close() { _ = c.ch.Close() }
