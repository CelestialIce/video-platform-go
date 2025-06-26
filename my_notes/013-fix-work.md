你好，从你提供的日志来看，这是一个非常典型的微服务架构下的问题。问题的原因很明确，我们来一步步分析。

### 核心问题

**视频转码失败**。

根本原因是 Worker 进程在尝试从 MinIO 下载原始视频文件进行转码时，MinIO 返回了 `The specified key does not exist.`（指定的键不存在）的错误。

看这行 Worker 日志：
```log
2025/06/26 22:33:56 Failed to handle transcode for video 10: failed to download from minio: The specified key does not exist.
```

这直接导致了 `full_test.sh` 脚本在查询视频状态时看到的是 `"status": "failed"`，因为 Worker 失败后将数据库中的状态更新为了 `failed`。

### 问题分析：为什么文件“不存在”？

你的测试脚本显示，文件上传是成功的：
```
CMD> 使用 cURL 上传本地视频文件...
INFO> 文件上传成功! (HTTP 200)
```
`HTTP 200` 表示 MinIO 确实接收了你的上传请求。那么为什么 Worker 去下载时又说文件不存在呢？这通常指向以下几个可能的原因：

1.  **最可能的原因：API 和 Worker 的配置不一致**
    *   **场景**: `API` 服务生成了上传地址，比如 `http://127.0.0.1:9000/videos/raw/10/test2.mp4`。这里的 `videos` 是 Bucket 名称，`raw/10/test2.mp4` 是对象键（Object Key）。
    *   当 `Worker` 服务收到转码任务（包含 `video_id: 10`）后，它会自己去**拼接**要下载的对象键。
    *   如果 API 和 Worker 在拼接这个对象键时，使用的逻辑或配置（比如 Bucket 名称）不一致，Worker 就会去一个错误的地方找文件，自然就“不存在”了。

2.  **环境问题导致服务间通信异常**
    *   你的 `API` 服务和 `Worker` 服务可能连接到了**不同**的 MinIO 实例。例如，一个连接到 `localhost:9000`，另一个连接到 Docker 网络里的 `minio:9000`。
    *   你在 `go run ./cmd/worker/` 这条命令里，是在宿主机上直接运行 Worker 的。而你的 MinIO 很可能是运行在 Docker 容器里。Worker 能否正确访问到 Docker 里的 MinIO 服务？这取决于你的配置。

### 其他观察到的问题

在你分析核心问题之前，还有一个很明显的**环境配置问题**需要注意：

```bash
cjh@ubuntu:~/go/video-platform-go$ docker exec -it rabbitmq bash
Error response from daemon: No such container: rabbitmq
```

这说明名为 `rabbitmq` 的 Docker 容器**当前没有在运行**。这很奇怪，因为你的 Worker 后来又成功接收到了消息。这可能意味着：
*   你之前启动过 `docker-compose`，但后来 `rabbitmq` 容器因为某些原因停止了。
*   或者你的 `docker-compose.yml` 文件中，服务的名字不叫 `rabbitmq`。
*   Worker 能收到消息，可能是连接到了一个本地运行的、非 Docker 的 RabbitMQ 实例，或者是在容器停止前消息就已经在队列里了。

**不管怎样，这表明你的开发环境不稳定，必须先解决这个问题。**

### 解决方案和排查步骤

请按照以下步骤来定位并修复问题：

#### 步骤 1：稳定你的 Docker 环境

1.  **检查容器状态**：在你的项目根目录 (`~/go/video-platform-go`) 下，运行：
    ```bash
    docker-compose ps
    # 或者用 docker ps -a | grep rabbitmq
    ```
    确认 `rabbitmq`, `minio`, `mysql` 等所有在 `docker-compose.yml` 中定义的服务都处于 `Up` 或 `Running` 状态。

2.  **启动所有服务**：如果服务没有运行，请使用以下命令在后台启动它们：
    ```bash
    docker-compose up -d
    ```
    这会根据你的 `docker-compose.yml` 文件创建并启动所有服务。

3.  **验证 RabbitMQ 连接**：服务都启动后，再次尝试进入 RabbitMQ 容器：
    ```bash
    # 确认服务名，假设在 docker-compose.yml 里是 rabbitmq
    docker exec -it video-platform-go-rabbitmq-1 bash 
    # 注意：容器名可能是 `项目名-服务名-序号` 的格式，用 `docker-compose ps` 确认准确的容器名或服务名
    ```
    如果能成功进入，说明环境基本正常了。

