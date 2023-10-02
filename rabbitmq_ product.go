package rabbitmq

import (
	"fmt"
	"log"

	"github.com/streadway/amqp"
)

func SendMessageToRabbitMQ(rabbitMQArg RabbitMQArg, data []byte) error {

	connStr := fmt.Sprintf("amqp://%s:%s@%s/", rabbitMQArg.Username, rabbitMQArg.Password, rabbitMQArg.Host)

	conn, err := amqp.Dial(connStr)
	if err != nil {
		log.Printf("Failed to connect to RabbitMQ: %v", err)
		return err
	}

	defer conn.Close()

	ch, err := conn.Channel() // 建立一個新的 channel
	if err != nil {
		log.Printf("Failed to open a channel: %s", err)
		return err
	}

	defer ch.Close() // 確保 channel 在結束時關閉

	// 創建/確認你的 exchange 存在
	err = ch.ExchangeDeclare(
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

	// 發送訊息到你的 exchange
	err = ch.Publish(
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
