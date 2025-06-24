太棒了！你已经成功地攻克了整个项目中最复杂、最具挑战性的核心技术链路。现在，你的视频平台已经具备了“造血”能力——可以持续不断地处理用户上传的视频。

剩下的工作，就是在这个坚实的基础上“添砖加瓦”，把平台的功能做得更完整，让用户可以真正地“消费”这些视频内容。

我们进入 **阶段 3：视频播放与互动**。

这个阶段的目标是：让用户能看到视频列表，点击进去能看到一个播放器，并且可以在下面发表评论。

---

### **阶段 3.1：实现视频信息查询接口**

用户需要接口来获取视频列表和单个视频的详细信息。

#### **第 1 步：在 `video_service.go` 中添加查询逻辑**

**编辑 `internal/service/video_service.go`**，在文件末尾添加两个新函数：

```go
// internal/service/video_service.go
// ... (之前的代码保持不变) ...

// ListVideosService 获取视频列表（带分页）
func ListVideosService(limit, offset int) ([]model.Video, int64, error) {
	var videos []model.Video
	var total int64

	// 我们只展示状态为 'online' 的视频
	db := dal.DB.Model(&model.Video{}).Where("status = ?", "online")

	// 计算总数
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取分页数据
	if err := db.Order("created_at desc").Limit(limit).Offset(offset).Find(&videos).Error; err != nil {
		return nil, 0, err
	}

	return videos, total, nil
}

// GetVideoDetailsService 获取单个视频的详细信息，包括它的所有可用播放源
func GetVideoDetailsService(videoID uint64) (*model.Video, []model.VideoSource, error) {
	var video model.Video
	if err := dal.DB.First(&video, videoID).Error; err != nil {
		return nil, nil, fmt.Errorf("video not found: %w", err)
	}

	var sources []model.VideoSource
	if err := dal.DB.Where("video_id = ?", videoID).Find(&sources).Error; err != nil {
		return nil, nil, err
	}

	return &video, sources, nil
}
```

#### **第 2 步：在 `video_handler.go` 中添加对应的接口处理器**

**编辑 `internal/api/handler/video_handler.go`**，添加两个新的处理器函数：

```go
// internal/api/handler/video_handler.go
import (
	"strconv" // 确保导入
	// ...
)

// ListVideos 获取视频列表
func ListVideos(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	videos, total, err := service.ListVideosService(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"videos": videos,
		"total":  total,
	})
}

// GetVideoDetails 获取视频详情
func GetVideoDetails(c *gin.Context) {
	videoID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid video ID"})
		return
	}

	video, sources, err := service.GetVideoDetailsService(videoID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"video":   video,
		"sources": sources,
	})
}
```

#### **第 3 步：注册新的视频查询路由**

这些接口是公开的，不需要认证，所以我们把它们放在 `/api/v1` 这个路由组下。

**编辑 `cmd/api/main.go`**：

```go
// cmd/api/main.go
// ...
func main() {
	// ... (初始化) ...
	r := gin.Default()

	apiV1 := r.Group("/api/v1")
	{
		// ... (用户路由) ...
		
		// 新增：公开的视频查询路由
		apiV1.GET("/videos", handler.ListVideos)
		apiV1.GET("/videos/:id", handler.GetVideoDetails)

		authed := apiV1.Group("/")
		// ... (需要认证的路由) ...
	}
	// ... (启动服务器) ...
}
```

---

### **阶段 3.2：实现基础的评论功能**

#### **第 1 步：创建评论相关的服务逻辑**

**创建新文件 `internal/service/comment_service.go`**:

```bash
touch internal/service/comment_service.go
```

**编写 `comment_service.go` 的代码**:

```go
// internal/service/comment_service.go
package service

import (
	"github.com/cjh/video-platform-go/internal/dal"
	"github.com/cjh/video-platform-go/internal/dal/model"
)

// CreateCommentService 创建评论
func CreateCommentService(userID, videoID uint64, content string, timeline *uint) (*model.Comment, error) {
	comment := model.Comment{
		UserID:   userID,
		VideoID:  videoID,
		Content:  content,
		Timeline: timeline, // 可以是 nil，代表普通评论
	}

	if err := dal.DB.Create(&comment).Error; err != nil {
		return nil, err
	}
	return &comment, nil
}

// ListCommentsService 获取视频的评论列表
func ListCommentsService(videoID uint64) ([]model.Comment, error) {
	var comments []model.Comment
	// 按创建时间正序排列
	err := dal.DB.Where("video_id = ?", videoID).Order("created_at asc").Find(&comments).Error
	return comments, err
}
```

