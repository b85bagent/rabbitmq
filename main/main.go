package main

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/b85bagent/rabbitmq"

	"github.com/streadway/amqp"
)

func main() {
	test_consumer()
	// test_product()

}

func test_consumer() {
	rabbitMQArg := rabbitmq.RabbitMQArg{
		Host:               "10.11.233.80:5672",
		Username:           "admin",
		Password:           "admin123",
		RabbitMQExchange:   "format",
		RabbitMQRoutingKey: "format",
		RabbitMQQueue:      "rpc-modules",
	}

	RPCResponse := rabbitmq.RPCResponse{}

	err := rabbitmq.ListenRabbitMQUsingRPC(rabbitMQArg, RPCResponse, func(msg amqp.Delivery, ch *amqp.Channel, RPCResponse rabbitmq.RPCResponse) error {

		log.Println(string(msg.Body))

		RPCResponse.Status = "Success"
		RPCResponse.StatusCode = 200
		t := make(map[string]interface{})
		t["message"] = "Agent get MQ message Successfully"

		RPCResponse.Response = t
		RPCResponse.Queue = rabbitMQArg.RabbitMQQueue
		RPCResponse.Timestamp = time.Now()

		err := SaveYAMLToFile(msg.Body, "./blackboxTest1.yaml")
		if err != nil {
			t := make(map[string]interface{})
			t["message"] = "Error writing to target.yaml:" + err.Error()
			log.Printf("Error writing to target.yaml: %v", err)
			RPCResponse.Status = "Failed"
			RPCResponse.StatusCode = 400
			RPCResponse.Response = t
			RPCResponse.Queue = rabbitMQArg.RabbitMQQueue
			RPCResponse.Timestamp = time.Now()
		}

		response, err := json.Marshal(RPCResponse)
		if err != nil {
			log.Printf("Error marshaling to JSON: %v\n", err)
			return err
		}

		// 發送回應到 reply_to 隊列
		errReplay := ch.Publish(
			"",          // exchange
			msg.ReplyTo, // routing key
			false,       // mandatory
			false,       // immediate
			amqp.Publishing{
				ContentType:   "text/plain",
				CorrelationId: msg.CorrelationId,
				Body:          response,
			})
		if errReplay != nil {
			log.Printf("Failed to publish a message Replay: %s", errReplay)
			return errReplay
		}

		// 發送 ack 確認消息已經被處理
		err = msg.Ack(false)
		if err != nil {
			log.Printf("Error acknowledging message : %s", err)
			return err
		}
		return nil
	})

	if err != nil {
		log.Fatal(err)
	}
}

func test_product() {
	rabbitMQArg := rabbitmq.RabbitMQArg{
		Host:               "10.11.233.80:5672",
		Username:           "admin",
		Password:           "admin123",
		RabbitMQExchange:   "lex-test-queue",
		RabbitMQRoutingKey: "lex-test-queue",
		RabbitMQQueue:      "rpc-modules",
	}
	data := []byte("Agent get MQ message Successfully")

	err := rabbitmq.SendMessageToRabbitMQ(rabbitMQArg, data)

	if err != nil {
		log.Fatal(err)
	}
}

func SaveYAMLToFile(content []byte, filepath string) error {

	err := os.WriteFile(filepath, []byte(content), 0644)
	if err != nil {
		log.Println("無法寫入檔案：", err)
		return err
	}

	log.Println("已成功儲存 YAML 檔案：", filepath)
	return nil
}
