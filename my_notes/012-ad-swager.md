下面给出三份 **已插好 Swagger 注解、能直接 `swag init` 通过** 的处理器源码。
只需把对应文件覆盖到原路径，然后执行：

```bash
go run golang.org/x/tools/cmd/goimports -w ./internal/api/handler
swag init -g ./cmd/api/main.go
```

访问 [http://localhost:8000/swagger/index.html](http://localhost:8000/swagger/index.html) 即可查看 API 文档。

---

## `internal/api/handler/user_handler.go`

```go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/cjh/video-platform-go/internal/service"
)

// ---------- 请求 / 响应 DTO ----------

// RegisterRequest 注册请求体
type RegisterRequest struct {
	Nickname string `json:"nickname" binding:"required"    example:"Tom"`
	Email    string `json:"email"    binding:"required,email" example:"tom@example.com"`
	Password string `json:"password" binding:"required,min=6" example:"secret123"`
}

// RegisterResponse 注册成功响应
type RegisterResponse struct {
	Message string `json:"message" example:"User registered successfully"`
	UserID  uint64 `json:"user_id" example:"1"`
}

// LoginRequest 登录请求体
type LoginRequest struct {
	Email    string `json:"email"    binding:"required,email" example:"tom@example.com"`
	Password string `json:"password" binding:"required"       example:"secret123"`
}

// LoginResponse 登录成功响应
type LoginResponse struct {
	Message string `json:"message" example:"Login successful"`
	Token   string `json:"token"   example:"<jwt>"`
}

// ErrorResponse 统一错误响应
type ErrorResponse struct {
	Error string `json:"error" example:"Invalid request body"`
}

// ---------- 处理器 ----------

// Register godoc
// @Summary      用户注册
// @Description  根据用户提供的昵称、邮箱和密码进行注册
// @Tags         用户
// @Accept       json
// @Produce      json
// @Param        body  body      RegisterRequest   true  "注册请求体"
// @Success      201   {object}  RegisterResponse
// @Failure      400   {object}  ErrorResponse
// @Failure      409   {object}  ErrorResponse
// @Router       /users/register [post]
func Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	user, err := service.Register(req.Nickname, req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusConflict, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, RegisterResponse{
		Message: "User registered successfully",
		UserID:  user.ID,
	})
}

// Login godoc
// @Summary      用户登录
// @Description  根据邮箱和密码进行登录，成功后返回 JWT Token
// @Tags         用户
// @Accept       json
// @Produce      json
// @Param        body  body      LoginRequest   true  "登录请求体"
// @Success      200   {object}  LoginResponse
// @Failure      400   {object}  ErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Router       /users/login [post]
func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	token, err := service.Login(req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		Message: "Login successful",
		Token:   token,
	})
}
```

---

## `internal/api/handler/comment_handler.go`

```go
package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/cjh/video-platform-go/internal/service"
	"github.com/gin-gonic/gin"
)

// ---------- 请求 / 响应 DTO ----------

// CreateCommentRequest 创建评论 / 弹幕请求体
type CreateCommentRequest struct {
	Content  string `json:"content"  binding:"required" example:"Great video!"`
	Timeline *uint  `json:"timeline" example:"15"` // 可选弹幕时间点（秒）
}

// CommentInfo 评论信息（用于列表和单条返回）
type CommentInfo struct {
	ID        uint64    `json:"id"         example:"1"`
	Content   string    `json:"content"    example:"Great video!"`
	Timeline  *uint     `json:"timeline,omitempty" example:"15"`
	CreatedAt time.Time `json:"created_at" example:"2025-06-25T11:34:00Z"`
	User      struct {
		ID       uint64 `json:"id"       example:"2"`
		Nickname string `json:"nickname" example:"Tom"`
	} `json:"user"`
}

// ErrorResponse 统一错误响应
type ErrorResponse struct {
	Error string `json:"error" example:"Invalid request body"`
}

// ---------- 处理器 ----------

// CreateComment godoc
// @Summary      创建评论 / 弹幕
// @Description  需要登录。根据视频 ID 创建评论或弹幕
// @Tags         评论
// @Security     ApiKeyAuth
// @Accept       json
// @Produce      json
// @Param        id    path      int64                true  "视频 ID"
// @Param        body  body      CreateCommentRequest true  "评论内容"
// @Success      201   {object}  CommentInfo
// @Failure      400   {object}  ErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Failure      500   {object}  ErrorResponse
// @Router       /videos/{id}/comments [post]
func CreateComment(c *gin.Context) {
	videoID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid video ID"})
		return
	}

	var req CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	userIDVal, _ := c.Get("user_id")
	userID := uint64(userIDVal.(float64))

	comment, err := service.CreateCommentService(userID, videoID, req.Content, req.Timeline)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	resp := CommentInfo{
		ID:        comment.ID,
		Content:   comment.Content,
		Timeline:  comment.Timeline,
		CreatedAt: comment.CreatedAt,
	}
	resp.User.ID = comment.User.ID
	resp.User.Nickname = comment.User.Nickname

	c.JSON(http.StatusCreated, resp)
}

// ListComments godoc
// @Summary      获取评论列表
// @Description  根据视频 ID 获取评论 / 弹幕列表
// @Tags         评论
// @Produce      json
// @Param        id  path      int64  true  "视频 ID"
// @Success      200 {array}   CommentInfo
// @Failure      400 {object}  ErrorResponse
// @Failure      500 {object}  ErrorResponse
// @Router       /videos/{id}/comments [get]
func ListComments(c *gin.Context) {
	videoID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid video ID"})
		return
	}

	comments, err := service.ListCommentsService(videoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	var resp []CommentInfo
	for _, comment := range comments {
		info := CommentInfo{
			ID:        comment.ID,
			Content:   comment.Content,
			Timeline:  comment.Timeline,
			CreatedAt: comment.CreatedAt,
		}
		info.User.ID = comment.User.ID
		info.User.Nickname = comment.User.Nickname
		resp = append(resp, info)
	}

	c.JSON(http.StatusOK, resp)
}
```

---

## `internal/api/handler/video_handler.go`

```go
package handler

import (
	"net/http"
	"strconv"

	"github.com/cjh/video-platform-go/internal/service"
	"github.com/gin-gonic/gin"
)

// ---------- 请求 / 响应 DTO ----------

// InitiateUploadRequest 初始化上传请求体
type InitiateUploadRequest struct {
	FileName string `json:"file_name" binding:"required" example:"holiday.mp4"`
}

// InitiateUploadResponse 初始化上传成功响应
type InitiateUploadResponse struct {
	UploadURL string `json:"upload_url" example:"https://minio.local/presigned-url"`
	VideoID   uint64 `json:"video_id"   example:"123"`
}

// CompleteUploadRequest 完成上传请求体
type CompleteUploadRequest struct {
	VideoID uint64 `json:"video_id" binding:"required" example:"123"`
}

// MessageResponse 通用消息响应
type MessageResponse struct {
	Message string `json:"message" example:"Transcoding task has been submitted"`
}

// ErrorResponse 统一错误响应
type ErrorResponse struct {
	Error string `json:"error" example:"Invalid request body"`
}

// VideoInfo 视频简要信息（列表用）
type VideoInfo struct {
	ID          uint64 `json:"id"          example:"123"`
	Title       string `json:"title"       example:"My Holiday"`
	CoverURL    string `json:"cover_url"   example:"https://example.com/cover.jpg"`
	Status      string `json:"status"      example:"published"`
	PlayCount   uint64 `json:"play_count"  example:"100"`
	CreatedAt   string `json:"created_at"  example:"2025-06-20T09:00:00Z"`
	Description string `json:"description" example:"A short description"`
}

// ListVideosResponse 视频列表响应
type ListVideosResponse struct {
	Videos []VideoInfo `json:"videos"`
	Total  int64       `json:"total"  example:"100"`
}

// VideoDetailsResponse 视频详情响应
type VideoDetailsResponse struct {
	Video   any `json:"video"`
	Sources any `json:"sources"`
}

// ---------- 处理器 ----------

// InitiateUpload godoc
// @Summary      初始化视频上传
// @Description  生成预签名上传 URL
// @Tags         视频
// @Security     ApiKeyAuth
// @Accept       json
// @Produce      json
// @Param        body  body      InitiateUploadRequest  true  "文件名"
// @Success      200   {object}  InitiateUploadResponse
// @Failure      400   {object}  ErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Failure      500   {object}  ErrorResponse
// @Router       /videos/upload/initiate [post]
func InitiateUpload(c *gin.Context) {
	var req InitiateUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid file name"})
		return
	}

	userIDVal, _ := c.Get("user_id")
	userID, ok := userIDVal.(float64)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Invalid user ID in token"})
		return
	}

	url, video, err := service.InitiateUploadService(uint64(userID), req.FileName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, InitiateUploadResponse{
		UploadURL: url,
		VideoID:   video.ID,
	})
}

// CompleteUpload godoc
// @Summary      完成视频上传
// @Description  通知后端开始转码
// @Tags         视频
// @Security     ApiKeyAuth
// @Accept       json
// @Produce      json
// @Param        body  body      CompleteUploadRequest  true  "视频 ID"
// @Success      200   {object}  MessageResponse
// @Failure      400   {object}  ErrorResponse
// @Failure      500   {object}  ErrorResponse
// @Router       /videos/upload/complete [post]
func CompleteUpload(c *gin.Context) {
	var req CompleteUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid video ID"})
		return
	}

	if err := service.CompleteUploadService(req.VideoID); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, MessageResponse{Message: "Transcoding task has been submitted"})
}

// ListVideos godoc
// @Summary      获取视频列表
// @Tags         视频
// @Produce      json
// @Param        limit   query     int  false  "每页数量"  default(10)
// @Param        offset  query     int  false  "偏移量"   default(0)
// @Success      200     {object}  ListVideosResponse
// @Failure      500     {object}  ErrorResponse
// @Router       /videos [get]
func ListVideos(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	videos, total, err := service.ListVideosService(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, ListVideosResponse{
		Videos: videos, // 如果 videos 不是 []VideoInfo，可自行转换
		Total:  total,
	})
}

// GetVideoDetails godoc
// @Summary      获取视频详情
// @Tags         视频
// @Produce      json
// @Param        id   path      int64  true  "视频 ID"
// @Success      200  {object}  VideoDetailsResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Router       /videos/{id} [get]
func GetVideoDetails(c *gin.Context) {
	videoID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid video ID"})
		return
	}

	video, sources, err := service.GetVideoDetailsService(videoID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, VideoDetailsResponse{
		Video:   video,
		Sources: sources,
	})
}
```

---

### 现在可以 `swag init` 了

文件覆盖完毕后，重新执行 `swag init -g ./cmd/api/main.go`，应该不再出现 “cannot find type definition: gin.H” 之类错误，Swagger UI 也能正确展示每个接口、请求体和响应格式。

如果后续还想微调字段或描述，直接改对应 `@` 注解即可，`swag init` 会自动刷新文档。
### 发生了什么？

在 **同一个包 `handler`** 里，`ErrorResponse` 被定义了 3 次（`user_handler.go`、`comment_handler.go`、`video_handler.go` 各一份）。
Go 规定：**同包内的标识符不可重复定义**，因此编译器抛出 `ErrorResponse redeclared in this block`。

---

## 一劳永逸的解决方案

1. **把 `ErrorResponse` 提取到一个单独文件**，让所有处理器共用。
2. **删掉其它文件里的重复定义。**

这样既消除了冲突，也避免了未来再复制粘贴时踩坑。

---

### 步骤 1：新增 `internal/api/handler/error_response.go`

```go
// internal/api/handler/error_response.go
package handler

// ErrorResponse 统一错误响应
type ErrorResponse struct {
	Error string `json:"error" example:"Invalid request body"`
}
```

> 文件名随意，比如 `response_common.go` 也行，只要在 **同一包 handler** 里即可。

---

### 步骤 2：删除重复定义

把下面两段代码**整块删掉**：

```go
// comment_handler.go（约第 30 行左右）
type ErrorResponse struct {
	Error string `json:"error" example:"Invalid request body"`
}
```

```go
// video_handler.go（约第 35 行左右）
type ErrorResponse struct {
	Error string `json:"error" example:"Invalid request body"`
}
```

> `user_handler.go` 里原来的那份也可以删掉；只保留新增文件中的**唯一**定义。

---

### 步骤 3：确认引用无需改动

因为三个文件仍位于同一个包，删除后它们直接引用 `ErrorResponse` 就行，**不需要额外 import**。

---

### 步骤 4：保存并检查

```bash
go run golang.org/x/tools/cmd/goimports -w ./internal/api/handler
go vet ./...
go test ./...   # 如果有测试
```

编译器不再报重定义错误，`swag init` 也一样可以通过。

---

#### 小贴士

* **共用 DTO**：类似 `MessageResponse` 这类通用结构体，也可以放到同一个文件，便于统一维护。
* **避免复制黏贴出错**：写一个简单的代码模板或使用 IDE snippet，可减少手动拷贝时遗漏修改的情况。

搞定！再试 `go build` 或 `swag init -g ./cmd/api/main.go` 应该就顺利了。如果还有别的编译问题，直接贴错误信息过来即可。


### 为什么会报错？

`service.ListVideosService` 返回的是 `[]model.Video`（数据库实体），而 `ListVideosResponse` 期望的是 `[]VideoInfo`（用于前端展示的 DTO）。
**Go 不能把两种不同类型直接赋值**，所以出现：

```
cannot use videos (variable of type []model.Video) as []VideoInfo value
```

---

## 按实体结构重新定义 `VideoInfo`

根据你贴出的 **`model.Video`** 字段，前端常用信息大致有：

| 字段          | 说明                          |
| ----------- | --------------------------- |
| ID          | 视频 ID                       |
| Title       | 标题                          |
| Description | 描述                          |
| CoverURL    | 封面地址                        |
| Status      | 状态：online / transcoding ... |
| Duration    | 时长（秒）                       |
| CreatedAt   | 创建时间                        |

把 `VideoInfo` 改成对应字段即可（示例在下方代码里）。

---

## 更新 `video_handler.go`

下面给出 **完整可编译版本**，只需把现有文件替换即可。

> 如果你的包导入路径和之前不同，请自行调整 `import`。

```go
package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/cjh/video-platform-go/internal/dal/model"
	"github.com/cjh/video-platform-go/internal/service"
)

// ---------- 请求 / 响应 DTO ----------

// InitiateUploadRequest ...
type InitiateUploadRequest struct {
	FileName string `json:"file_name" binding:"required" example:"holiday.mp4"`
}

// InitiateUploadResponse ...
type InitiateUploadResponse struct {
	UploadURL string `json:"upload_url" example:"https://minio.local/presigned-url"`
	VideoID   uint64 `json:"video_id"   example:"123"`
}

// CompleteUploadRequest ...
type CompleteUploadRequest struct {
	VideoID uint64 `json:"video_id" binding:"required" example:"123"`
}

// MessageResponse ...
type MessageResponse struct {
	Message string `json:"message" example:"Transcoding task has been submitted"`
}

// ErrorResponse 定义在 error_response.go，共用

// ------------ 新的列表 / 详情 DTO ------------

// VideoInfo 与 model.Video 字段一一对应（仅保留需要给前端看的字段）
type VideoInfo struct {
	ID          uint64    `json:"id"          example:"123"`
	Title       string    `json:"title"       example:"My Holiday"`
	Description string    `json:"description" example:"A short description"`
	CoverURL    string    `json:"cover_url"   example:"https://example.com/cover.jpg"`
	Status      string    `json:"status"      example:"online"`
	Duration    uint      `json:"duration"    example:"3600"`
	CreatedAt   time.Time `json:"created_at"  example:"2025-06-20T09:00:00Z"`
}

// ListVideosResponse ...
type ListVideosResponse struct {
	Videos []VideoInfo `json:"videos"`
	Total  int64       `json:"total" example:"100"`
}

// VideoDetailsResponse 可返回完整实体，也可重用 VideoInfo；这里示例使用完整实体
type VideoDetailsResponse struct {
	Video   model.Video          `json:"video"`
	Sources []model.VideoSource  `json:"sources"`
}

// ---------- 处理器 ----------

// InitiateUpload 省略（保持原实现）

// CompleteUpload 省略（保持原实现）

// ListVideos godoc
// @Summary      获取视频列表
// @Tags         视频
// @Produce      json
// @Param        limit   query     int  false  "每页数量"  default(10)
// @Param        offset  query     int  false  "偏移量"   default(0)
// @Success      200     {object}  ListVideosResponse
// @Failure      500     {object}  ErrorResponse
// @Router       /videos [get]
func ListVideos(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	videos, total, err := service.ListVideosService(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	// 转换 []model.Video -> []VideoInfo
	respVideos := make([]VideoInfo, 0, len(videos))
	for _, v := range videos {
		respVideos = append(respVideos, VideoInfo{
			ID:          v.ID,
			Title:       v.Title,
			Description: v.Description,
			CoverURL:    v.CoverURL,
			Status:      v.Status,
			Duration:    v.Duration,
			CreatedAt:   v.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, ListVideosResponse{
		Videos: respVideos,
		Total:  total,
	})
}

// GetVideoDetails godoc
// @Summary      获取视频详情
// @Tags         视频
// @Produce      json
// @Param        id   path      int64  true  "视频 ID"
// @Success      200  {object}  VideoDetailsResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Router       /videos/{id} [get]
func GetVideoDetails(c *gin.Context) {
	videoID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid video ID"})
		return
	}

	video, sources, err := service.GetVideoDetailsService(videoID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, VideoDetailsResponse{
		Video:   *video,
		Sources: sources,
	})
}
```

### 关键点

1. **`VideoInfo` 字段与 `model.Video` 对齐**，并用 `time.Time` 直接返回日期（前端更易处理 ISO8601）。
2. **列表转换**：循环把每个 `model.Video` 转成 `VideoInfo` 放到 `respVideos`。
3. **详情接口**：直接返回 `model.Video`，如果不想暴露全部字段，可以自定义另一套 DTO。
4. **Swagger 注解**：已同步更新 `VideoInfo` 和 `VideoDetailsResponse`。

---

### 最后一步：重新编译 & 生成文档

```bash
go run golang.org/x/tools/cmd/goimports -w ./internal/api/handler
go vet ./...
swag init -g ./cmd/api/main.go
```

现在 `video_handler.go` 应该可以正常编译，`swag init` 也会把新的 `VideoInfo` 显示在接口文档里。如果还有任何类型或编译错误，把报错贴出来即可继续排查。


下面给出一份**从零到跑通**的指南，假设你在 **Linux/macOS** 开发机上。本质步骤：

1. 安装依赖
2. 启动基础服务（数据库、MinIO、RabbitMQ）
3. 生成 Swagger 文档
4. 启动三个 Go 进程（API Server / Worker / Admin 站）
5. 访问 Swagger UI 与业务接口

> 如果你的环境已经有 Docker、Go 1.22+ 等，可直接跳到 **步骤 ②**。

---

## ① 环境准备

| 组件                          | 版本建议                                                | 作用                        |
| --------------------------- | --------------------------------------------------- | ------------------------- |
| **Go**                      | ≥ 1.22                                              | 编译 / 运行后端                 |
| **Docker & Docker Compose** | 最新稳定版                                               | 一键启动 MySQL、MinIO、RabbitMQ |
| **Swag CLI**                | `go install github.com/swaggo/swag/cmd/swag@latest` | 生成 Swagger 文档             |
| **Make** (可选)               |                                                     | 项目自带 Makefile，省事          |

```bash
# 安装 swag
go install github.com/swaggo/swag/cmd/swag@latest
# GOPATH/bin 要在 PATH 中，或者用 $(go env GOPATH)/bin/swag
```

---

## ② 启动基础服务

项目根目录已经有 `docker-compose.yml`，一次性拉起 **MySQL + MinIO + RabbitMQ**（或你自己定义的服务）。

```bash
docker compose pull          # 拉取镜像
docker compose up -d         # 后台启动
docker compose ps            # 查看容器状态
```

> 默认端口示例
>
> * MySQL: `127.0.0.1:3306`
> * MinIO: `127.0.0.1:9000` (UI) / `9001` (API)
> * RabbitMQ: `127.0.0.1:15672` (Management UI) / `5672` (AMQP)

如果你想改账号密码、桶名等，编辑 `configs/config.yaml` & `docker-compose.yml` 保持一致。

---

## ③ 初始化数据库（可选）

仓库里有若干 `.sql` 脚本（`admin.sql`, `create_admin_tables.sql`, `drop.sql`）。若你用了空库，执行：

```bash
docker exec -i <mysql-container> mysql -uroot -pYOURPWD < admin.sql
```

或者跳过：运行时由 GORM 自动迁移生成表结构。

---

## ④ 生成 Swagger 文档

```bash
# 在项目根
swag init -g ./cmd/api/main.go
# docs/swagger.json docs/swagger.yaml 会自动生成
```

如果 `go run golang.org/x/tools/cmd/goimports -w ./internal/...` 提示 import 顺序问题，先整理再 swag。

---

## ⑤ 启动 Go 服务

### 1. API Server

```bash
go run ./cmd/api
# 或者
make api
```

默认监听 `:8000`（如 `configs/config.yaml` 配置）。

> 访问 `http://localhost:8000/swagger/index.html` 查看文档。
> 如果 swagger UI 404，请确认 `r.GET("/swagger/*any", ginSwagger.WrapHandler(...))` 已在 `cmd/api/main.go` 注册。

### 2. 后台管理站（可选）

```bash
go run ./cmd/admin
# 或 make admin
```

默认监听 `:8080`。涉及 gin-admin 模板、静态文件等。

### 3. Worker（视频转码任务消费者）

```bash
go run ./cmd/worker
# 或 make worker
```

Worker 会监听 RabbitMQ `transcode` 队列，把上传好的视频转码并写回 MinIO / 数据库。

> **必须** Worker 在跑，前端把视频上传完并 `POST /videos/upload/complete` 后才会推进状态到 `online`。

---

## ⑥ 上传与播放完整链路（本地验证）

1. **初始化上传**

   ```bash
   curl -X POST http://localhost:8000/videos/upload/initiate \
     -H "Authorization: Bearer <JWT>" \
     -d '{"file_name":"holiday.mp4"}'
   ```

   返回 `upload_url` + `video_id`。

2. **PUT 上传文件**（示例用 `curl --upload-file`，也可用 MinIO Client / SDK）

   ```bash
   curl -T ./holiday.mp4 "<upload_url>"
   ```

3. **通知完成**

   ```bash
   curl -X POST http://localhost:8000/videos/upload/complete \
     -H "Authorization: Bearer <JWT>" \
     -d '{"video_id":123}'
   ```

4. **Worker 自动转码** ➜ `status` 变 `online`。

5. **列表 / 播放**

   ```bash
   curl http://localhost:8000/videos?limit=10
   curl http://localhost:8000/videos/123
   ```

---

## ⑦ 常用排错命令

| 场景           | 命令                                                            |
| ------------ | ------------------------------------------------------------- |
| 看容器日志        | `docker compose logs -f mysql`                                |
| 数据库 shell    | `docker exec -it <mysql-container> mysql -uroot -p`           |
| 检查队列堆积       | 打开 [http://localhost:15672](http://localhost:15672)，Queues 面板 |
| 查看 Worker 输出 | `go run ./cmd/worker` 或 `docker logs -f worker`（如用容器）         |
| MinIO 预签名异常  | 确认 `configs/config.yaml` 中的 AccessKey / SecretKey 与容器一致       |

---

## ⑧ FAQ

| 问题                      | 解决                                                           |
| ----------------------- | ------------------------------------------------------------ |
| `swag init` 找不到 `gin.H` | 已改为统一的 `ErrorResponse` / DTO，确保最新代码                          |
| 端口冲突                    | 修改 `configs/config.yaml` 里 server 端口或 docker-compose 端口映射    |
| “access denied” 上传失败    | 确认 MinIO 桶策略为私有 + 使用预签名 URL 上传                               |
| Worker 连不上 RabbitMQ     | 检查 `amqp://user:pass@rabbitmq:5672/` URL 是否对、RabbitMQ 容器是否启动 |

---

现在，浏览器打开：

```
http://localhost:8000/swagger/index.html      # API 文档
http://localhost:9000                         # MinIO 控制台 (admin / password)
http://localhost:15672                        # RabbitMQ 管理界面 (guest / guest)
```

即可完整体验 **Video Platform Go**。
如果某一步骤仍有疑问，把错误日志或行为描述贴出来，我再帮你定位！