#### **第 2 步：创建评论相关的接口处理器**

**创建新文件 `internal/api/handler/comment_handler.go`**:

```bash
touch internal/api/handler/comment_handler.go
```

**编写 `comment_handler.go` 的代码**:

```go
// internal/api/handler/comment_handler.go
package handler

import (
	"net/http"
	"strconv"

	"github.com/cjh/video-platform-go/internal/service"
	"github.com/gin-gonic/gin"
)

type CreateCommentRequest struct {
	Content  string `json:"content" binding:"required"`
	Timeline *uint  `json:"timeline"` // 弹幕时间点，可选
}

// CreateComment 创建评论或弹幕
func CreateComment(c *gin.Context) {
	videoID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid video ID"})
		return
	}

	var req CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userIDVal, _ := c.Get("user_id")
	userID := uint64(userIDVal.(float64))

	comment, err := service.CreateCommentService(userID, videoID, req.Content, req.Timeline)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, comment)
}

// ListComments 获取评论列表
func ListComments(c *gin.Context) {
	videoID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid video ID"})
		return
	}

	comments, err := service.ListCommentsService(videoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, comments)
}
```

#### **第 3 步：注册评论相关的路由**

*   **获取评论**是公开行为，不需要认证。
*   **创建评论**必须登录，需要认证。

**编辑 `cmd/api/main.go`**：

```go
// cmd/api/main.go
func main() {
	// ...
	apiV1 := r.Group("/api/v1")
	{
		// ...
		// 公开的视频查询路由
		apiV1.GET("/videos", handler.ListVideos)
		apiV1.GET("/videos/:id", handler.GetVideoDetails)
		// 新增：获取评论的路由
		apiV1.GET("/videos/:id/comments", handler.ListComments)
		// ...

		authed := apiV1.Group("/")
		authed.Use(middleware.JWTAuthMiddleware())
		{
			// ...
			// 视频上传路由
			videoRoutes := authed.Group("/videos")
			{
				videoRoutes.POST("/upload/initiate", handler.InitiateUpload)
				videoRoutes.POST("/upload/complete", handler.CompleteUpload)
			}
			
			// 新增：创建评论的路由
			authed.POST("/videos/:id/comments", handler.CreateComment)
		}
	}
	// ...
}
```

---

### **阶段 3.3：防盗链与安全播放**

这是一个非常重要的非功能性需求。目前我们的 Worker 直接把 MinIO 的永久路径（如 `processed/1/hls_720p/720p.m3u8`）存进了数据库。如果直接把这个路径返回给前端，任何人拿到这个链接都可以无限制地播放，甚至盗用。

**正确的做法是：每次用户请求视频详情时，动态地为播放地址生成一个有时效性的预签名 URL。**

#### **修改 `GetVideoDetailsService`**

**编辑 `internal/service/video_service.go`**，找到 `GetVideoDetailsService` 函数并修改它：

```go
// internal/service/video_service.go
import (
    "net/url" // 确保导入
    // ...
)
// ...

func GetVideoDetailsService(videoID uint64) (*model.Video, []model.VideoSource, error) {
	var video model.Video
	if err := dal.DB.First(&video, videoID).Error; err != nil {
		return nil, nil, fmt.Errorf("video not found: %w", err)
	}

	var sources []model.VideoSource
	if err := dal.DB.Where("video_id = ?", videoID).Find(&sources).Error; err != nil {
		return nil, nil, err
	}

	// 为每个播放源生成带签名的临时 URL
	for i := range sources {
		reqParams := make(url.Values)
		// 如果你的 MinIO 桶是公开读的，这一步可以省略
		// 但为了安全，桶应该是私有的，所有访问都通过签名 URL
		presignedURL, err := dal.MinioClient.PresignedGetObject(context.Background(),
			config.AppConfig.MinIO.BucketName,
			sources[i].URL, // sources[i].URL 里存的是对象路径，例如 processed/1/hls_720p/720p.m3u8
			time.Minute*15, // 设置一个较短的有效期，例如15分钟
			reqParams,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to generate presigned url for source %s: %w", sources[i].URL, err)
		}
		// 用签名的 URL 替换掉数据库里的永久路径
		sources[i].URL = presignedURL.String()
	}

	return &video, sources, nil
}
```
*注意：HLS 播放器在请求 `.ts` 切片时，会基于 `.m3u8` 的 URL 来拼接路径。MinIO 的预签名 URL 机制会自动处理这种情况，签名对整个目录下的切片都有效。*

