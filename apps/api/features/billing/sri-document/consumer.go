package sri_document

import (
	"encoding/json"
	"fmt"

	"pengi-med-saas/core/brokers/rabbitmq"

	amqp "github.com/rabbitmq/amqp091-go"
)

// StartConsumer declares kind's queue (with its retry ladder and dead-letter
// queue) and consumes its tasks on ch.
func (l *Lifecycle) StartConsumer(ch *amqp.Channel, kind Kind) error {
	q, err := rabbitmq.DeclareQueueWithRetry(ch, kind.Queue)
	if err != nil {
		return fmt.Errorf("declare %s: %w", kind.Queue, err)
	}
	rabbitmq.StartConsumer(ch, q.Name, l.HandleTask(kind), IsRetryable)
	return nil
}

// HandleTask decodes one queued task of kind and processes its document.
func (l *Lifecycle) HandleTask(kind Kind) func(body []byte) error {
	return func(body []byte) error {
		var msg map[string]uint64
		if err := json.Unmarshal(body, &msg); err != nil {
			return fmt.Errorf("decode %s task: %w", kind.Name, err)
		}
		id, ok := msg[kind.MessageIDField]
		if !ok {
			return fmt.Errorf("decode %s task: missing %q", kind.Name, kind.MessageIDField)
		}
		return l.Process(kind, id)
	}
}
