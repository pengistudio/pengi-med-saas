package rabbitmq

import (
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	"sync/atomic"
	"time"

	"pengi-med-saas/core/logger"

	"github.com/gin-gonic/gin"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// retryDelays is the backoff ladder for retryable failures, one TTL queue per step:
// RabbitMQ only expires messages at the head of a queue, so mixing TTLs in a single
// queue would let a long delay hold back shorter ones queued behind it.
var retryDelays = []time.Duration{10 * time.Second, time.Minute, 5 * time.Minute}

const retryCountHeader = "x-retry-count"

var errPanic = errors.New("queue handler panicked")

var publishChannel atomic.Pointer[amqp.Channel]

func getRabbitMQConnectionString() string {
	user := os.Getenv("RABBITMQ_USER")
	if user == "" {
		user = "guest"
	}
	password := os.Getenv("RABBITMQ_PASSWORD")
	if password == "" {
		password = "guest"
	}
	host := os.Getenv("RABBITMQ_HOST")
	if host == "" {
		host = "rabbitmq"
	}
	port := os.Getenv("RABBITMQ_PORT")
	if port == "" {
		port = "5672"
	}
	return fmt.Sprintf("amqp://%s:%s@%s:%s/", user, password, host, port)
}

func StartRabbitMQ() (*amqp.Connection, error) {
	conn, err := amqp.Dial(getRabbitMQConnectionString())
	if err != nil {
		logger.Log.Error("Failed to connect to RabbitMQ", zap.Error(err))
		return nil, err
	}

	logger.Log.Info("🐇✅ Connected to RabbitMQ successfully")
	return conn, nil
}

func GetChannelMQ(conn *amqp.Connection) (*amqp.Channel, error) {
	channel, err := conn.Channel()
	if err != nil {
		logger.Log.Error("Failed to open RabbitMQ channel", zap.Error(err))
		return nil, err
	}
	return channel, nil
}

// PublishChannel returns the channel HTTP handlers publish on, or nil while
// RabbitMQ is disconnected.
func PublishChannel() *amqp.Channel {
	return publishChannel.Load()
}

// Run keeps a RabbitMQ connection alive for the life of the process. It dials with
// backoff, opens the shared publish channel, then gives each init its own channel
// (a consumer must not share its channel — see the note in cmd/main.go). When the
// connection or any of those channels closes, everything is torn down and rebuilt.
// Blocks forever; run it in a goroutine.
func Run(inits ...func(ch *amqp.Channel) error) {
	backoff := time.Second
	for {
		if err := runConnection(inits); err != nil {
			logger.Log.Warn("RabbitMQ unavailable, retrying", zap.Duration("in", backoff), zap.Error(err))
			time.Sleep(backoff)
			backoff = min(backoff*2, 30*time.Second)
			continue
		}
		backoff = time.Second
	}
}

// runConnection returns nil if the connection came up and later dropped, or the
// error that prevented it from coming up at all.
func runConnection(inits []func(ch *amqp.Channel) error) error {
	conn, err := StartRabbitMQ()
	if err != nil {
		return err
	}
	defer conn.Close()

	lost := make(chan error, len(inits)+2)
	watch := func(closed chan *amqp.Error) {
		go func() {
			if reason, ok := <-closed; ok {
				lost <- reason
			} else {
				lost <- errors.New("closed")
			}
		}()
	}
	watch(conn.NotifyClose(make(chan *amqp.Error, 1)))

	pubCh, err := GetChannelMQ(conn)
	if err != nil {
		return err
	}
	watch(pubCh.NotifyClose(make(chan *amqp.Error, 1)))

	for _, init := range inits {
		ch, err := GetChannelMQ(conn)
		if err != nil {
			return err
		}
		watch(ch.NotifyClose(make(chan *amqp.Error, 1)))
		if err := init(ch); err != nil {
			return err
		}
	}

	publishChannel.Store(pubCh)
	reason := <-lost
	publishChannel.Store(nil)
	logger.Log.Warn("RabbitMQ connection lost", zap.Error(reason))
	return nil
}

func DeclareQueue(channel *amqp.Channel, queueName string) (amqp.Queue, error) {
	queue, err := channel.QueueDeclare(
		queueName,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return amqp.Queue{}, err
	}
	return queue, nil
}

// DeclareQueueWithRetry declares queueName plus its retry ladder and dead-letter
// queue. The main queue keeps nil arguments on purpose: queue arguments are
// immutable, and redeclaring an existing queue with new ones fails with
// PRECONDITION_FAILED.
func DeclareQueueWithRetry(channel *amqp.Channel, queueName string) (amqp.Queue, error) {
	queue, err := DeclareQueue(channel, queueName)
	if err != nil {
		return amqp.Queue{}, err
	}
	if _, err := DeclareQueue(channel, deadLetterQueueName(queueName)); err != nil {
		return amqp.Queue{}, err
	}
	for _, delay := range retryDelays {
		_, err := channel.QueueDeclare(retryQueueName(queueName, delay), true, false, false, false, amqp.Table{
			"x-message-ttl":             delay.Milliseconds(),
			"x-dead-letter-exchange":    "",
			"x-dead-letter-routing-key": queueName,
		})
		if err != nil {
			return amqp.Queue{}, err
		}
	}
	return queue, nil
}

// The TTL is part of the name because a retry queue's TTL can't change once declared.
func retryQueueName(queueName string, delay time.Duration) string {
	return fmt.Sprintf("%s.retry.%ds", queueName, int(delay.Seconds()))
}

func deadLetterQueueName(queueName string) string {
	return queueName + ".dlq"
}

func PublishMessage(channel *amqp.Channel, queueName string, body []byte) error {
	err := channel.Publish(
		"",        // exchange
		queueName, // routing key
		false,     // mandatory
		false,     // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	)
	return err
}

// StartConsumer consumes queueName with manual acks. A failed message is re-queued
// through the retry ladder when isRetryable says so, and parked in the dead-letter
// queue otherwise (or once retries are exhausted). Panics are recovered per message
// and dead-lettered without retry.
func StartConsumer(channel *amqp.Channel, queueName string, handler func([]byte) error, isRetryable func(error) bool) {
	go func() {
		if err := channel.Qos(1, 0, false); err != nil {
			logger.Log.Error("RabbitMQ Qos Error", zap.String("queue", queueName), zap.Error(err))
			return
		}

		msgs, err := channel.Consume(
			queueName, // queue
			"",        // consumer
			false,     // auto-ack
			false,     // exclusive
			false,     // no-local
			false,     // no-wait
			nil,       // args
		)

		if err != nil {
			logger.Log.Error("RabbitMQ Consume Error", zap.String("queue", queueName), zap.Error(err))
			return
		}

		for msg := range msgs {
			settle(channel, queueName, msg, handler, isRetryable)
		}
	}()
}

func settle(channel *amqp.Channel, queueName string, msg amqp.Delivery, handler func([]byte) error, isRetryable func(error) bool) {
	err := runHandler(queueName, handler, msg.Body)
	if err == nil {
		if ackErr := msg.Ack(false); ackErr != nil {
			logger.Log.Error("Failed to ack queue message", zap.String("queue", queueName), zap.Error(ackErr))
		}
		return
	}

	attempt := retryCount(msg.Headers)
	headers := amqp.Table{}
	for k, v := range msg.Headers {
		headers[k] = v
	}

	var target string
	if !errors.Is(err, errPanic) && isRetryable(err) && attempt < len(retryDelays) {
		target = retryQueueName(queueName, retryDelays[attempt])
		headers[retryCountHeader] = int32(attempt + 1)
		logger.Log.Warn("Queue message failed, scheduling retry",
			zap.String("queue", queueName),
			zap.Int("attempt", attempt+1),
			zap.Duration("delay", retryDelays[attempt]),
			zap.Error(err),
		)
	} else {
		target = deadLetterQueueName(queueName)
		headers["x-error"] = err.Error()
		headers["x-failed-at"] = time.Now().UTC().Format(time.RFC3339)
		logger.Log.Error("Queue message failed, moving to dead-letter queue",
			zap.String("queue", queueName),
			zap.Int("retries", attempt),
			zap.Error(err),
		)
	}

	pubErr := channel.Publish("", target, false, false, amqp.Publishing{
		ContentType:  msg.ContentType,
		DeliveryMode: amqp.Persistent,
		Headers:      headers,
		Body:         msg.Body,
	})
	if pubErr != nil {
		logger.Log.Error("Failed to re-route queue message, requeueing", zap.String("queue", queueName), zap.String("target", target), zap.Error(pubErr))
		_ = msg.Nack(false, true)
		return
	}
	if ackErr := msg.Ack(false); ackErr != nil {
		logger.Log.Error("Failed to ack queue message", zap.String("queue", queueName), zap.Error(ackErr))
	}
}

func runHandler(queueName string, handler func([]byte) error, body []byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			logger.Log.Error("Queue handler panicked",
				zap.String("queue", queueName),
				zap.Any("panic", r),
				zap.ByteString("stack", debug.Stack()),
			)
			err = fmt.Errorf("%w: %v", errPanic, r)
		}
	}()
	return handler(body)
}

func retryCount(headers amqp.Table) int {
	switch v := headers[retryCountHeader].(type) {
	case int32:
		return int(v)
	case int64:
		return int(v)
	case int:
		return v
	}
	return 0
}

// GetChannel retrieves a RabbitMQ channel injected into the Gin context
func GetChannel(c *gin.Context, key string) *amqp.Channel {
	ch, exists := c.Get(key)
	if !exists {
		return nil
	}
	return ch.(*amqp.Channel)
}
