package rabbitmq

import (
	"fmt"
	"log"
	"sync"
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
		wg   *sync.WaitGroup
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
			wg.Add(1)
			go func(m amqp.Delivery) {
				defer wg.Done()
				if err := handleFunc(m, ch, response); err != nil {
					log.Printf("Handler error: %s", err)
					_ = m.Nack(false, true)
				}
				_ = m.Ack(false)
			}(d)
		}

		wg.Wait()

		// If msgs channel is closed, loop will break and try to reconnect
	}

	// This will probably never be reached because of the infinite loop
	// You might need to add some logic to break out of the loop if necessary
	defer ch.Close()
	defer conn.Close()
	return nil
}
