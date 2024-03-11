package rabbitmq

import (
	"fmt"
	"log"
	"time"

	"github.com/streadway/amqp"
)

type MessageHandler func(msg amqp.Delivery) error

// Consume for rabbitMQ messages , rpc mode
func ListenRabbitMQUsingRPC(rabbitMQArg RabbitMQArg, response RPCResponse, handleFunc func(msg amqp.Delivery, ch *amqp.Channel, response RPCResponse) error) error {
	var (
		conn *amqp.Connection
		ch   *amqp.Channel
		err  error
	)
	connStr := fmt.Sprintf("amqp://%s:%s@%s/", rabbitMQArg.Username, rabbitMQArg.Password, rabbitMQArg.Host)
	retryCount := 0  // 重試計數器
	maxRetries := 50 // 最大重試次數

	establishConnection := func() error {
		conn, err = amqp.Dial(connStr)
		if err != nil {
			return err
		}

		ch, err = conn.Channel()
		if err != nil {
			conn.Close()
			return err
		}

		err = ch.Qos(
			15,
			0,
			false,
		)
		if err != nil {
			ch.Close()
			conn.Close()
			return err
		}

		_, err = ch.QueueDeclare(
			rabbitMQArg.RabbitMQQueue,
			false,
			false,
			false,
			false,
			nil,
		)
		return err
	}

	for {
		if err = establishConnection(); err != nil {
			retryCount++
			log.Printf("Failed to connect to RabbitMQ (Retry %d/%d): %v", retryCount, maxRetries, err)
			if retryCount >= maxRetries {
				log.Printf("RabbitMQ Max retries %d reached. Exiting...", maxRetries)
				return err
			}

			time.Sleep(1 * time.Second)
			continue
		}

		// Reset retry counter after successful connection
		retryCount = 0

		// Start consuming messages
		msgs, err := ch.Consume(
			rabbitMQArg.RabbitMQQueue,
			"",
			false,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			log.Printf("Failed to consume from queue: %s", err)
			ch.Close()
			conn.Close()
			continue
		}

		closeChan := conn.NotifyClose(make(chan *amqp.Error))
		go func() {
			for {
				select {
				case <-closeChan:
					log.Printf("RabbitMQ connection closed unexpectedly. Attempting to reconnect...")
					for {
						// Attempt to reconnect
						if err := establishConnection(); err == nil {
							log.Printf("Reconnected to RabbitMQ successfully.")
							break
						}
						time.Sleep(time.Second)
					}
				}
			}
		}()

		for d := range msgs {

			go func(m amqp.Delivery) {

				if err := handleFunc(m, ch, response); err != nil {
					log.Printf("Handler error: %s", err)
					_ = m.Nack(false, true)
				}
				_ = m.Ack(false)
			}(d)
		}

		// If msgs channel is closed, loop will break and try to reconnect
	}

	// This will probably never be reached because of the infinite loop
	// You might need to add some logic to break out of the loop if necessary
	defer ch.Close()
	defer conn.Close()
	return nil
}

func ListenRabbitMQUsingRPCForInterval(rabbitMQArg RabbitMQArg, response RPCResponse, handleFunc func(msg amqp.Delivery, ch *amqp.Channel, response RPCResponse) error) error {
	conn, err := amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s/", rabbitMQArg.Username, rabbitMQArg.Password, rabbitMQArg.Host))
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close() // 确保连接被关闭

	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open a channel: %v", err)
	}
	defer ch.Close() // 确保通道被关闭

	msgs, err := ch.Consume(
		rabbitMQArg.RabbitMQQueue,
		"",
		false, // auto-ack 设置为false，我们将不自动发送ACK
		false, // exclusive
		false, // no-local
		true,  // no-wait 设置为true，不等待消息就断开连接
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to register a consumer: %v", err)
	}

	log.Printf("Connected to RabbitMQ, checking for messages...")
	select {
	case d, ok := <-msgs:
		if !ok {
			log.Println("No messages in queue, disconnecting...")
			return nil
		}
		log.Println("Message received, processing...")
		if err := handleFunc(d, ch, response); err != nil {
			log.Printf("Error handling message: %v", err)
		}
		// 不发送ACK，让消息依赖于TTL
		return nil
	case <-time.After(5 * time.Second): // 等待一定时间，如果没有消息则断开连接
		log.Println("No messages received within timeout period, disconnecting...")
		return nil
	}
}
