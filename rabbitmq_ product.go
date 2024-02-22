package rabbitmq

import (
	"fmt"
	"log"
	"sync"

	"github.com/streadway/amqp"
)

type RabbitMQClient struct {
	rabbitMQArg RabbitMQArg
	conn        *amqp.Connection
	ch          *amqp.Channel
	mu          sync.Mutex
}

func NewRabbitMQClient(rabbitMQArg RabbitMQArg) (*RabbitMQClient, error) {
	conn, ch, err := createConnectionAndChannel(rabbitMQArg)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection or channel: %v", err)
	}

	return &RabbitMQClient{
		rabbitMQArg: rabbitMQArg,
		conn:        conn,
		ch:          ch,
	}, nil
}

func (c *RabbitMQClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ch != nil {
		c.ch.Close()
		c.ch = nil
	}
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
}

func (c *RabbitMQClient) SendMessage(data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := ensureExchangeExists(c.ch, c.conn, c.rabbitMQArg); err != nil {
		return logAndReturnError(fmt.Sprintf("failed to ensure exchange exists: %v", err))
	}

	// 新增：確保Queue存在
	if err := c.EnsureQueueAndBind(); err != nil {
		return logAndReturnError(fmt.Sprintf("failed to ensure queue exists: %v", err))
	}

	if err := publishMessage(c.ch, c.rabbitMQArg, data); err != nil {
		return logAndReturnError(fmt.Sprintf("failed to publish a message: %v", err))
	}

	return nil
}

func createConnectionAndChannel(rabbitMQArg RabbitMQArg) (*amqp.Connection, *amqp.Channel, error) {
	connStr := fmt.Sprintf("amqp://%s:%s@%s/", rabbitMQArg.Username, rabbitMQArg.Password, rabbitMQArg.Host)
	conn, err := amqp.Dial(connStr)
	if err != nil {
		return nil, nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, nil, err
	}
	return conn, ch, nil
}

func ensureExchangeExists(ch *amqp.Channel, conn *amqp.Connection, rabbitMQArg RabbitMQArg) error {
	// 創建/確認你的 exchange 存在
	err := ch.ExchangeDeclare(
		rabbitMQArg.RabbitMQExchange, // name
		"direct",                     // type
		false,                        // durable
		false,                        // auto-deleted
		false,                        // internal
		false,                        // no-wait
		nil,                          // arguments
	)

	if err != nil {
		// Check if the error is a PRECONDITION_FAILED error
		if amqpErr, ok := err.(*amqp.Error); ok && amqpErr.Code == amqp.PreconditionFailed {

			// Recreate channel as it might have been closed due to the error
			ch, err = conn.Channel()
			if err != nil {
				log.Printf("Failed to open a new channel: %s", err)
				return err
			}
			defer ch.Close()

			// Try declaring the exchange with durable set to true
			err = ch.ExchangeDeclare(
				rabbitMQArg.RabbitMQExchange, // name
				"direct",                     // type
				true,                         // durable
				false,                        // auto-deleted
				false,                        // internal
				false,                        // no-wait
				nil,                          // arguments
			)
			if err != nil {
				log.Printf("Failed to declare an exchange with durable set to true: %s", err)
				return err
			}
		} else {
			log.Printf("Failed to declare an exchange: %s", err)
			return err
		}
	}
	return nil
}

func publishMessage(ch *amqp.Channel, rabbitMQArg RabbitMQArg, data []byte) error {
	// 發送訊息到你的 exchange
	err := ch.Publish(
		rabbitMQArg.RabbitMQExchange,   // exchange name
		rabbitMQArg.RabbitMQRoutingKey, // routing key
		false,                          // mandatory
		false,                          // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        data, // 這是你想要發送的資料
		})

	if err != nil {
		log.Printf("Failed to publish a message: %s", err)
		return err
	}
	return nil
}

func (c *RabbitMQClient) EnsureQueueAndBind() error {
	// 确保Queue存在
	_, err := c.ch.QueueDeclare(
		c.rabbitMQArg.RabbitMQRoutingKey, // 这里使用RoutingKey作为Queue名称
		true,                             // durable
		false,                            // delete when unused
		false,                            // exclusive
		false,                            // no-wait
		nil,                              // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare a queue: %v", err)
	}

	// 绑定Queue到Exchange
	err = c.ch.QueueBind(
		c.rabbitMQArg.RabbitMQRoutingKey, // queue name
		c.rabbitMQArg.RabbitMQRoutingKey, // routing key
		c.rabbitMQArg.RabbitMQExchange,   // exchange
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to bind a queue to exchange: %v", err)
	}

	return nil
}

func closeResources(conn *amqp.Connection, ch *amqp.Channel) {
	ch.Close()
	conn.Close()
}

func logAndReturnError(errMsg string) error {
	log.Println(errMsg)
	return fmt.Errorf(errMsg)
}
