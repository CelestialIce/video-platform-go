// internal/service/comment_service.go
package service

import (
	"log"

	"github.com/cjh/video-platform-go/internal/dal"
	"github.com/cjh/video-platform-go/internal/dal/model"
)

// CreateCommentService 创建评论 (V2版，返回带用户信息)
func CreateCommentService(userID, videoID uint64, content string, timeline *uint) (*model.Comment, error) {
	comment := model.Comment{
		UserID:   userID,
		VideoID:  videoID,
		Content:  content,
		Timeline: timeline,
	}

	// 1. 先创建评论
	if err := dal.DB.Create(&comment).Error; err != nil {
		return nil, err
	}

	// 2. 创建成功后，使用 Preload 重新查询，以加载 User 信息
	if err := dal.DB.Preload("User").First(&comment, comment.ID).Error; err != nil {
		// 即使查询失败，评论也已创建成功，所以只记录错误，但返回已创建的 comment
		log.Printf("Failed to preload user for new comment: %v", err)
	}
	
	return &comment, nil
}

// ListCommentsService 获取视频的评论列表 (V2版，带用户信息)
func ListCommentsService(videoID uint64) ([]model.Comment, error) {
	var comments []model.Comment
	// 使用 Preload("User") 来预加载关联的用户数据
	err := dal.DB.Preload("User").Where("video_id = ?", videoID).Order("created_at asc").Find(&comments).Error
	return comments, err
}