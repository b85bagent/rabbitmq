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

	retryCount := 0  // 加入一個重試計數器
	maxRetries := 50 // 您可以設定您希望的最大重試次數

	for {

		signalChan := make(chan bool, 1) // 每次循環開始時都重新創建一個 signalChan

		connStr := fmt.Sprintf("amqp://%s:%s@%s/", rabbitMQArg.Username, rabbitMQArg.Password, rabbitMQArg.Host)

		conn, err := amqp.Dial(connStr)
		if err != nil {
			retryCount++ // 增加重試計數
			log.Printf("Failed to connect to RabbitMQ (Retry %d/%d): %v", retryCount, maxRetries, err)

			if retryCount > maxRetries { // 檢查是否超過最大重試次數
				log.Printf("rabbitMQ Max retries %d reached. Exiting...", maxRetries)
				break
			}

			time.Sleep(1 * time.Second) // 等待 1 秒再嘗試重新連接
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

		// 设置预取计数
		err = ch.Qos(
			15,    // 预取计数设置为1，你可以根据需要增加这个数值
			0,     // prefetchSize - 按照AMQP 0-9-1标准，prefetchSize必须设置为0
			false, // 是否将这个设置应用于整个Channel，false表示只对当前消费者有效
		)
		if err != nil {
			log.Fatalf("Failed to set channel QoS: %s", err)
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
				conn.Close()
				break outerLoop
			case msg, ok := <-msgs:
				if !ok {
					break outerLoop
				}
				go func(m amqp.Delivery) { // 在goroutine中处理消息
					if err := handleFunc(m, ch, response); err != nil {
						log.Printf("Handler error: %s", err)
					}
					// 确认消息处理完成，可以选择在handleFunc内部确认
					m.Ack(false)
				}(msg)
			}
		}

		ch.Close()
		conn.Close()
		time.Sleep(time.Second) // 等待 1 秒再嘗試重新連接
	}

	return nil
}