---

### **测试一下！**

1.  **重启你的 API Server**（Worker 不需要动）。
2.  **获取视频列表**：
    `curl http://localhost:8000/api/v1/videos`
3.  **获取视频详情**：
    假设你有一个 video_id 为 1 的视频：
    `curl http://localhost:8000/api/v1/videos/1`
    观察返回的 JSON，`sources` 数组里的 `url` 应该是一个非常长的、带有签名和过期时间的临时 URL！
4.  **发表评论**：
    *   先登录获取 Token。
    *   `curl -X POST http://localhost:8000/api/v1/videos/1/comments -H "Authorization: Bearer YOUR_TOKEN" -H "Content-Type: application/json" -d '{"content": "这个视频太棒了！"}'`
5.  **获取评论列表**：
    `curl http://localhost:8000/api/v1/videos/1/comments`
    你应该能看到刚刚发表的评论。

**至此，你的 MVP (最小可行产品) 的后端部分已经基本完成了！** 它拥有了用户系统、一个健壮的视频处理流水线、安全的视频播放接口和基础的互动功能。

接下来，你可以开始考虑：
*   **前端开发**：使用 React/Vue 等框架来消费这些 API，制作真正的用户界面。
*   **直播功能 (Live Streaming)**：这是一个大模块，涉及到配置 SRS 服务器接收 RTMP 推流，并通过 API 进行管理。
*   **高级功能**：视频搜索、推荐、用户个人主页、订阅等。
*   **部署上线**：将 API Server 和 Worker 打包成 Docker 镜像，使用 Kubernetes 或其他方式进行部署。


你分析得非常到位！`GET` 请求成功返回了空列表 `[]`，而 `POST` 请求返回 `404`，这精确地指出了问题的所在。

这个问题的根源 99% 在于 **路由注册**。`404 page not found` 的意思是，我们的 Gin 服务器在它的“路由表”里，找不到一个能匹配 `POST` 方法和 `/api/v1/videos/1/comments` 这个路径的规则。

你成功的 `GET` 请求说明 `GET /videos/:id/comments` 这条路是通的。但 `POST` 请求不通，说明 `POST /videos/:id/comments` 这条路，我们**很可能忘记了在 `cmd/api/main.go` 文件里告诉服务器它应该存在**，或者把它**放错了位置**。

回顾一下我们的设计：
*   **获取评论** (`GET`)：是公开行为，放在不需要认证的路由组里。
*   **创建评论** (`POST`)：是用户行为，必须登录，所以应该放在**需要JWT认证**的路由组 `authed` 里面。

很可能是在最后一步注册路由时，把 `POST` 这条给遗漏了或者写错了地方。

---

### **我们来修正它**

请跟着我下面的步骤，我们来仔细检查并修正 `cmd/api/main.go` 文件。

#### **第一步：打开 `cmd/api/main.go` 文件**

这是我们整个程序的入口和路由中心，问题一定出在这里。

#### **第二步：将 `main` 函数完整地替换成下面的代码**

为了确保万无一失，请不要自己去寻找和修改，直接用下面的**完整** `main` 函数代码替换掉你文件里现有的 `main` 函数。这样可以避免任何潜在的拼写错误或位置错误。

