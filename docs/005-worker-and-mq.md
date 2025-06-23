好的，非常好！文件已经安稳地躺在 MinIO 的存储桶里了。现在，我们要启动整个后台自动化流水线的后半部分。

这后半部分的核心是：**“一个负责派发任务，一个负责埋头干活”**。

*   **派发任务**：我们的 API 服务器。当用户告诉它“我上传完了”，它就往 RabbitMQ 消息队列里扔一个“去转码”的指令。
*   **埋头干活**：我们的 Worker 程序。这是一个**独立运行的**程序，它的唯一工作就是守在 RabbitMQ 队列旁，拿到指令后，就去 MinIO 下载视频，用`ffmpeg`处理，再把结果存回去，最后更新数据库。

我们一步一步来，构建这个强大的后台。

---

### **阶段 2.5：完成上传与触发转码**

#### **第 1 步：配置 RabbitMQ**

首先，我们需要让 Go 程序知道如何连接 RabbitMQ。

**1.1. 更新 `configs/config.yaml`**
在文件末尾添加 `rabbitmq` 部分：

```yaml
# ... (minio 配置下方) ...

rabbitmq:
  url: "amqp://user:password@127.0.0.1:5672/" # 这是 docker-compose 中定义的用户和密码
  transcode_queue: "video_transcoding_queue" # 我们给转码任务队列起个名字
```

**1.2. 更新 `internal/config/config.go`**
在 `Config` 结构体中添加 `RabbitMQ` 的映射。

```go
// internal/config/config.go
type Config struct {
	// ... (MinIO 结构体下方) ...

	RabbitMQ struct {
		URL            string `mapstructure:"url"`
		TranscodeQueue string `mapstructure:"transcode_queue"`
	} `mapstructure:"rabbitmq"`
}
```

#### **第 2 步：创建消息队列(MQ)的连接和发布逻辑**

我们需要一个专门的地方来处理和 RabbitMQ 的交互。

**2.1. 创建 `internal/dal/mq.go` 文件**

```bash
touch internal/dal/mq.go
```

**2.2. 编写 `mq.go` 的代码**
将以下代码粘贴到 `internal/dal/mq.go` 中。

```go
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
```

**2.3. 在 API 服务器启动时初始化 MQ**
编辑 `cmd/api/main.go`，调用 `InitRabbitMQ`。

```go
// cmd/api/main.go
func main() {
	// ...
	// 2. 初始化数据库、MinIO 和 RabbitMQ
	dal.InitMySQL(&config.AppConfig)
	dal.InitMinIO(&config.AppConfig)
	dal.InitRabbitMQ(&config.AppConfig) // <-- 新增这一行
	log.Println("Database, MinIO and RabbitMQ initialized")
	// ...
}
```

#### **第 3 步：实现“完成上传”的接口和逻辑**

**3.1. 在 `internal/service/video_service.go` 中添加新函数**

```go
// internal/service/video_service.go
package service

import (
	"context"
	"encoding/json" // 确保导入
	"fmt"             // 确保导入
	"path/filepath"
	"time"
	// ... 其他 import ...
)

// TranscodeTaskPayload 是我们要发送到消息队列的任务内容
type TranscodeTaskPayload struct {
	VideoID uint64 `json:"video_id"`
}

// CompleteUploadService 处理“完成上传”的逻辑
func CompleteUploadService(videoID uint64) error {
	// 1. 验证视频是否存在且状态正确
	var video model.Video
	if err := dal.DB.First(&video, videoID).Error; err != nil {
		return fmt.Errorf("video with id %d not found", videoID)
	}
	if video.Status != "uploading" {
		return fmt.Errorf("video status is not 'uploading'")
	}

	// 2. 更新视频状态为 'transcoding'（准备中）
	if err := dal.DB.Model(&video).Update("status", "transcoding").Error; err != nil {
		return err
	}

	// 3. 创建任务并发送到 RabbitMQ
	task := TranscodeTaskPayload{VideoID: videoID}
	body, err := json.Marshal(task)
	if err != nil {
		// 如果序列化失败，最好把状态改回来，或者标记为失败
		dal.DB.Model(&video).Update("status", "failed")
		return fmt.Errorf("failed to create transcode task: %v", err)
	}

	return dal.PublishTranscodeTask(context.Background(), body)
}
// ... InitiateUploadService 保持不变 ...
```

**3.2. 在 `internal/api/handler/video_handler.go` 中添加新接口**

