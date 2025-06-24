// internal/dal/model/comment.go
package model

import "time"

// Comment 对应数据库中的 'comments' 表
type Comment struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement"`
	VideoID   uint64    `gorm:"not null;index:idx_video_timeline"`
	UserID    uint64    `gorm:"not null"`
	Content   string    `gorm:"type:text;not null"`
	Timeline  *uint     `gorm:"comment:弹幕出现时间点，单位秒; 若为普通评论则为NULL"` // 使用指针 *uint 来允许 NULL 值
	CreatedAt time.Time `gorm:"autoCreateTime"`

	User      User      `gorm:"foreignKey:UserID"` // <-- 新增这一行，建立关联
}

func (Comment) TableName() string {
	return "comments"
}