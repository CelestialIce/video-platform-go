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

// ListCommentsService 获取视频的评论列表 (V2版，带用户信息)
func ListCommentsService(videoID uint64) ([]model.Comment, error) {
	var comments []model.Comment
	// 使用 Preload("User") 来预加载关联的用户数据
	err := dal.DB.Preload("User").Where("video_id = ?", videoID).Order("created_at asc").Find(&comments).Error
	return comments, err
}