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