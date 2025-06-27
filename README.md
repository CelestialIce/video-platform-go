# 基于 Go 的视频平台 MVP (Video Platform MVP in Go)

这是一个使用 Go 语言构建的视频点播（VOD）平台后端的最小可行产品（MVP）。项目采用现代化、可扩展的架构，实现了一个从视频上传、异步转码到安全播放的完整业务流程。

## 技术栈 (Core Technologies)

- **后端语言**: Go (Golang)
- **Web 框架**: Gin
- **数据库 ORM**: GORM
- **数据库**: MySQL 8.0
- **对象存储**: MinIO (S3 兼容)
- **消息队列**: RabbitMQ
- **媒体处理**: FFmpeg
- **核心依赖**: Docker & Docker Compose

## 已实现功能 (Features)

- **用户系统**:
    - [x] 用户注册 (`/register`)
    - [x] 用户登录 (`/login`)
    - [x] 基于 JWT (JSON Web Token) 的接口认证与授权

- **视频点播 (VOD) 流水线**:
    - [x] **安全上传**: 客户端请求预签名 URL，将视频文件直接上传至 MinIO，不占用服务器带宽。
    - [x] **异步处理**: 上传完成后，通过 RabbitMQ 消息队列触发后台转码任务，API 接口立即响应，不阻塞用户。
    - [x] **后台转码 Worker**: 独立的 Worker 程序消费转码任务，使用 `ffmpeg` 将视频转为 HLS (`.m3u8`) 格式。
    - [x] **数据持久化**: 视频元数据、用户数据、评论等存储在 MySQL 中。转码完成的视频源信息也会被记录。

- **视频播放与互动**:
    - [x] **视频列表与详情**: 提供公开的 API 接口，用于查询视频列表（分页）和单个视频的详细信息。
    - [x] **安全播放**: 视频播放地址通过动态生成的预签名 URL 提供，有效防盗链，保护内容安全。
    - [x] **评论系统**: 登录用户可以对视频发表评论；任何人都可以查看评论列表。

## 项目结构 (Project Structure)

```
.
├── cmd/                # 主程序入口
│   ├── api/            # API 服务器 (Gin)
│   │   └── main.go
│   └── worker/         # 后台转码 Worker
│       └── main.go
├── configs/            # 配置文件目录
│   └── config.yaml
├── internal/           # 项目内部代码 (不对外暴露)
│   ├── api/            # API 层的路由、处理器、中间件
│   ├── config/         # 配置加载 (Viper)
│   ├── dal/            # 数据访问层 (GORM, MinIO, RabbitMQ)
│   ├── service/        # 业务逻辑层
│   └── worker/         # Worker 的核心处理逻辑 (FFmpeg)
├── docker-compose.yml  # docker-compose 配置文件
├── go.mod              # Go 模块依赖管理
├── go.sum              # Go 模块依赖校验
└── README.md           # 项目说明文档
```

## 如何构建与运行 (Getting Started)

#### 1. 前提条件
- 已安装 Go (版本 1.18+)
- 已安装 Docker 和 Docker Compose
- 已安装 `ffmpeg` (在你的系统上全局可用)
  ```bash
  # For Ubuntu/Debian
  sudo apt-get update && sudo apt-get install ffmpeg
  ```

#### 2. 配置
- 复制或重命名 `configs/config.yaml.example` (如果存在) 为 `config.yaml`。
- 打开 `configs/config.yaml`，根据你的环境修改 `mysql` 的密码和 `jwt` 的密钥。

#### 3. 启动基础设施
在项目根目录下，一键启动 MySQL, Redis, RabbitMQ, MinIO 和 SRS。
```bash
docker-compose up -d
```

#### 4. 初始化数据库
- 使用你喜欢的数据库客户端 (如 Navicat, DBeaver) 连接到 `localhost:3306`。
- 执行项目初期提供的 SQL DDL 脚本，创建 `video_platform_mvp` 数据库及所有表。

#### 5. 运行后端服务
你需要**打开两个独立的终端**来分别运行 API 服务器和 Worker。

- **终端 1: 启动 API 服务器**
  ```bash
  go run ./cmd/api/main.go
  ```
  成功后，你将看到日志显示服务器正在监听 `:8000` 端口。

- **终端 2: 启动转码 Worker**
  ```bash
  go run ./cmd/worker/main.go
  ```
  成功后，你将看到日志显示 "Waiting for messages..."。

## API 接口文档 (API Endpoints)

API Base URL: `http://localhost:8000/api/v1`

---

### 用户模块

| Method | Endpoint              | 认证 | 请求体 (Body)                                        | 描述           |
|--------|-----------------------|------|------------------------------------------------------|----------------|
| `POST` | `/users/register`     | 否   | `{"nickname": "u", "email": "e@e.com", "password": "p"}` | 用户注册       |
| `POST` | `/users/login`        | 否   | `{"email": "e@e.com", "password": "p"}`                | 用户登录，返回 JWT |

---

### 视频模块

| Method | Endpoint                  | 认证 | 请求体 (Body)                             | 描述                                     |
|--------|---------------------------|------|-------------------------------------------|------------------------------------------|
| `POST` | `/videos/upload/initiate` | 是   | `{"file_name": "my-video.mp4"}`           | 申请预签名上传 URL                       |
| `POST` | `/videos/upload/complete` | 是   | `{"video_id": 1}`                         | 通知服务器上传完成，触发转码             |
| `GET`  | `/videos`                 | 否   | *无* (Query: `limit`, `offset`)         | 获取已上线的视频列表                     |
| `GET`  | `/videos/:id`             | 否   | *无*                                      | 获取单个视频详情和带签名的播放地址       |

---

### 评论模块

| Method | Endpoint                 | 认证 | 请求体 (Body)                             | 描述                         |
|--------|--------------------------|------|-------------------------------------------|------------------------------|
| `POST` | `/videos/:id/comments`   | 是   | `{"content": "...", "timeline": 10}`      | 对视频发表评论或弹幕         |
| `GET`  | `/videos/:id/comments`   | 否   | *无*                                      | 获取指定视频的所有评论       |

---

## 核心业务流程：视频上传与转码

1.  **客户端** (已登录) 调用 `POST /videos/upload/initiate`，请求上传。
2.  **API 服务器** 在数据库 `videos` 表创建一条记录 (状态为 `uploading`)，并向 MinIO 请求一个预签名 `PUT` URL。
3.  **客户端** 拿到 URL 后，直接将视频文件 `PUT` 到 MinIO。
4.  上传成功后，**客户端** 调用 `POST /videos/upload/complete`，并附上 `video_id`。
5.  **API 服务器** 将 `videos` 表中的状态更新为 `transcoding`，然后向 RabbitMQ 的 `video_transcoding_queue` 队列中发布一条包含 `video_id` 的任务消息。
6.  **Worker 程序** 监听到该消息，从 MinIO 下载原始视频，使用 `ffmpeg` 转码成 HLS 格式，再将 `.m3u8` 和 `.ts` 文件上传回 MinIO。
7.  **Worker 程序** 将转码结果（播放地址等）写入 `video_sources` 表，并将 `videos` 表的状态更新为 `online`。任务完成。

## 后续计划 (Next Steps)

- [ ] 实现多码率转码和封面图截取
- [ ] 对接 SRS 实现视频直播功能
- [ ] 编写 Dockerfile，将项目服务容器化部署
- [ ] 引入搜索引擎 (Elasticsearch / MeiliSearch)