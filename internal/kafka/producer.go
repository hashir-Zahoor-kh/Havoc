package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	segkafka "github.com/segmentio/kafka-go"
)

// Producer publishes JSON-encoded messages to a single Kafka topic.
type Producer struct {
	writer *segkafka.Writer
	topic  string
}

// NewProducer returns a Producer for the given brokers and topic.
func NewProducer(brokers []string, topic string) *Producer {
	return &Producer{
		writer: &segkafka.Writer{
			Addr:         segkafka.TCP(brokers...),
			Topic:        topic,
			Balancer:     &segkafka.Hash{},
			RequiredAcks: segkafka.RequireAll,
			BatchTimeout: 10 * time.Millisecond,
		},
		topic: topic,
	}
}

// Publish marshals v to JSON and writes it with the given key. An empty key
// causes the message to be randomly partitioned.
func (p *Producer) Publish(ctx context.Context, key string, v any) error {
	payload, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal %s payload: %w", p.topic, err)
	}
	msg := segkafka.Message{Value: payload}
	if key != "" {
		msg.Key = []byte(key)
	}
	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("write to %s: %w", p.topic, err)
	}
	return nil
}

// Close flushes pending writes and releases the underlying connection.
func (p *Producer) Close() error { return p.writer.Close() }
