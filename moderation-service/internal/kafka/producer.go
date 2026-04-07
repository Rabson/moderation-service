package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

type Producer struct {
	enabled bool
	writer  *kafka.Writer
}

func NewProducer(enabled bool, brokers []string, topic string) *Producer {
	if !enabled {
		return &Producer{enabled: false}
	}
	w := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		RequiredAcks: kafka.RequireOne,
		Async:        true,
		BatchTimeout: 10 * time.Millisecond,
	}
	return &Producer{enabled: true, writer: w}
}

func (p *Producer) Publish(ctx context.Context, key string, value any) error {
	if !p.enabled || p.writer == nil {
		return nil
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal kafka event: %w", err)
	}

	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(key),
		Value: payload,
	})
}

func (p *Producer) Close() error {
	if !p.enabled || p.writer == nil {
		return nil
	}
	return p.writer.Close()
}
