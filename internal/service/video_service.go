// internal/service/video_service.go
package service

import (
	"context"
	"path/filepath"
	"time"
	"github.com/cjh/video-platform-go/internal/config"
	"github.com/cjh/video-platform-go/internal/dal"
	"github.com/cjh/video-platform-go/internal/dal/model"
	// "github.com/minio/minio-go/v7" - Unused import
	"encoding/json" // 确保导入
	"fmt"             // 确保导入
)

// TranscodeTaskPayload 是我们要发送到消息队列的任务内容
type TranscodeTaskPayload struct {
	VideoID uint64 `json:"video_id"`
}

// CompleteUploadService 处理“完成上传”的逻辑
func CompleteUploadService(videoID uint64) error {
	// 1. 验证视频是否存在且状态正确
	var video model.Video
	if err := dal.DB.First(&video, videoID).Error; err != nil {
		return fmt.Errorf("video with id %d not found", videoID)
	}
	if video.Status != "uploading" {
		return fmt.Errorf("video status is not 'uploading'")
	}

	// 2. 更新视频状态为 'transcoding'（准备中）
	if err := dal.DB.Model(&video).Update("status", "transcoding").Error; err != nil {
		return err
	}

	// 3. 创建任务并发送到 RabbitMQ
	task := TranscodeTaskPayload{VideoID: videoID}
	body, err := json.Marshal(task)
	if err != nil {
		// 如果序列化失败，最好把状态改回来，或者标记为失败
		dal.DB.Model(&video).Update("status", "failed")
		return fmt.Errorf("failed to create transcode task: %v", err)
	}

	return dal.PublishTranscodeTask(context.Background(), body)
}

// InitiateUploadService 处理视频上传初始化的逻辑
func InitiateUploadService(userID uint64, fileName string) (string, *model.Video, error) {
	// 1. 创建视频记录
	video := model.Video{
		UserID: userID,
		Title:  fileName, // 暂时用文件名作为标题
		Status: "uploading",
	}
	if err := dal.DB.Create(&video).Error; err != nil {
		return "", nil, err
	}

	// 2. 生成预签名 URL
	// 对象在 MinIO 中的存储路径，例如: raw/123/video.mp4
	objectName := filepath.Join("raw", fmt.Sprintf("%d", video.ID), fileName)
	bucketName := config.AppConfig.MinIO.BucketName
	expiration := time.Hour * 24 // URL 有效期 24 小时

	presignedURL, err := dal.MinioClient.PresignedPutObject(context.Background(), bucketName, objectName, expiration)
	if err != nil {
		return "", nil, err
	}

	return presignedURL.String(), &video, nil
}

// ... 之后我们会在这里添加 CompleteUploadService ...