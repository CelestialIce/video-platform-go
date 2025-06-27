// internal/service/user_service.go
package service

import (
	"errors"
	"time"
	// -- FIX THESE LINES --
	"github.com/cjh/video-platform-go/internal/config"
	"github.com/cjh/video-platform-go/internal/dal"
	"github.com/cjh/video-platform-go/internal/dal/model"
	// ---------------------
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// ... rest of the file remains the same ...

// Register 处理用户注册逻辑
func Register(nickname, email, password string) (*model.User, error) {
	// 检查邮箱是否已被注册
	var existingUser model.User
	if err := dal.DB.Where("email = ?", email).First(&existingUser).Error; err == nil {
		return nil, errors.New("email already exists")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err // 其他数据库错误
	}

	// 哈希密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// 创建用户
	newUser := model.User{
		Nickname:       nickname,
		Email:          email,
		HashedPassword: string(hashedPassword),
	}

	if err := dal.DB.Create(&newUser).Error; err != nil {
		return nil, err
	}

	return &newUser, nil
}

// Login 处理用户登录逻辑
func Login(email, password string) (token string, userID uint64, nickname string, err error) {
	var user model.User
	if err = dal.DB.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = errors.New("invalid credentials")
		}
		return
	}

	// 校验密码
	if err = bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(password)); err != nil {
		err = errors.New("invalid credentials")
		return
	}

	// 生成 JWT
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"role":    user.Role,
		"exp":     time.Now().Add(time.Hour * time.Duration(config.AppConfig.JWT.ExpireHours)).Unix(),
		"iat":     time.Now().Unix(),
	}
	j := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err = j.SignedString([]byte(config.AppConfig.JWT.Secret))
	if err != nil {
		return
	}

	userID = user.ID
	nickname = user.Nickname
	return
}
