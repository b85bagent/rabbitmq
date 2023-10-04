package rabbitmq

import (
	"fmt"
	"log"
	"time"

	"github.com/streadway/amqp"
)

type MessageHandler func(msg amqp.Delivery) error

// Consume for rabbitMQ messages , rpc mode
func ListenRabbitMQUsingRPC(rabbitMQArg RabbitMQArg, response string, handleFunc func(msg amqp.Delivery, ch *amqp.Channel, response string) error) error {

	retryCount := 0                  // 加入一個重試計數器
	maxRetries := 50                 // 您可以設定您希望的最大重試次數
	signalChan := make(chan bool, 1) // 用于发送信号的通道

	for {
		connStr := fmt.Sprintf("amqp://%s:%s@%s/", rabbitMQArg.Username, rabbitMQArg.Password, rabbitMQArg.Host)

		conn, err := amqp.Dial(connStr)
		if err != nil {
			log.Printf("Failed to connect to RabbitMQ: %v", err)
			retryCount++                 // 增加重試計數
			if retryCount > maxRetries { // 檢查是否超過最大重試次數
				log.Printf("rabbitMQ Max retries %d reached. Exiting...", maxRetries)
				break
			}
			time.Sleep(5 * time.Second) // 等待 5 秒再嘗試重新連接
			continue

		}

		// 如果成功連接，則重設計數器
		retryCount = 0

		ch, err := conn.Channel() // 建立一個新的 channel
		if err != nil {
			log.Printf("Failed to open a channel: %s", err)
			conn.Close()
			continue
		}

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
			log.Printf("Failed to declare a queue: %s", err)
			ch.Close()
			conn.Close()
			continue
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
			log.Printf("Failed to consume from queue: %s", err)
			ch.Close()
			conn.Close()
			continue
		}

		closeChan := conn.NotifyClose(make(chan *amqp.Error))
		go func() {
			err := <-closeChan
			log.Printf("RabbitMQ connection closed: %v", err)
			signalChan <- true // 发送一个信号，让外部循环继续
		}()

	outerLoop:
		for {
			select {
			case <-signalChan: // 接收到信号时，退出select，继续外部循环
				ch.Close()
				conn.Close()
				break outerLoop
			case msg, ok := <-msgs:
				if !ok {
					break outerLoop
				}
				// 调用传递进来的 handler 函数处理消息
				if err := handleFunc(msg, ch, response); err != nil {
					log.Printf("Handler error: %s", err)
				}
			}
		}

		ch.Close()
		conn.Close()
		time.Sleep(5 * time.Second) // 等待 5 秒再嘗試重新連接
	}

	return nil
}
