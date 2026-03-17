package rabbitmq

import (
	"fmt"
	"os"

	"pengi-med-saas/core/logger"

	"github.com/gin-gonic/gin"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

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
		host = "pengi-rabbitmq"
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

func StartRabbitMQWithChannel() (*amqp.Connection, *amqp.Channel, error) {
	conn, err := StartRabbitMQ()
	if err != nil {
		return nil, nil, err
	}

	channel, err := GetChannelMQ(conn)
	if err != nil {
		conn.Close()
		return nil, nil, err
	}

	return conn, channel, nil
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

func PublishMessage(channel *amqp.Channel, queueName string, body []byte) error {
	err := channel.Publish(
		"",        // exchange
		queueName, // routing key
		false,     // mandatory
		false,     // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
	return err
}

func StartConsumer(channel *amqp.Channel, queueName string, callback func([]byte) error) {
	go func() {
		msgs, err := channel.Consume(
			queueName, // queue
			"",        // consumer
			true,      // auto-ack
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
			if err := callback(msg.Body); err != nil {
				logger.Log.Warn("Error processing queue message",
					zap.String("queue", queueName),
					zap.Error(err),
				)
			}
		}
	}()
}

// GetChannel retrieves a RabbitMQ channel injected into the Gin context
func GetChannel(c *gin.Context, key string) *amqp.Channel {
	ch, exists := c.Get(key)
	if !exists {
		return nil
	}
	return ch.(*amqp.Channel)
}
