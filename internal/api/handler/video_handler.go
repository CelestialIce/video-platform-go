// internal/api/handler/video_handler.go
package handler

import (
	"net/http"
	"github.com/cjh/video-platform-go/internal/service"
	"github.com/gin-gonic/gin"
	"strconv" // 确保导入
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