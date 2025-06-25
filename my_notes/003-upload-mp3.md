太好了！成功解决模块问题是 Go 开发的一大步。现在，我们正式进入项目最核心、最有趣的部分——**阶段 2：视频上传与转码**。

这个阶段的目标是打通从用户上传视频到后台自动处理的完整“流水线”。

**我们将要实现的流程：**

1.  **客户端（用 Postman/curl 模拟）** 向我们的 API 请求一个安全的“上传链接”。
2.  **API 服务器** 连接 MinIO，生成一个有时效性的**预签名 URL (Presigned URL)**，并把它返回给客户端。
3.  **客户端** 使用这个 URL，直接将视频文件上传到 MinIO。这样做的好处是**上传的巨大流量不经过我们的 API 服务器**，极大地减轻了服务器的负担。
4.  **客户端** 上传成功后，再调用 API 的一个接口，通知“上传已完成”。
5.  **API 服务器** 收到通知后，向 RabbitMQ 发送一条“转码”任务消息。
6.  **转码 Worker（一个独立的 Go 程序）** 监听到这条消息，从 MinIO 下载原视频，使用 `ffmpeg` 进行转码，然后将转码好的 HLS 文件（`.m3u8` 和 `.ts`）再上传回 MinIO。
7.  **转码 Worker** 最后更新数据库，将视频状态标记为 `online`。

听起来很复杂，但跟着步骤来，你会发现它非常清晰。

---

### **第 1 步：安装新依赖和 `ffmpeg`**

1.  **安装 Go 依赖包：**
    在你的项目根目录 `~/go/video-platform-go` 下，运行：
    ```bash
    # MinIO Go SDK
    go get -u github.com/minio/minio-go/v7
    # RabbitMQ Go 客户端
    go get -u github.com/rabbitmq/amqp091-go
    ```

2.  **安装 `ffmpeg`：**
    我们的 Go 程序会调用 `ffmpeg` 这个强大的命令行工具来处理视频。你需要在你的 Ubuntu 系统上安装它。
    ```bash
    sudo apt update
    sudo apt install ffmpeg
    ```
    安装完成后，可以运行 `ffmpeg -version` 来验证一下。

---

### **第 2 步：更新配置文件**

我们需要在配置中加入 MinIO 的连接信息。

**2.1. 编辑 `configs/config.yaml`**
在文件末尾添加 `minio` 部分：

```yaml
# ... (server, mysql, redis, jwt 配置保持不变) ...

minio:
  endpoint: "127.0.0.1:9000"
  access_key_id: "minioadmin" # 这是 docker-compose.yml 中定义的
  secret_access_key: "minioadmin" # 这也是 docker-compose.yml 中定义的
  use_ssl: false # 本地开发，不使用 HTTPS
  bucket_name: "videos" # 我们将要把视频存放到这个桶里
```

**2.2. 编辑 `internal/config/config.go`**
在 `Config` 结构体中添加 `MinIO` 部分来映射新的配置。

```go
// internal/config/config.go
// ...

type Config struct {
	// ... (Server, MySQL, Redis, JWT 结构体保持不变) ...

	MinIO struct {
		Endpoint        string `mapstructure:"endpoint"`
		AccessKeyID     string `mapstructure:"access_key_id"`
		SecretAccessKey string `mapstructure:"secret_access_key"`
		UseSSL          bool   `mapstructure:"use_ssl"`
		BucketName      string `mapstructure:"bucket_name"`
	} `mapstructure:"minio"`
}

// ... (Init 函数保持不变) ...
```

---

### **第 3 步：初始化 MinIO 客户端和创建存储桶**

我们需要一个地方来初始化 MinIO 客户端，并确保我们配置的存储桶（Bucket）存在。我们把它和数据库初始化放在一起。

**3.1. 编辑 `internal/dal/db.go`**
在这个文件中，我们不仅初始化 MySQL，也初始化 MinIO。

