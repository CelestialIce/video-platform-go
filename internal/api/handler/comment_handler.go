// internal/api/handler/comment_handler.go
package handler

import (
	"net/http"
	"strconv"
	"time"
	"github.com/cjh/video-platform-go/internal/service"
	"github.com/gin-gonic/gin"
)

type CreateCommentRequest struct {
	Content  string `json:"content" binding:"required"`
	Timeline *uint  `json:"timeline"` // 弹幕时间点，可选
}

// CommentInfo 是我们要返回给前端的评论结构
type CommentInfo struct {
	ID        uint64    `json:"id"`
	Content   string    `json:"content"`
	Timeline  *uint     `json:"timeline,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	User      struct {
		ID       uint64 `json:"id"`
		Nickname string `json:"nickname"`
	} `json:"user"`
}

// CreateComment 创建评论或弹幕 (V2版，返回一致的结构)
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

	// 构造和 ListComments 一致的响应结构
	response := CommentInfo{
		ID:        comment.ID,
		Content:   comment.Content,
		Timeline:  comment.Timeline,
		CreatedAt: comment.CreatedAt,
	}
	response.User.ID = comment.User.ID
	response.User.Nickname = comment.User.Nickname

	c.JSON(http.StatusCreated, response)
}

// ListComments 获取评论列表 (V2版)
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

	// 构造响应数据，避免暴露过多用户信息
	var response []CommentInfo
	for _, comment := range comments {
		info := CommentInfo{
			ID:        comment.ID,
			Content:   comment.Content,
			Timeline:  comment.Timeline,
			CreatedAt: comment.CreatedAt,
		}
		info.User.ID = comment.User.ID
		info.User.Nickname = comment.User.Nickname
		response = append(response, info)
	}

	c.JSON(http.StatusOK, response)
}