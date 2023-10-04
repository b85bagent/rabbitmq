package rabbitmq

import "time"

type RabbitMQArg struct {
	Host               string
	Username           string
	Password           string
	RabbitMQExchange   string
	RabbitMQRoutingKey string
	RabbitMQQueue      string
}

type RPCResponse struct {
	Status     string    `json:"status"`
	StatusCode int       `json:"status_code"`
	Message    string    `json:"message"`
	Timestamp  time.Time `json:"timestamp"`
	Queue      string    `json:"queue"`
}

type OuterResponse struct {
	Detail RPCResponse `json:"detail"`
}
