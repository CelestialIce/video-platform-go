// internal/dal/mq.go
package dal

import (
	"context"
	"log"
	"github.com/cjh/video-platform-go/internal/config"
	"github.com/rabbitmq/amqp091-go"
)

var (
	MQConn *amqp091.Connection
	MQChan *amqp091.Channel
)

// InitRabbitMQ 初始化 RabbitMQ 连接和通道
func InitRabbitMQ(cfg *config.Config) {
	var err error
	MQConn, err = amqp091.Dial(cfg.RabbitMQ.URL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}

	MQChan, err = MQConn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}

	// 声明一个队列，如果不存在则会创建
	// durable: true 表示队列在 RabbitMQ 重启后仍然存在
	_, err = MQChan.QueueDeclare(
		cfg.RabbitMQ.TranscodeQueue, // name
		true,                       // durable
		false,                      // delete when unused
		false,                      // exclusive
		false,                      // no-wait
		nil,                        // arguments
	)
	if err != nil {
		log.Fatalf("Failed to declare a queue: %v", err)
	}
	log.Println("RabbitMQ connection and queue declared")
}

// PublishTranscodeTask 发布转码任务到队列
func PublishTranscodeTask(ctx context.Context, body []byte) error {
	return MQChan.PublishWithContext(ctx,
		"", // exchange
		config.AppConfig.RabbitMQ.TranscodeQueue, // routing key (queue name)
		false, // mandatory
		false, // immediate
		amqp091.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
}