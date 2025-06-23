// internal/dal/model/video.go
package model

import "time"

// Video 对应数据库中的 'videos' 表
type Video struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement"`
	UserID      uint64    `gorm:"not null"`
	Title       string    `gorm:"type:varchar(255);not null"`
	Description string    `gorm:"type:text"`
	Status      string    `gorm:"type:enum('uploading','transcoding','online','failed','private');default:'uploading'"`
	Duration    uint      `gorm:"comment:视频时长，单位秒"`
	CoverURL    string    `gorm:"type:varchar(1024)"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

func (Video) TableName() string {
	return "videos"
}

// VideoSource 对应数据库中的 'video_sources' 表
type VideoSource struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement"`
	VideoID   uint64    `gorm:"not null;uniqueIndex:uk_video_quality"`
	Quality   string    `gorm:"type:varchar(20);not null;uniqueIndex:uk_video_quality;comment:例如: 360p, 720p, 1080p"`
	Format    string    `gorm:"type:varchar(20);not null;comment:例如: HLS, DASH, MP4"`
	URL       string    `gorm:"type:varchar(1024);not null;comment:播放地址, M3U8文件或MP4文件"`
	FileSize  uint64    `gorm:"comment:文件大小，单位字节"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

func (VideoSource) TableName() string {
	return "video_sources"
}