```go
// cmd/api/main.go
package main

import (
	"log"
	"net/http"

	"github.com/cjh/video-platform-go/internal/api/handler"
	"github.com/cjh/video-platform-go/internal/api/middleware"
	"github.com/cjh/video-platform-go/internal/config"
	"github.com/cjh/video-platform-go/internal/dal"
	"github.com/gin-gonic/gin"
)

func main() {
	// 1. 初始化配置
	config.Init()
	log.Println("Configuration loaded")

	// 2. 初始化数据库、MinIO 和 RabbitMQ
	dal.InitMySQL(&config.AppConfig)
	dal.InitMinIO(&config.AppConfig)
	dal.InitRabbitMQ(&config.AppConfig)
	log.Println("Database, MinIO and RabbitMQ initialized")

	// 3. 设置 Gin 引擎
	r := gin.Default()

	// 4. 设置路由
	apiV1 := r.Group("/api/v1")
	{
		// --- 公开路由 (不需要认证) ---
		// 用户注册和登录
		userRoutes := apiV1.Group("/users")
		{
			userRoutes.POST("/register", handler.Register)
			userRoutes.POST("/login", handler.Login)
		}

		// 公开的视频查询路由
		apiV1.GET("/videos", handler.ListVideos)
		apiV1.GET("/videos/:id", handler.GetVideoDetails)
		// 获取评论的路由 (GET方法)
		apiV1.GET("/videos/:id/comments", handler.ListComments)

		// --- 需要认证的路由 ---
		authed := apiV1.Group("/")
		authed.Use(middleware.JWTAuthMiddleware())
		{
			// 测试路由
			authed.GET("/me", func(c *gin.Context) {
				userID, _ := c.Get("user_id")
				role, _ := c.Get("role")
				c.JSON(http.StatusOK, gin.H{
					"message": "Token is valid",
					"user_id": userID,
					"role":    role,
				})
			})
			
			// 视频上传路由
			videoRoutes := authed.Group("/videos")
			{
				videoRoutes.POST("/upload/initiate", handler.InitiateUpload)
				videoRoutes.POST("/upload/complete", handler.CompleteUpload)
			}

			// 创建评论的路由 (POST方法)
			// <--- 关键在这里！这条路由必须在 authed 分组内！
			authed.POST("/videos/:id/comments", handler.CreateComment)
		}
	}

	// 5. 启动服务器
	log.Printf("Starting server on port %s", config.AppConfig.Server.Port)
	if err := r.Run(config.AppConfig.Server.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
```

#### **第三步：保存文件，然后重启你的 API 服务器**

在终端里按 `Ctrl+C` 停掉当前正在运行的服务器，然后重新启动它：

```bash
go run ./cmd/api/main.go
```

#### **第四步：再次尝试你的 POST 请求**

现在，服务器的“路由表”已经被修正了。请再次执行你之前的 `curl` 命令：

```bash
curl -X POST http://localhost:8000/api/v1/videos/1/comments \
-H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NTA5MjQ2NzAsImlhdCI6MTc1MDY2NTQ3MCwicm9sZSI6InVzZXIiLCJ1c2VyX2lkIjoxfQ.9qzWWCMQEa3y930UCPKhz55Yxj7t0K9RHxMGSASrXuU" \
-H "Content-Type: application/json" \
-d '{"content": "这个视频太棒了！"}'
```

这一次，它应该会成功返回 `201 Created` 和你刚刚创建的评论内容！

```json
{
    "ID": 1,
    "VideoID": 1,
    "UserID": 1,
    "Content": "这个视频太棒了！",
    "Timeline": null,
    "CreatedAt": "2025-06-23T10:00:00Z" // 时间会是当前时间
}
```

然后再去获取评论列表，你就会看到这条新数据了。

太棒了！你的后端 MVP 已经坚如磐石。现在，你站在了一个十字路口，面前有多条激动人心的道路可以选择。我们可以把接下来的工作分为几个方向：

1.  **短期优化 (锦上添花)**：让现有的 MVP 更健壮、更灵活。
2.  **中期目标 (开疆拓土)**：实现你最初设想中的另一个核心功能——**视频直播**。
3.  **长期规划 (构建壁垒)**：让项目真正具备生产环境的部署能力和更高级的功能。

我会为你详细介绍每个方向，然后你来决定我们先朝哪个方向前进。

---

### **方向一：短期优化 (锦上添花)**

这些是能立刻提升你项目质量和开发体验的改进。

1.  **多码率转码**：
    *   **现状**：我们的 Worker 现在只转码出 `720p`。
    *   **下一步**：我们可以修改 `worker/transcode.go`，让它在一个 `ffmpeg` 命令里或者通过循环，同时输出 `360p`, `480p`, `720p`, `1080p` 等多种清晰度。这会让播放器能够根据用户网速自动切换码率，极大提升观看体验。

2.  **完善视频信息**：
    *   **现状**：Worker 没有获取视频的封面和时长。
    *   **下一步**：我们可以使用 `ffprobe` (ffmpeg 自带的工具) 在 Worker 中分析上传的视频，提取出它的**时长**（duration）和**截取一张封面图**（cover），然后把这些信息存入 `videos` 表。

