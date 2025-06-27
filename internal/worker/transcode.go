// internal/worker/transcode.go
package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/cjh/video-platform-go/internal/config"
	"github.com/cjh/video-platform-go/internal/dal"
	"github.com/cjh/video-platform-go/internal/dal/model"
	"github.com/minio/minio-go/v7"
)

// ffprobe 用于解析视频信息的结构体
type ffprobeFormat struct {
	Duration string `json:"duration"`
}
type ffprobeOutput struct {
	Format ffprobeFormat `json:"format"`
}

// HandleTranscode 是处理转码任务的核心函数 (V2版)
func HandleTranscode(videoID uint64) error {
	// --- 0. 准备工作 ---
	var video model.Video
	// 从数据库获取视频的完整信息，包括了我们新加的 OriginalFileName
	if err := dal.DB.First(&video, videoID).Error; err != nil {
		return fmt.Errorf("video %d not found: %w", videoID, err)
	}

	tempDir, err := os.MkdirTemp("", fmt.Sprintf("video-%d-*", videoID))
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	bucketName := config.AppConfig.MinIO.BucketName

	// 【核心修改】使用 video.OriginalFileName 而不是 video.Title 来构建下载路径
	rawObjectName := filepath.Join("raw", fmt.Sprintf("%d", video.ID), video.OriginalFileName)
	// 本地保存的文件名也使用 OriginalFileName，保持一致性
	localRawPath := filepath.Join(tempDir, video.OriginalFileName)

	// 下载原始视频文件
	if err := dal.MinioClient.FGetObject(context.Background(), bucketName, rawObjectName, localRawPath, minio.GetObjectOptions{}); err != nil {
		// 下载失败，更新数据库状态并返回错误
		dal.DB.Model(&video).Update("status", "failed")
		// 在日志中明确指出是哪个对象键下载失败，方便排查
		return fmt.Errorf("failed to download from minio (key: %s): %w", rawObjectName, err)
	}
	log.Printf("Downloaded %s to %s", rawObjectName, localRawPath)

	// --- 1. 获取视频信息 (时长和封面) ---
	// 1.1 获取时长
	cmdProbe := exec.Command("ffprobe", "-v", "quiet", "-print_format", "json", "-show_format", localRawPath)
	outputProbe, err := cmdProbe.CombinedOutput()
	if err != nil {
		log.Printf("ffprobe failed: %s", string(outputProbe))
		return fmt.Errorf("ffprobe command failed: %w", err)
	}
	var probeData ffprobeOutput
	json.Unmarshal(outputProbe, &probeData)
	durationFloat, _ := strconv.ParseFloat(probeData.Format.Duration, 64)
	durationUint := uint(durationFloat)

	// 1.2 截取封面图 (视频第1秒)
	coverPath := filepath.Join(tempDir, "cover.jpg")
	cmdCover := exec.Command("ffmpeg", "-i", localRawPath, "-ss", "00:00:01.000", "-vframes", "1", coverPath)
	if outputCover, err := cmdCover.CombinedOutput(); err != nil {
		log.Printf("Failed to generate cover: %s", string(outputCover))
		// 封面生成失败不是致命错误，可以继续
	}

	// 1.3 上传封面图
	coverObjectName := filepath.ToSlash(filepath.Join("processed", fmt.Sprintf("%d", videoID), "cover.jpg"))
	if _, err := dal.MinioClient.FPutObject(context.Background(), bucketName, coverObjectName, coverPath, minio.PutObjectOptions{}); err != nil {
		log.Printf("Failed to upload cover: %v", err)
		// 上传失败也不是致命错误
	}

	// --- 2. 循环执行多码率转码 ---
	profiles := config.AppConfig.FFMpeg.Profiles
	var newVideoSources []model.VideoSource

	for _, profile := range profiles {
		outputDir := filepath.Join(tempDir, fmt.Sprintf("hls_%s", profile.Name))
		os.Mkdir(outputDir, 0755)
		outputM3u8 := filepath.Join(outputDir, fmt.Sprintf("%s.m3u8", profile.Name))

		cmdTranscode := exec.Command("ffmpeg",
			"-i", localRawPath,
			"-c:v", "libx264", "-c:a", "aac",
			"-vf", "scale="+profile.Resolution,
			"-hls_time", "10", "-hls_list_size", "0",
			"-f", "hls", outputM3u8,
		)

		log.Printf("Executing ffmpeg for profile %s: %s", profile.Name, cmdTranscode.String())
		if output, err := cmdTranscode.CombinedOutput(); err != nil {
			log.Printf("FFMPEG error for profile %s: %s", profile.Name, string(output))
			dal.DB.Model(&video).Update("status", "failed")
			return fmt.Errorf("ffmpeg command failed for profile %s: %w", profile.Name, err)
		}

		// 上传转码后的文件
		processedPathPrefix := filepath.ToSlash(filepath.Join("processed", fmt.Sprintf("%d", videoID), fmt.Sprintf("hls_%s", profile.Name)))
		files, _ := os.ReadDir(outputDir)
		var totalSize uint64 // <-- 新增：用于累加文件大小
		for _, file := range files {
			localFilePath := filepath.Join(outputDir, file.Name())

			// 获取文件信息以得到大小
			fileInfo, err := os.Stat(localFilePath)
			if err == nil {
				totalSize += uint64(fileInfo.Size()) // <-- 新增：累加大小
			}

			_, err = dal.MinioClient.FPutObject(context.Background(), bucketName,
				filepath.ToSlash(filepath.Join(processedPathPrefix, file.Name())),
				localFilePath,
				minio.PutObjectOptions{},
			)
			if err != nil {
				dal.DB.Model(&video).Update("status", "failed")
				return fmt.Errorf("failed to upload HLS file %s: %w", file.Name(), err)
			}
		}

		// 准备要写入数据库的 video_source
		newVideoSources = append(newVideoSources, model.VideoSource{
			VideoID:  video.ID,
			Quality:  profile.Name,
			Format:   "HLS",
			URL:      filepath.ToSlash(filepath.Join(processedPathPrefix, fmt.Sprintf("%s.m3u8", profile.Name))),
			FileSize: totalSize, // <-- 新增：填充文件大小
		})
	}

	// --- 3. 使用数据库事务，一次性更新所有信息 ---
	tx := dal.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// 3.1 更新主视频表信息 (时长, 封面, 状态)
	updates := map[string]interface{}{
		"status":    "online",
		"duration":  durationUint,
		"cover_url": filepath.ToSlash(coverObjectName),
	}
	if err := tx.Model(&video).Updates(updates).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 3.2 批量创建视频源记录
	if err := tx.Create(&newVideoSources).Error; err != nil {
		tx.Rollback()
		return err
	}

	log.Println("Successfully updated database in a transaction.")
	return tx.Commit().Error
}
