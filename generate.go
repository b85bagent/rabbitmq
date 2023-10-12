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
	Status        string    `json:"status"`
	StatusCode    int       `json:"status_code"`
	Message       string    `json:"message"`
	CorrelationId string    `json:"correlation_id"`
	Timestamp     time.Time `json:"timestamp"`
	Queue         string    `json:"queue"`
}

const (
	Response_Failed       = "Failed"
	Response_Success      = "Success"
	Response_Failed_Code  = 400
	Response_Success_Code = 200
)