```go
// internal/dal/db.go
package dal

import (
	"context"
	"log"
	"github.com/cjh/video-platform-go/internal/config" // 确认是新的模块路径
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	DB          *gorm.DB
	MinioClient *minio.Client
)

// InitMySQL 初始化数据库连接
func InitMySQL(cfg *config.Config) {
    // ... (这部分代码不变) ...
}

// InitMinIO 初始化 MinIO 客户端
func InitMinIO(cfg *config.Config) {
	var err error
	MinioClient, err = minio.New(cfg.MinIO.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinIO.AccessKeyID, cfg.MinIO.SecretAccessKey, ""),
		Secure: cfg.MinIO.UseSSL,
	})
	if err != nil {
		log.Fatalf("Failed to initialize minio client: %v", err)
	}

	// 检查存储桶是否存在，如果不存在则创建
	bucketName := cfg.MinIO.BucketName
	ctx := context.Background()
	exists, err := MinioClient.BucketExists(ctx, bucketName)
	if err != nil {
		log.Fatalf("Failed to check if bucket exists: %v", err)
	}
	if !exists {
		err = MinioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			log.Fatalf("Failed to create bucket: %v", err)
		}
		log.Printf("Bucket '%s' created successfully.", bucketName)
	} else {
		log.Printf("Bucket '%s' already exists.", bucketName)
	}
}
```

**3.2. 编辑 `cmd/api/main.go`**
在 `main` 函数中调用新的 `InitMinIO` 函数。

```go
// cmd/api/main.go
// ...
func main() {
	// ... (初始化配置) ...

	// 2. 初始化数据库和 MinIO
	dal.InitMySQL(&config.AppConfig)
	dal.InitMinIO(&config.AppConfig) // <-- 新增这一行
	log.Println("Database and MinIO initialized")

	// ... (启动服务器) ...
}
```

---

### **第 4 步：创建视频上传的业务逻辑和服务**

我们需要一个新的 service 文件来处理视频相关的逻辑。

**4.1. 创建 `internal/service/video_service.go`**

```bash
touch internal/service/video_service.go
```

**4.2. 编写视频服务代码**
将以下代码粘贴到 `internal/service/video_service.go` 中。

```go
// internal/service/video_service.go
package service

import (
	"context"
	"path/filepath"
	"time"
	"github.com/cjh/video-platform-go/internal/config"
	"github.com/cjh/video-platform-go/internal/dal"
	"github.com/cjh/video-platform-go/internal/dal/model"
	"github.com/minio/minio-go/v7"
)

// InitiateUploadService 处理视频上传初始化的逻辑
func InitiateUploadService(userID uint64, fileName string) (string, *model.Video, error) {
	// 1. 创建视频记录
	video := model.Video{
		UserID: userID,
		Title:  fileName, // 暂时用文件名作为标题
		Status: "uploading",
	}
	if err := dal.DB.Create(&video).Error; err != nil {
		return "", nil, err
	}

	// 2. 生成预签名 URL
	// 对象在 MinIO 中的存储路径，例如: raw/123/video.mp4
	objectName := filepath.Join("raw", fmt.Sprintf("%d", video.ID), fileName)
	bucketName := config.AppConfig.MinIO.BucketName
	expiration := time.Hour * 24 // URL 有效期 24 小时

	presignedURL, err := dal.MinioClient.PresignedPutObject(context.Background(), bucketName, objectName, expiration)
	if err != nil {
		return "", nil, err
	}

	return presignedURL.String(), &video, nil
}

// ... 之后我们会在这里添加 CompleteUploadService ...
```
*注意：`fmt.Sprintf` 需要 `import "fmt"`，你的编辑器应该会自动帮你添加。*

---

### **第 5 步：创建视频上传的 API 接口**

**5.1. 创建 `internal/api/handler/video_handler.go`**

```bash
touch internal/api/handler/video_handler.go
```

**5.2. 编写视频处理器代码**
将以下代码粘贴到 `internal/api/handler/video_handler.go` 中。

