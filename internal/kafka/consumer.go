package kafka

import (
	"context"

	segkafka "github.com/segmentio/kafka-go"
)

// Consumer is a thin wrapper around kafka-go's Reader that commits offsets
// after the caller has successfully handled each message.
type Consumer struct {
	reader *segkafka.Reader
}

// NewConsumer constructs a Consumer for the given brokers, topic, and group.
func NewConsumer(brokers []string, topic, group string) *Consumer {
	return &Consumer{
		reader: segkafka.NewReader(segkafka.ReaderConfig{
			Brokers:  brokers,
			Topic:    topic,
			GroupID:  group,
			MinBytes: 1,
			MaxBytes: 10 << 20, // 10 MiB is plenty for our small JSON payloads
		}),
	}
}

// Consume calls handle for each message until ctx is cancelled or handle
// returns an error. Offsets are only committed after handle returns nil.
func (c *Consumer) Consume(ctx context.Context, handle func(ctx context.Context, key, value []byte) error) error {
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			return err
		}
		if err := handle(ctx, msg.Key, msg.Value); err != nil {
			return err
		}
		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			return err
		}
	}
}

// Close releases the underlying connection.
func (c *Consumer) Close() error { return c.reader.Close() }
