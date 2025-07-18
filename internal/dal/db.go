// internal/dal/db.go
package dal

import (
	"context"
	"fmt"
	"log"

	"github.com/cjh/video-platform-go/internal/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	DB          *gorm.DB
	MinioClient *minio.Client
)

// InitMySQL 初始化数据库连接 (V2版，兼容新配置)
func InitMySQL(cfg *config.Config) {
	var err error

	// 从独立的配置字段拼接 DSN 字符串
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.MySQL.User,
		cfg.MySQL.Password,
		cfg.MySQL.Host,
		cfg.MySQL.Port,
		cfg.MySQL.Database,
	)

	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	log.Println("Database connection established")
}

// InitMinIO 初始化 MinIO 客户端并设置存储桶为公开可读
func InitMinIO(cfg *config.Config) {
	var err error
	MinioClient, err = minio.New(cfg.MinIO.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinIO.AccessKeyID, cfg.MinIO.SecretAccessKey, ""),
		Secure: cfg.MinIO.UseSSL,
	})
	if err != nil {
		log.Fatalf("Failed to initialize minio client: %v", err)
	}

	// 检查存储桶是否存在，如果不存在则创建
	bucketName := cfg.MinIO.BucketName
	ctx := context.Background()
	exists, err := MinioClient.BucketExists(ctx, bucketName)
	if err != nil {
		log.Fatalf("Failed to check if bucket exists: %v", err)
	}
	if !exists {
		err = MinioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			log.Fatalf("Failed to create bucket: %v", err)
		}
		log.Printf("Bucket '%s' created successfully.", bucketName)
	} else {
		log.Printf("Bucket '%s' already exists.", bucketName)
	}

	// ***此处新增***
	// 设置存储桶策略为公开可读
	policy := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Principal": "*",
				"Action": ["s3:GetObject"],
				"Resource": ["arn:aws:s3:::` + bucketName + `/*"]
			}
		]
	}`
	err = MinioClient.SetBucketPolicy(ctx, bucketName, policy)
	if err != nil {
		log.Printf("Warning: Could not set bucket policy to public-read: %v", err)
	} else {
		log.Printf("Bucket '%s' set to public-read successfully", bucketName)
	}
}