```go
// internal/api/handler/video_handler.go

type CompleteUploadRequest struct {
	VideoID uint64 `json:"video_id" binding:"required"`
}

func CompleteUpload(c *gin.Context) {
	var req CompleteUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid video ID"})
		return
	}

	// 可以在这里加一层验证，确保操作者是视频的上传者
	// userIDVal, _ := c.Get("user_id") ...

	err := service.CompleteUploadService(req.VideoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Transcoding task has been submitted"})
}
```

**3.3. 在 `cmd/api/main.go` 中注册新路由**

```go
// cmd/api/main.go
// ...
videoRoutes := authed.Group("/videos")
{
	// POST /api/v1/videos/upload/initiate
	videoRoutes.POST("/upload/initiate", handler.InitiateUpload)
	// POST /api/v1/videos/upload/complete
	videoRoutes.POST("/upload/complete", handler.CompleteUpload) // <-- 新增这一行
}
// ...
```

至此，我们的 API 服务器已经具备了“派发任务”的能力！但现在还没有人“接任务”。

---

### **阶段 2.6：创建并实现转码 Worker**

现在我们来创建那个“埋头干活”的工人程序。

#### **第 1 步：创建 Worker 的程序入口**

这是一个全新的、独立的 Go 程序。

```bash
# 在项目根目录执行
mkdir -p cmd/worker
touch cmd/worker/main.go
```

#### **第 2 步：编写 Worker 的主程序 `cmd/worker/main.go`**

这个程序很关键，它负责初始化所有需要的连接，然后进入一个无限循环来等待和处理任务。
**将以下代码完整地粘贴到 `cmd/worker/main.go` 中。**

```go
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
```
*你可能会发现 `internal/worker` 这个包不存在，我们马上创建它。*

#### **第 3 步：实现核心转码逻辑**

**3.1. 创建 `internal/worker/transcode.go` 文件**

```bash
mkdir -p internal/worker
touch internal/worker/transcode.go
```

**3.2. 编写转码逻辑 `transcode.go`**
这是整个项目技术含量最高的部分。我们将在这里调用 `ffmpeg`。
**将以下代码完整地粘贴到 `internal/worker/transcode.go` 中。**

