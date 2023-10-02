package rabbitmq

import (
	"fmt"
	"log"

	"github.com/streadway/amqp"
)

type MessageHandler func(msg amqp.Delivery) error

// Consume for rabbitMQ messages , rpc mode
func ListenRabbitMQUsingRPC(rabbitMQArg RabbitMQArg, response string, handleFunc func(msg amqp.Delivery, ch *amqp.Channel, response string) error) error {

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

	// 聲明一个 Queue
	queue, err := ch.QueueDeclare(
		rabbitMQArg.RabbitMQQueue, // queue name
		false,                     // durable
		false,                     // delete when unused
		false,                     // exclusive
		false,                     // no-wait
		nil,                       // arguments
	)
	if err != nil {
		log.Fatal(err)
	}

	// 从 Queue 中消费消息
	msgs, err := ch.Consume(
		queue.Name,
		"",
		false, // 我們現在要手動發送 ack，所以設定 auto-ack 為 false
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatal(err)
	}

	for msg := range msgs {

		// 调用传递进来的 handler 函数处理消息
		if err := handleFunc(msg, ch, response); err != nil {
			log.Printf("Handler error: %s", err)
		}

	}

	return nil
}
