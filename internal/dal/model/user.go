// internal/dal/model/user.go
package model

import "time"

type User struct {
	ID             uint64    `gorm:"primaryKey;autoIncrement"`
	Nickname       string    `gorm:"type:varchar(50);not null"`
	Email          string    `gorm:"type:varchar(100);not null;unique"`
	HashedPassword string    `gorm:"type:varchar(255);not null"`
	Role           string    `gorm:"type:enum('user','admin','auditor');default:'user'"`
	CreatedAt      time.Time `gorm:"autoCreateTime"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime"`
}

// TableName 指定 GORM 应该使用的表名
func (User) TableName() string {
	return "users"
}