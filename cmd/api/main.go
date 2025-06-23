// cmd/api/main.go
package main


import (
	"log"
	"net/http"
	// -- FIX THESE LINES --
	"github.com/cjh/video-platform-go/internal/api/handler"
	"github.com/cjh/video-platform-go/internal/api/middleware"
	"github.com/cjh/video-platform-go/internal/config"
	"github.com/cjh/video-platform-go/internal/dal"
	// ---------------------
	"github.com/gin-gonic/gin"
)


func main() {
	// 1. 初始化配置
	config.Init()
	log.Println("Configuration loaded")

	// 2. 初始化数据库
	dal.InitMySQL(&config.AppConfig)
	dal.InitMinIO(&config.AppConfig) // <-- 新增这一行
	log.Println("Database initialized")

	// 3. 设置 Gin 引擎
	r := gin.Default()

	// 4. 设置路由
	// API 版本分组
	apiV1 := r.Group("/api/v1")
	{
		// 用户路由 (无需认证)
		userRoutes := apiV1.Group("/users")
		{
			userRoutes.POST("/register", handler.Register)
			userRoutes.POST("/login", handler.Login)
		}

		// 需要认证的路由组
		authed := apiV1.Group("/")
		authed.Use(middleware.JWTAuthMiddleware()) // 应用 JWT 中间件
		{
			// 测试路由，用于验证 token
			authed.GET("/me", func(c *gin.Context) {
				userID, _ := c.Get("user_id")
				role, _ := c.Get("role")
				c.JSON(http.StatusOK, gin.H{
					"message": "Token is valid",
					"user_id": userID,
					"role":    role,
				})
			})
			// 视频路由
			videoRoutes := authed.Group("/videos")
			{
				// POST /api/v1/videos/upload/initiate
				videoRoutes.POST("/upload/initiate", handler.InitiateUpload)
			}
			}
	}

	// 5. 启动服务器
	log.Printf("Starting server on port %s", config.AppConfig.Server.Port)
	if err := r.Run(config.AppConfig.Server.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}