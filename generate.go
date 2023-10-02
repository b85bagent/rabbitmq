package rabbitmq

type RabbitMQArg struct {
	Host               string
	Username           string
	Password           string
	RabbitMQExchange   string
	RabbitMQRoutingKey string
	RabbitMQQueue      string
}
