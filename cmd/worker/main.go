// cmd/worker/main.go
package main

import (
	"encoding/json"
	"log"

	"github.com/cjh/video-platform-go/internal/config"
	"github.com/cjh/video-platform-go/internal/dal"
	"github.com/cjh/video-platform-go/internal/service"
	"github.com/cjh/video-platform-go/internal/worker"
)

func main() {
	// 1. 初始化配置 (和 API Server 一样)
	config.Init()
	log.Println("Worker: Configuration loaded")

	// 2. 初始化所有连接 (数据库, MinIO, RabbitMQ)
	dal.InitMySQL(&config.AppConfig)
	dal.InitMinIO(&config.AppConfig)
	dal.InitRabbitMQ(&config.AppConfig) // Worker也需要连接MQ来消费
	log.Println("Worker: Database, MinIO and RabbitMQ initialized")

	// 3. 开始消费消息
	qName := config.AppConfig.RabbitMQ.TranscodeQueue
	msgs, err := dal.MQChan.Consume(
		qName, // queue
		"",    // consumer
		false, // auto-ack (!!!) 我们要手动确认
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		log.Fatalf("Failed to register a consumer: %v", err)
	}

	// 4. 使用一个 "forever" channel 来阻塞主 goroutine
	var forever chan struct{}

	go func() {
		for d := range msgs {
			log.Printf("Received a message: %s", d.Body)

			var task service.TranscodeTaskPayload
			if err := json.Unmarshal(d.Body, &task); err != nil {
				log.Printf("Error unmarshalling message: %s", err)
				d.Nack(false, false) // 消息格式错误，直接丢弃
				continue
			}

			// 调用真正的处理函数
			err := worker.HandleTranscode(task.VideoID)
			if err != nil {
				log.Printf("Failed to handle transcode for video %d: %v", task.VideoID, err)
				// 这里可以加入重试逻辑，但现在我们先简单地 Nack
				d.Nack(false, false) // 处理失败，也可以选择 requeue
			} else {
				log.Printf("Successfully transcoded video %d", task.VideoID)
				d.Ack(false) // !!! 非常重要：处理成功后，手动确认消息
			}
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever // 阻塞主线程，让 worker 一直运行
}