// internal/worker/transcode.go
package worker

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"github.com/cjh/video-platform-go/internal/config"
	"github.com/cjh/video-platform-go/internal/dal"
	"github.com/cjh/video-platform-go/internal/dal/model"
	"github.com/minio/minio-go/v7"
)

// HandleTranscode 是处理转码任务的核心函数
func HandleTranscode(videoID uint64) error {
	// 0. 从数据库获取视频信息
	var video model.Video
	if err := dal.DB.First(&video, videoID).Error; err != nil {
		return fmt.Errorf("video %d not found: %w", videoID, err)
	}

	// 1. 创建临时工作目录
	tempDir, err := os.MkdirTemp("", fmt.Sprintf("video-%d-*", videoID))
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir) // 保证函数结束时删除临时目录

	// 2. 从 MinIO 下载原始视频
	bucketName := config.AppConfig.MinIO.BucketName
	// 原始文件路径我们当时是这样设计的：raw/{video_id}/{file_name}
	// 但 file_name 我们没有存，可以从 video.Title 读取（因为我们用它做了标题）
	// 更稳妥的做法是在 videos 表加一个 original_object_name 字段
	// 这里我们先用 Title 简化处理
	rawObjectName := filepath.Join("raw", fmt.Sprintf("%d", video.ID), video.Title)
	localRawPath := filepath.Join(tempDir, video.Title)

	err = dal.MinioClient.FGetObject(context.Background(), bucketName, rawObjectName, localRawPath, minio.GetObjectOptions{})
	if err != nil {
		dal.DB.Model(&video).Update("status", "failed")
		return fmt.Errorf("failed to download from minio: %w", err)
	}
	log.Printf("Downloaded %s to %s", rawObjectName, localRawPath)

	// 3. 执行 FFMPEG 转码 (以720p为例)
	outputDir := filepath.Join(tempDir, "hls_720p")
	os.Mkdir(outputDir, 0755)
	outputM3u8 := filepath.Join(outputDir, "720p.m3u8")

	// ffmpeg -i [输入文件] -c:v libx264 -c:a aac -vf "scale=-2:720" -hls_time 10 -hls_list_size 0 -f hls [输出.m3u8]
	cmd := exec.Command("ffmpeg",
		"-i", localRawPath,
		"-c:v", "libx264",
		"-c:a", "aac",
		"-vf", "scale=-2:720", // 保持宽高比，高度为720p
		"-hls_time", "10", // 每个 ts 切片 10 秒
		"-hls_list_size", "0", // 0表示保留所有切片
		"-f", "hls",
		outputM3u8,
	)

	log.Printf("Executing ffmpeg command: %s", cmd.String())
	output, err := cmd.CombinedOutput() // 获取标准输出和错误输出
	if err != nil {
		log.Printf("FFMPEG error output: %s", string(output))
		dal.DB.Model(&video).Update("status", "failed")
		return fmt.Errorf("ffmpeg command failed: %w", err)
	}
	log.Printf("FFMPEG success output: %s", string(output))

	// 4. 将转码后的 HLS 文件上传到 MinIO
	processedPathPrefix := filepath.Join("processed", fmt.Sprintf("%d", video.ID), "hls_720p")

	files, err := os.ReadDir(outputDir)
	if err != nil {
		return fmt.Errorf("failed to read HLS output dir: %w", err)
	}

	for _, file := range files {
		localFilePath := filepath.Join(outputDir, file.Name())
		remoteObjectName := filepath.Join(processedPathPrefix, file.Name())

		_, err := dal.MinioClient.FPutObject(context.Background(), bucketName, remoteObjectName, localFilePath, minio.PutObjectOptions{})
		if err != nil {
			dal.DB.Model(&video).Update("status", "failed")
			return fmt.Errorf("failed to upload HLS file %s: %w", file.Name(), err)
		}
	}
	log.Printf("Uploaded HLS files to %s", processedPathPrefix)

	// 5. 更新数据库
	// 开启一个事务来确保数据一致性
	tx := dal.DB.Begin()
	m3u8URL := filepath.ToSlash(filepath.Join(processedPathPrefix, "720p.m3u8")) // 确保是 / 分隔符
	// 创建视频源记录
	videoSource := model.VideoSource{
		VideoID: video.ID,
		Quality: "720p",
		Format:  "HLS",
		URL:     m3u8URL,
	}
	if err := tx.Create(&videoSource).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 更新主视频表状态
	if err := tx.Model(&video).Update("status", "online").Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}