// internal/service/video_service.go
package service

import (
	"context"
	"log"
	"path/filepath"
	"time"

	"github.com/cjh/video-platform-go/internal/config"
	"github.com/cjh/video-platform-go/internal/dal"
	"github.com/cjh/video-platform-go/internal/dal/model"

	// "github.com/minio/minio-go/v7" - Unused import
	"encoding/json" // 确保导入
	"fmt"           // 确保导入
	"net/url"       // 确保导入
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

// 新签名：InitiateUploadService(userID uint64, fileName, title, description string)
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

// ListVideosService 获取视频列表（带分页）
func ListVideosService(limit, offset int) ([]model.Video, int64, error) {
	var videos []model.Video
	var total int64

	// 我们只展示状态为 'online' 的视频
	db := dal.DB.Model(&model.Video{}).Where("status = ?", "online")

	// 计算总数
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取分页数据
	if err := db.Order("created_at desc").Limit(limit).Offset(offset).Find(&videos).Error; err != nil {
		return nil, 0, err
	}

	// 为每个视频生成带签名的临时 CoverURL
	for i := range videos {
		if videos[i].CoverURL != "" {
			reqParams := make(url.Values)
			// 生成带签名的临时 URL
			presignedURL, err := dal.MinioClient.PresignedGetObject(context.Background(),
				config.AppConfig.MinIO.BucketName,
				videos[i].CoverURL, // CoverURL 里存的是对象路径
				time.Minute*15,     // 设置一个较短的有效期，例如15分钟
				reqParams,
			)
			if err != nil {
				log.Printf("Failed to generate presigned cover url: %v", err)
			}
			// 用签名的 URL 替换掉数据库里的永久路径
			videos[i].CoverURL = presignedURL.String()
		}
	}

	return videos, total, nil
}

// GetVideoDetailsService 获取单个视频的详细信息，包括它的所有可用播放源
func GetVideoDetailsService(videoID uint64) (*model.Video, []model.VideoSource, error) {
	var video model.Video
	if err := dal.DB.First(&video, videoID).Error; err != nil {
		return nil, nil, fmt.Errorf("video not found: %w", err)
	}

	var sources []model.VideoSource
	if err := dal.DB.Where("video_id = ?", videoID).Find(&sources).Error; err != nil {
		return nil, nil, err
	}

	// 为每个播放源生成带签名的临时 URL
	for i := range sources {
		reqParams := make(url.Values)
		// 如果你的 MinIO 桶是公开读的，这一步可以省略
		// 但为了安全，桶应该是私有的，所有访问都通过签名 URL
		presignedURL, err := dal.MinioClient.PresignedGetObject(context.Background(),
			config.AppConfig.MinIO.BucketName,
			sources[i].URL, // sources[i].URL 里存的是对象路径，例如 processed/1/hls_720p/720p.m3u8
			time.Minute*15, // 设置一个较短的有效期，例如15分钟
			reqParams,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to generate presigned url for source %s: %w", sources[i].URL, err)
		}
		// 用签名的 URL 替换掉数据库里的永久路径
		sources[i].URL = presignedURL.String()
	}

	return &video, sources, nil
}
