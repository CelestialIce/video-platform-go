// internal/dal/model/video.go
package model

import "time"

// internal/dal/model/video.go
type Video struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID      uint64    `gorm:"not null"                 json:"user_id"`
	Title       string    `gorm:"type:varchar(255);not null" json:"title"`
	Description string    `gorm:"type:text"                json:"description"`
	Status      string    `gorm:"type:enum('uploading','transcoding','online','failed','private');default:'uploading'" json:"status"`
	Duration    uint      `json:"duration"`
	CoverURL    string    `gorm:"type:varchar(1024)"       json:"cover_url"`
	CreatedAt   time.Time `gorm:"autoCreateTime"           json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"           json:"updated_at"`
}



func (Video) TableName() string {
	return "videos"
}

type VideoSource struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	VideoID   uint64    `gorm:"not null;uniqueIndex:uk_video_quality" json:"video_id"`
	Quality   string    `gorm:"type:varchar(20);not null;uniqueIndex:uk_video_quality" json:"quality"`
	Format    string    `gorm:"type:varchar(20);not null" json:"format"`
	URL       string    `gorm:"type:varchar(1024);not null" json:"url"`
	FileSize  uint64    `json:"file_size"`
	CreatedAt time.Time `gorm:"autoCreateTime"            json:"created_at"`
}

func (VideoSource) TableName() string {
	return "video_sources"
}	