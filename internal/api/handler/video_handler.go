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