```go
// internal/worker/transcode.go
package worker

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cjh/video-platform-go/internal/config"
	"github.com/cjh/video-platform-go/internal/dal"
	"github.com/cjh/video-platform-go/internal/dal/model"
)

// HandleTranscode 是处理转码任务的核心函数
func HandleTranscode(videoID uint64) error {
	// 0. 从数据库获取视频信息
	var video model.Video
	if err := dal.DB.First(&video, videoID).Error; err != nil {
		return fmt.Errorf("video %d not found: %w", videoID, err)
	}

	// 1. 创建临时工作目录
	tempDir, err := os.MkdirTemp("", fmt.Sprintf("video-%d-*", videoID))
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir) // 保证函数结束时删除临时目录

	// 2. 从 MinIO 下载原始视频
	bucketName := config.AppConfig.MinIO.BucketName
	// 原始文件路径我们当时是这样设计的：raw/{video_id}/{file_name}
	// 但 file_name 我们没有存，可以从 video.Title 读取（因为我们用它做了标题）
	// 更稳妥的做法是在 videos 表加一个 original_object_name 字段
	// 这里我们先用 Title 简化处理
	rawObjectName := filepath.Join("raw", fmt.Sprintf("%d", video.ID), video.Title)
	localRawPath := filepath.Join(tempDir, video.Title)

	err = dal.MinioClient.FGetObject(context.Background(), bucketName, rawObjectName, localRawPath, minio.GetObjectOptions{})
	if err != nil {
		dal.DB.Model(&video).Update("status", "failed")
		return fmt.Errorf("failed to download from minio: %w", err)
	}
	log.Printf("Downloaded %s to %s", rawObjectName, localRawPath)

	// 3. 执行 FFMPEG 转码 (以720p为例)
	outputDir := filepath.Join(tempDir, "hls_720p")
	os.Mkdir(outputDir, 0755)
	outputM3u8 := filepath.Join(outputDir, "720p.m3u8")

	// ffmpeg -i [输入文件] -c:v libx264 -c:a aac -vf "scale=-2:720" -hls_time 10 -hls_list_size 0 -f hls [输出.m3u8]
	cmd := exec.Command("ffmpeg",
		"-i", localRawPath,
		"-c:v", "libx264",
		"-c:a", "aac",
		"-vf", "scale=-2:720", // 保持宽高比，高度为720p
		"-hls_time", "10", // 每个 ts 切片 10 秒
		"-hls_list_size", "0", // 0表示保留所有切片
		"-f", "hls",
		outputM3u8,
	)

	log.Printf("Executing ffmpeg command: %s", cmd.String())
	output, err := cmd.CombinedOutput() // 获取标准输出和错误输出
	if err != nil {
		log.Printf("FFMPEG error output: %s", string(output))
		dal.DB.Model(&video).Update("status", "failed")
		return fmt.Errorf("ffmpeg command failed: %w", err)
	}
	log.Printf("FFMPEG success output: %s", string(output))

	// 4. 将转码后的 HLS 文件上传到 MinIO
	processedPathPrefix := filepath.Join("processed", fmt.Sprintf("%d", video.ID), "hls_720p")

	files, err := os.ReadDir(outputDir)
	if err != nil {
		return fmt.Errorf("failed to read HLS output dir: %w", err)
	}

	for _, file := range files {
		localFilePath := filepath.Join(outputDir, file.Name())
		remoteObjectName := filepath.Join(processedPathPrefix, file.Name())

		_, err := dal.MinioClient.FPutObject(context.Background(), bucketName, remoteObjectName, localFilePath, minio.PutObjectOptions{})
		if err != nil {
			dal.DB.Model(&video).Update("status", "failed")
			return fmt.Errorf("failed to upload HLS file %s: %w", file.Name(), err)
		}
	}
	log.Printf("Uploaded HLS files to %s", processedPathPrefix)

	// 5. 更新数据库
	// 开启一个事务来确保数据一致性
	tx := dal.DB.Begin()
	m3u8URL := filepath.ToSlash(filepath.Join(processedPathPrefix, "720p.m3u8")) // 确保是 / 分隔符
	// 创建视频源记录
	videoSource := model.VideoSource{
		VideoID: video.ID,
		Quality: "720p",
		Format:  "HLS",
		URL:     m3u8URL,
	}
	if err := tx.Create(&videoSource).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 更新主视频表状态
	if err := tx.Model(&video).Update("status", "online").Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}
```
*注：`minio.GetObjectOptions{}` 和 `minio.PutObjectOptions{}` 需要 `import "github.com/minio/minio-go/v7"`。你的编辑器会自动处理。*

---

### **阶段 2.7：同时运行并完整测试**

现在我们有了两个程序：`API Server` 和 `Worker`。我们需要**同时运行它们**。

**你需要打开两个终端。**

*   **在第一个终端**，启动 API 服务器：
    ```bash
    # 终端 1 (位于项目根目录)
    go run ./cmd/api/main.go
    ```

*   **在第二个终端**，启动 Worker：
    ```bash
    # 终端 2 (位于项目根目录)
    go run ./cmd/worker/main.go
    ```
    你应该能看到 Worker 打印出 "Waiting for messages"。

**现在，进行完整的端到端测试：**

1.  **(登录)** 获取你的 JWT Token。
2.  **(请求上传)** 调用 `/api/v1/videos/upload/initiate`，得到 `upload_url` 和 `video_id`。
3.  **(上传文件)** 使用 `curl -T` 将你的测试视频上传到返回的 `upload_url`。
4.  **(完成上传)** **这是新步骤！** 调用 `/api/v1/videos/upload/complete` 接口，并把上一步得到的 `video_id` 放到请求体里。
    ```bash
    # 把 YOUR_JWT_TOKEN 和 video_id 换成你自己的
    curl -X POST http://localhost:8000/api/v1/videos/upload/complete \
    -H "Authorization: Bearer YOUR_JWT_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{
        "video_id": 1
    }'
    ```

**观察你的两个终端！**

*   **API 服务器终端**：会显示处理 `/upload/complete` 请求的日志。
*   **Worker 终端**：在 API 调用后几乎立刻，它会打印 "Received a message"，然后开始打印下载、执行 FFMPEG、上传和成功的日志！

如果一切顺利，几秒或几十秒后（取决于视频大小），Worker 会打印出成功日志。这时，你可以去数据库里检查 `videos` 表的状态是否变成了 `online`，以及 `video_sources` 表里是否多了一条记录！

恭喜你！你已经构建了一个工业级的、异步的、可靠的视频处理流水线！