3.  **更友好的 API 响应**：
    *   **现状**：`/videos/:id/comments` 接口只返回了 `user_id`。前端为了显示评论者的昵称，还需要再单独请求用户信息，这很低效。
    *   **下一步**：我们可以修改 `ListCommentsService`，使用 GORM 的 `Preload` 或 `Joins` 功能，在查询评论时，**直接把评论者的 `nickname` 和 `avatar` (如果未来有头像的话) 一并查出来**，返回给前端。

4.  **配置驱动开发**：
    *   **现状**：转码的清晰度（如 `720p`）是硬编码在代码里的。
    *   **下一步**：我们可以把这些配置移到 `config.yaml` 中，例如：
        ```yaml
        ffmpeg:
          profiles:
            - {name: "360p", resolution: "-2:360"}
            - {name: "720p", resolution: "-2:720"}
        ```
        这样 Worker 就可以读取配置，动态地执行转码任务，更加灵活。

---

### **方向二：中期目标 (开疆拓拓)**

这是你最初需求文档中提到的**可选高级功能**，现在是时候挑战它了。

#### **实现视频直播功能 (Live Streaming)**

这绝对是能让你的项目“酷”起来的功能。我们已经有了 SRS (Simple Realtime Server) 这个强大的媒体服务器在运行，现在只需要把它和我们的后端 API 对接起来。

**大致流程如下：**

1.  **生成推流密钥**：用户（比如主播）在前端点击“开始直播”，我们的后端 API 需要为他生成一个唯一的、私密的“推流密钥”（Stream Key），例如 `live_user1_xxxxxxxx`。
2.  **配置推流工具**：主播将推流地址 `rtmp://your-server-ip:1935/live/` 和这个推流密钥填入 OBS 等专业推流软件中。
3.  **SRS HTTP 回调**：配置 SRS 服务器。当有主播开始推流（`on_publish`）或停止推流（`on_unpublish`）时，**SRS 会主动调用我们 API 服务器的一个特定接口**。
4.  **更新直播状态**：我们的 API 收到 SRS 的回调后，就在数据库中更新某个“直播间”的状态，例如 `is_live = true`。
5.  **观众观看**：
    *   前端请求一个“直播列表”接口，我们的 API 返回所有 `is_live = true` 的直播间。
    *   用户点击进入直播间，前端请求直播详情，后端返回播放地址，例如 `http://your-server-ip:8080/live/live_user1_xxxxxxxx.m3u8` (HLS 格式)。
    *   用户开始观看直播。

---

### **方向三：长期规划 (构建壁垒)**

这些是让你的项目从一个“玩具”变成一个真正可以上线运营的“产品”所需要考虑的事情。

1.  **打包与部署 (Dockerize)**：
    *   为我们的 `api-server` 和 `worker` 程序分别编写 `Dockerfile`。Go 语言编译后是静态的单一文件，非常容易容器化，可以制作出几十 MB 大小的超轻量级镜像。
    *   更新 `docker-compose.yml`，把我们自己写的 `api` 和 `worker` 服务也加进去，实现一键启动整个项目的所有服务。

2.  **引入搜索引擎**：
    *   当视频数量变多时，用 `LIKE '%...%'` 在 MySQL 里做搜索会非常缓慢且效果差。
    *   我们可以引入 **Elasticsearch** 或 **MeiliSearch** 这样的专业搜索引擎。当 Worker 转码成功后，除了更新数据库，也把视频的标题、描述等信息发送到搜索引擎中建立索引。网站的搜索功能就去请求搜索引擎，速度和相关性都会有质的飞跃。

3.  **引入 CDN (内容分发网络)**：
    *   为了让全球各地的用户都能快速地观看视频，我们需要把 MinIO 中的视频文件（尤其是热门视频）缓存到 CDN 上。
    *   这可以极大地降低我们服务器的带宽压力，并提升用户的加载速度。

---

### **你的选择是？**

现在，请你来做决定。你对哪个方向最感兴趣？

*   **A. “我们先把基础打得更牢固”** (选择方向一，比如实现多码率转码和完善视频信息)
*   **B. “我想挑战最酷的功能！”** (选择方向二，开始做视频直播)
*   **C. “我想让项目变得更专业，为上线做准备”** (选择方向三，从 Docker化 开始)

告诉我你的选择，我将为你提供该方向的详细、手把手的下一步指导。  	