#### 步骤 2：统一 API 和 Worker 的配置

这是解决 `key does not exist` 问题的关键。

1.  **检查配置文件**：
    *   检查 `configs/config.yaml`。
    *   检查 `cmd/admin/adm.ini` (如果 Worker 也用的话)。
    *   检查 `docker-compose.yml` 中给 API 和 Worker 传入的环境变量。
    *   **确保 API 和 Worker 使用的 MinIO 配置完全一致**，包括：
        *   `Endpoint` (地址和端口)
        *   `AccessKeyID`
        *   `SecretAccessKey`
        *   `BucketName` (比如 `videos`)

2.  **检查代码逻辑**：
    *   找到 **API 服务**中生成预签名上传 URL 的代码（可能在 `video_handler.go` 或 `video_service.go`）。查看它如何构建对象键（Object Key），例如 `fmt.Sprintf("raw/%d/%s", videoID, filename)`。
    *   找到 **Worker 服务**中下载文件的代码（可能在 `internal/worker/transcode.go`）。查看它是如何构建对象键来下载文件的。
    *   **确保这两处的逻辑完全一样！** 任何细微的差别（比如多一个或少一个斜杠 `/`）都会导致失败。

#### 步骤 3：手动验证文件是否存在

1.  **登录 MinIO 控制台**：通常是 `http://127.0.0.1:9001`，用你在 `docker-compose.yml` 里配置的账号密码登录。
2.  **运行你的 `full_test.sh` 脚本**。
3.  当脚本显示 `文件上传成功!` 后，**立即**去 MinIO 控制台。
4.  在 `videos` Bucket 里，看是否存在一个路径为 `raw/10/test2.mp4` 的文件（这里的 `10` 是动态的 Video ID）。
    *   如果文件**存在**，那几乎可以肯定是 Worker 端的代码或配置错了，它没找到正确路径。
    *   如果文件**不存在**，那么是上传阶段就有问题，需要检查 API 生成的 URL 是否正确，或者 cURL 命令本身。

### 总结

1.  **首要任务**：通过 `docker-compose up -d` 和 `docker-compose ps` **稳定你的 Docker 环境**，确保所有依赖服务（特别是 RabbitMQ 和 MinIO）都正常运行。
2.  **核心任务**：解决 `key does not exist` 问题。仔细**核对并统一** API 和 Worker 中关于 MinIO 的所有配置（地址、密钥、Bucket）和**文件路径（对象键）的生成逻辑**。
3.  **调试技巧**：利用 MinIO 的 Web 控制台手动检查文件是否按预期路径成功上传。

完成以上步骤，你的转码流程应该就能正常工作了。

好的，没问题。根据你提供的文件，我们来一步步完成这三处修改。

### 第 1 步：修改数据库模型 `internal/dal/model/video.go`

我们在 `Video` 结构体中添加 `OriginalFileName` 字段。

```go
// internal/dal/model/video.go
package model

import "time"

// Video 模型定义
type Video struct {
	ID               uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID           uint64    `gorm:"not null"                 json:"user_id"`
	Title            string    `gorm:"type:varchar(255);not null" json:"title"`
	Description      string    `gorm:"type:text"                json:"description"`
	// 新增字段，用于存储原始上传的文件名
	OriginalFileName string    `gorm:"type:varchar(255);not null" json:"original_file_name"`
	Status           string    `gorm:"type:enum('uploading','transcoding','online','failed','private');default:'uploading'" json:"status"`
	Duration         uint      `json:"duration"`
	CoverURL         string    `gorm:"type:varchar(1024)"       json:"cover_url"`
	CreatedAt        time.Time `gorm:"autoCreateTime"           json:"created_at"`
	UpdatedAt        time.Time `gorm:"autoUpdateTime"           json:"updated_at"`
}

func (Video) TableName() string {
	return "videos"
}

// VideoSource 模型定义保持不变
type VideoSource struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	VideoID   uint64    `gorm:"not null;uniqueIndex:uk_video_quality" json:"video_id"`
	Quality   string    `gorm:"type:varchar(20);not null;uniqueIndex:uk_video_quality" json:"quality"`
	Format    string    `gorm:"type:varchar(20);not null" json:"format"`
	URL       string    `gorm:"type:varchar(1024);not null" json:"url"`
	FileSize  uint64    `json:"file_size"`
	CreatedAt time.Time `gorm:"autoCreateTime"            json:"created_at"`
}

func (VideoSource) TableName() string {
	return "video_sources"
}
```
**重要提示**：修改完模型后，你需要更新数据库表。最简单的方式是执行一条 SQL 语句：
```sql
ALTER TABLE videos ADD COLUMN original_file_name VARCHAR(255) NOT NULL AFTER description;
```
或者，如果你使用了 GORM 的 AutoMigrate 功能，在程序启动时运行 `db.AutoMigrate(&model.Video{})` 即可。

