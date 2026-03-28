// Package rabbitmq предоставляет клиент для RabbitMQ
package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// Message сообщение для отправки в очередь
type Message struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Payload   map[string]interface{} `json:"payload"`
	Timestamp time.Time              `json:"timestamp"`
}

// RabbitMQConfig конфигурация RabbitMQ
type RabbitMQConfig struct {
	URL         string
	Exchanges   []ExchangeConfig
	Queues      []QueueConfig
	Prefetch    int
	ReconnectDelay time.Duration
}

// ExchangeConfig конфигурация exchange
type ExchangeConfig struct {
	Name string
	Type string // direct, fanout, topic, headers
}

// QueueConfig конфигурация очереди
type QueueConfig struct {
	Name    string
	Durable bool
}

// RabbitMQClient клиент для RabbitMQ
type RabbitMQClient struct {
	mu            sync.RWMutex
	conn          *amqp.Connection
	channel       *amqp.Channel
	config        RabbitMQConfig
	logger        *zap.SugaredLogger
	reconnecting  bool
	stopChan      chan struct{}
	messageChan   <-chan amqp.Delivery
}

// NewRabbitMQClient создаёт новый RabbitMQ клиент
func NewRabbitMQClient(config RabbitMQConfig, logger *zap.SugaredLogger) *RabbitMQClient {
	return &RabbitMQClient{
		config:   config,
		logger:   logger,
		stopChan: make(chan struct{}),
	}
}

// Connect подключается к RabbitMQ
func (r *RabbitMQClient) Connect() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var err error
	r.conn, err = amqp.Dial(r.config.URL)
	if err != nil {
		return fmt.Errorf("dial rabbitmq: %w", err)
	}

	r.channel, err = r.conn.Channel()
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}

	// Настраиваем prefetch
	if err := r.channel.Qos(r.config.Prefetch, 0, false); err != nil {
		return fmt.Errorf("qos: %w", err)
	}

	// Создаём exchanges
	for _, ex := range r.config.Exchanges {
		if err := r.channel.ExchangeDeclare(
			ex.Name,
			ex.Type,
			true,  // durable
			false, // auto-deleted
			false, // internal
			false, // no-wait
			nil,   // arguments
		); err != nil {
			return fmt.Errorf("declare exchange %s: %w", ex.Name, err)
		}
	}

	// Создаём queues
	for _, q := range r.config.Queues {
		if _, err := r.channel.QueueDeclare(
			q.Name,
			q.Durable,
			false, // auto-deleted
			false, // exclusive
			false, // no-wait
			nil,   // arguments
		); err != nil {
			return fmt.Errorf("declare queue %s: %w", q.Name, err)
		}
	}

	// Запускаем reconnect loop
	go r.reconnectLoop()

	r.logger.Info("Connected to RabbitMQ")
	return nil
}

// Close закрывает соединение
func (r *RabbitMQClient) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	close(r.stopChan)

	if r.channel != nil {
		if err := r.channel.Close(); err != nil {
			r.logger.Errorf("Close channel error: %v", err)
		}
	}

	if r.conn != nil {
		if err := r.conn.Close(); err != nil {
			r.logger.Errorf("Close connection error: %v", err)
		}
	}

	r.logger.Info("Disconnected from RabbitMQ")
	return nil
}

// Publish публикует сообщение в exchange
func (r *RabbitMQClient) Publish(ctx context.Context, exchange, routingKey string, message Message) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.channel == nil {
		return fmt.Errorf("not connected")
	}

	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	if err := r.channel.PublishWithContext(
		ctx,
		exchange,
		routingKey,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Timestamp:    time.Now(),
			Body:         body,
		},
	); err != nil {
		return fmt.Errorf("publish: %w", err)
	}

	return nil
}

// Consume подписывается на очередь
func (r *RabbitMQClient) Consume(ctx context.Context, queue, consumer string) (<-chan Message, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.channel == nil {
		return nil, fmt.Errorf("not connected")
	}

	deliveries, err := r.channel.Consume(
		queue,
		consumer,
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("consume: %w", err)
	}

	r.messageChan = deliveries

	messageChan := make(chan Message)
	go func() {
		for d := range deliveries {
			var msg Message
			if err := json.Unmarshal(d.Body, &msg); err != nil {
				r.logger.Errorf("Unmarshal message error: %v", err)
				continue
			}

			select {
			case messageChan <- msg:
				// Сообщение отправлено
				if err := d.Ack(false); err != nil {
					r.logger.Errorf("Ack error: %v", err)
				}
			case <-ctx.Done():
				return
			case <-r.stopChan:
				return
			}
		}
	}()

	return messageChan, nil
}

// QueueBind связывает очередь с exchange
func (r *RabbitMQClient) QueueBind(queue, exchange, routingKey string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.channel == nil {
		return fmt.Errorf("not connected")
	}

	if err := r.channel.QueueBind(
		queue,
		routingKey,
		exchange,
		false, // no-wait
		nil,   // arguments
	); err != nil {
		return fmt.Errorf("bind queue: %w", err)
	}

	return nil
}

// IsConnected проверяет подключение
func (r *RabbitMQClient) IsConnected() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.conn != nil && !r.conn.IsClosed()
}

// reconnectLoop автоматически переподключается при разрыве
func (r *RabbitMQClient) reconnectLoop() {
	for {
		select {
		case <-r.stopChan:
			return
		case <-time.After(r.config.ReconnectDelay):
			if !r.IsConnected() {
				r.mu.Lock()
				if !r.reconnecting {
					r.reconnecting = true
					r.mu.Unlock()

					r.logger.Info("Attempting to reconnect to RabbitMQ...")
					if err := r.Connect(); err != nil {
						r.logger.Errorf("Reconnect error: %v", err)
					} else {
						r.logger.Info("Reconnected to RabbitMQ")
					}

					r.mu.Lock()
					r.reconnecting = false
				}
				r.mu.Unlock()
			}
		}
	}
}

// SendToQueue отправляет сообщение в очередь
func (r *RabbitMQClient) SendToQueue(ctx context.Context, queue string, message Message) error {
	return r.Publish(ctx, "", queue, message)
}

// SendToExchange отправляет сообщение в exchange
func (r *RabbitMQClient) SendToExchange(ctx context.Context, exchange, routingKey string, message Message) error {
	return r.Publish(ctx, exchange, routingKey, message)
}

// Ack подтверждает сообщение
func (r *RabbitMQClient) Ack(d amqp.Delivery) error {
	return d.Ack(false)
}

// Nack отрицает сообщение
func (r *RabbitMQClient) Nack(d amqp.Delivery, requeue bool) error {
	return d.Nack(false, requeue)
}