```go
// internal/api/handler/video_handler.go
package handler

import (
	"net/http"
	"github.com/cjh/video-platform-go/internal/service"
	"github.com/gin-gonic/gin"
)

type InitiateUploadRequest struct {
	FileName string `json:"file_name" binding:"required"`
}

func InitiateUpload(c *gin.Context) {
	var req InitiateUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file name"})
		return
	}

	// 从 JWT 中间件获取用户ID
	userIDVal, _ := c.Get("user_id")
	userID, ok := userIDVal.(float64) // JWT 解析出的数字是 float64
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID in token"})
		return
	}

	url, video, err := service.InitiateUploadService(uint64(userID), req.FileName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"upload_url": url,
		"video_id":   video.ID,
	})
}
```

---

### **第 6 步：在 API 服务器中注册新路由**

**编辑 `cmd/api/main.go`**，在需要认证的路由组 `authed` 中添加我们的新接口。

```go
// cmd/api/main.go
// ...
func main() {
    // ...
	authed := apiV1.Group("/")
	authed.Use(middleware.JWTAuthMiddleware())
	{
		// ... (老的 /me 路由) ...

		// 视频路由
		videoRoutes := authed.Group("/videos")
		{
			// POST /api/v1/videos/upload/initiate
			videoRoutes.POST("/upload/initiate", handler.InitiateUpload)
		}
	}
    // ...
}
```

---

### **第 7 步：运行 API 并测试“获取上传链接”**

现在，我们先不启动 Worker，只测试获取上传链接的流程。

**7.1. 准备一个测试用的视频文件**
在你的电脑上找一个小的 `.mp4` 文件，或者下载一个。比如，你可以把它放在 `~/Downloads/test.mp4`。

**7.2. 重启 API 服务器**
在终端里，按 `Ctrl+C` 停掉之前运行的服务器，然后重新启动它：

```bash
# 确保在 video-platform-go 目录下
go mod tidy
go run ./cmd/api/main.go
```

**7.3. 使用 `curl` 测试**

1.  **登录获取 Token (和之前一样)**
    ```bash
    curl -s -X POST http://localhost:8000/api/v1/users/login \
    -H "Content-Type: application/json" \
    -d '{
        "email": "test@example.com",
        "password": "password123"
    }' | tee login_response.json # 将响应保存到文件，方便提取token
    ```

2.  **从 `login_response.json` 文件中复制 token。**

3.  **请求上传链接**
    把 `YOUR_JWT_TOKEN` 换成你上一步复制的 token。

    ```bash
    curl -X POST http://localhost:8000/api/v1/videos/upload/initiate \
    -H "Authorization: Bearer YOUR_JWT_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{
        "file_name": "my-cool-video.mp4"
    }'
    ```

**预期结果：**
你会收到一个 `200 OK` 响应，里面包含一个 `upload_url` 和一个 `video_id`。这个 URL 就是 MinIO 的预签名 URL。

```json
{
    "upload_url": "http://127.0.0.1:9000/videos/raw/1/my-cool-video.mp4?X-Amz-Algorithm=...",
    "video_id": 1
}
```

**7.4. (可选) 直接用 `curl` 上传文件**
**复制上一步返回的整个 `upload_url`**，然后在终端里执行：
（确保你的测试视频路径 `~/Downloads/test.mp4` 是正确的）

```bash
curl -T ~/Downloads/test.mp4 "PUT_YOUR_UPLOAD_URL_HERE"
```

如果命令执行成功没有任何输出，说明上传成功了！你可以打开浏览器访问 MinIO 控制台 `http://localhost:9001`，用 `minioadmin/minioadmin` 登录，你应该能看到 `videos` 桶里有了一个 `raw/1/my-cool-video.mp4` 文件。

至此，我们已经完成了上传流水线的前半部分！

下一步，我们将实现**通知**和**后台 Worker 转码**的部分。如果你已经完成了以上步骤并且测试成功，请告诉我，我们继续！