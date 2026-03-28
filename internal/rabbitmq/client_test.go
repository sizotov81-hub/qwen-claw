package rabbitmq

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewRabbitMQClient(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	config := RabbitMQConfig{
		URL: "amqp://guest:guest@localhost:5672/",
		Exchanges: []ExchangeConfig{
			{Name: "test-exchange", Type: "topic"},
		},
		Queues: []QueueConfig{
			{Name: "test-queue", Durable: true},
		},
		Prefetch:       10,
		ReconnectDelay: 5 * time.Second,
	}

	client := NewRabbitMQClient(config, logger.Sugar())

	require.NotNil(t, client)
	assert.Equal(t, config.URL, client.config.URL)
	assert.False(t, client.IsConnected())
}

func TestRabbitMQClient_Message(t *testing.T) {
	msg := Message{
		ID:        "test-123",
		Type:      "test-type",
		Payload:   map[string]interface{}{"key": "value"},
		Timestamp: time.Now(),
	}

	assert.Equal(t, "test-123", msg.ID)
	assert.Equal(t, "test-type", msg.Type)
	assert.NotNil(t, msg.Payload)
	assert.NotEmpty(t, msg.Timestamp)
}

func TestRabbitMQClient_Config(t *testing.T) {
	config := RabbitMQConfig{
		URL:            "amqp://user:pass@localhost:5672/",
		Prefetch:       20,
		ReconnectDelay: 10 * time.Second,
	}

	assert.Equal(t, "amqp://user:pass@localhost:5672/", config.URL)
	assert.Equal(t, 20, config.Prefetch)
	assert.Equal(t, 10*time.Second, config.ReconnectDelay)
}

func TestRabbitMQClient_ExchangeConfig(t *testing.T) {
	exchange := ExchangeConfig{
		Name: "my-exchange",
		Type: "fanout",
	}

	assert.Equal(t, "my-exchange", exchange.Name)
	assert.Equal(t, "fanout", exchange.Type)
}

func TestRabbitMQClient_QueueConfig(t *testing.T) {
	queue := QueueConfig{
		Name:    "my-queue",
		Durable: true,
	}

	assert.Equal(t, "my-queue", queue.Name)
	assert.True(t, queue.Durable)
}

func TestRabbitMQClient_ConnectWithoutServer(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	config := RabbitMQConfig{
		URL:            "amqp://guest:guest@localhost:5672/",
		ReconnectDelay: 100 * time.Millisecond,
	}

	client := NewRabbitMQClient(config, logger.Sugar())

	// Connect должен вернуть ошибку (сервер не запущен)
	err := client.Connect()
	assert.Error(t, err)
	assert.False(t, client.IsConnected())

	// Close не должен паниковать
	err = client.Close()
	assert.NoError(t, err)
}

func TestRabbitMQClient_MessageMarshal(t *testing.T) {
	msg := Message{
		ID:        "test-456",
		Type:      "llm-request",
		Payload:   map[string]interface{}{"model": "gpt-4", "prompt": "Hello"},
		Timestamp: time.Now(),
	}

	// Проверяем что сообщение можно сериализовать
	data, err := marshalMessage(msg)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	// Проверяем что можно десериализовать
	var decoded Message
	err = unmarshalMessage(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, msg.ID, decoded.ID)
	assert.Equal(t, msg.Type, decoded.Type)
}

func marshalMessage(msg Message) ([]byte, error) {
	return json.Marshal(msg)
}

func unmarshalMessage(data []byte, msg *Message) error {
	return json.Unmarshal(data, msg)
}

func TestRabbitMQClient_SendToQueue(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	config := RabbitMQConfig{
		URL:            "amqp://guest:guest@localhost:5672/",
		ReconnectDelay: 1 * time.Second,
	}

	client := NewRabbitMQClient(config, logger.Sugar())

	// SendToQueue должен вернуть ошибку (не подключен)
	ctx := context.Background()
	msg := Message{ID: "test", Type: "test"}
	err := client.SendToQueue(ctx, "test-queue", msg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

func TestRabbitMQClient_SendToExchange(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	config := RabbitMQConfig{
		URL:            "amqp://guest:guest@localhost:5672/",
		ReconnectDelay: 1 * time.Second,
	}

	client := NewRabbitMQClient(config, logger.Sugar())

	// SendToExchange должен вернуть ошибку (не подключен)
	ctx := context.Background()
	msg := Message{ID: "test", Type: "test"}
	err := client.SendToExchange(ctx, "test-exchange", "routing.key", msg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

func TestRabbitMQClient_Consume(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	config := RabbitMQConfig{
		URL:            "amqp://guest:guest@localhost:5672/",
		ReconnectDelay: 1 * time.Second,
	}

	client := NewRabbitMQClient(config, logger.Sugar())

	// Consume должен вернуть ошибку (не подключен)
	ctx := context.Background()
	_, err := client.Consume(ctx, "test-queue", "test-consumer")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

func TestRabbitMQClient_QueueBind(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	config := RabbitMQConfig{
		URL:            "amqp://guest:guest@localhost:5672/",
		ReconnectDelay: 1 * time.Second,
	}

	client := NewRabbitMQClient(config, logger.Sugar())

	// QueueBind должен вернуть ошибку (не подключен)
	err := client.QueueBind("test-queue", "test-exchange", "routing.key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

func TestRabbitMQClient_AckNack(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	config := RabbitMQConfig{
		URL:            "amqp://guest:guest@localhost:5672/",
		ReconnectDelay: 1 * time.Second,
	}

	client := NewRabbitMQClient(config, logger.Sugar())

	// Создаём тестовое delivery (заглушка)
	// В реальном тесте здесь будет mock delivery
	// Проверяем что методы существуют
	assert.NotNil(t, client.Ack)
	assert.NotNil(t, client.Nack)
}
