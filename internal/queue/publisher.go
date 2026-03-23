package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/parking/api/internal/domain"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Publisher struct {
	ch *amqp.Channel
}

func NewPublisher(conn *amqp.Connection) (*Publisher, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("publisher: open channel: %w", err)
	}
	if err := declareQueue(ch, ParkingCreateQueue); err != nil {
		return nil, fmt.Errorf("publisher: declare queue: %w", err)
	}
	return &Publisher{ch: ch}, nil
}

func (p *Publisher) Publish(ctx context.Context, lot domain.ParkingLot) error {
	body, err := json.Marshal(lot)
	if err != nil {
		return fmt.Errorf("publisher: marshal: %w", err)
	}
	return p.ch.PublishWithContext(ctx,
		"",
		ParkingCreateQueue,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	)
}

func (p *Publisher) Close() { p.ch.Close() }