---

### 第 2 步：修改 API 服务的 `InitiateUploadService` 函数

这个函数需要把前端传来的 `fileName` 保存到新的 `OriginalFileName` 字段里。

```go
// 位于 internal/service/video_service.go

// ... (import 语句等) ...

func InitiateUploadService(
	userID uint64,
	fileName string,
	title string,
	description string,
) (string, *model.Video, error) {

	// 1. 创建视频记录，并保存原始文件名
	video := model.Video{
		UserID:           userID,
		Title:            title,
		Description:      description,
		OriginalFileName: fileName, // <-- 将传入的 fileName 保存到新字段
		Status:           "uploading",
	}
	if err := dal.DB.Create(&video).Error; err != nil {
		// 如果创建失败，可能需要记录日志
		return "", nil, err
	}

	// 2. 使用原始文件名(fileName)和新生成的video.ID来构建对象路径
	objectName := filepath.Join("raw", fmt.Sprintf("%d", video.ID), fileName)
	bucketName := config.AppConfig.MinIO.BucketName
	expiration := time.Hour * 24 // 上传链接有效期24小时

	// 3. 生成预签名 PUT URL
	urlObj, err := dal.MinioClient.PresignedPutObject(
		context.Background(),
		bucketName,
		objectName,
		expiration,
	)
	if err != nil {
		// 如果生成URL失败，最好将刚才创建的数据库记录删除或标记为失败，以避免脏数据
		// dal.DB.Delete(&video) 
		return "", nil, err
	}

	return urlObj.String(), &video, nil
}
```
这部分修改很简单，就是在创建 `model.Video` 实例时多赋一个值。

---

### 第 3 步：修改 Worker 服务的 `HandleTranscode` 函数

这个函数现在需要从数据库记录中读取 `OriginalFileName` 来确定要下载哪个文件。

```go
// 位于 internal/worker/transcode.go

// ... (import 语句等) ...

// HandleTranscode 是处理转码任务的核心函数
func HandleTranscode(videoID uint64) error {
	// --- 0. 准备工作 ---
	var video model.Video
	// 从数据库获取视频的完整信息，包括了我们新加的 OriginalFileName
	if err := dal.DB.First(&video, videoID).Error; err != nil {
		return fmt.Errorf("video %d not found: %w", videoID, err)
	}

	tempDir, err := os.MkdirTemp("", fmt.Sprintf("video-%d-*", videoID))
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	bucketName := config.AppConfig.MinIO.BucketName

	// 【核心修改】使用 video.OriginalFileName 而不是 video.Title 来构建下载路径
	rawObjectName := filepath.Join("raw", fmt.Sprintf("%d", video.ID), video.OriginalFileName)
	// 本地保存的文件名也使用 OriginalFileName，保持一致性
	localRawPath := filepath.Join(tempDir, video.OriginalFileName)

	// 下载原始视频文件
	if err := dal.MinioClient.FGetObject(context.Background(), bucketName, rawObjectName, localRawPath, minio.GetObjectOptions{}); err != nil {
		// 下载失败，更新数据库状态并返回错误
		dal.DB.Model(&video).Update("status", "failed")
		// 在日志中明确指出是哪个对象键下载失败，方便排查
		return fmt.Errorf("failed to download from minio (key: %s): %w", rawObjectName, err)
	}
	log.Printf("Downloaded %s to %s", rawObjectName, localRawPath)

	// --- 后续的转码、截图、上传等逻辑保持不变 ---
	// ... (ffprobe, ffmpeg, FPutObject 等代码) ...
    // ...

	return nil // 确保函数结尾有返回值
}
```
这里的关键修改是 `rawObjectName` 和 `localRawPath` 的构建，现在它们都依赖于从数据库中取出的 `video.OriginalFileName`，从而保证了与上传时使用的文件名一致。

---

现在，你已经完成了所有必要的代码修改。请确保数据库表结构已更新，然后重新编译并运行你的 API 和 Worker 服务，再次执行测试脚本即可。