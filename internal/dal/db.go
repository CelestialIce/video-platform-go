// internal/dal/db.go
package dal

import (
	"context"
	"log"
	"github.com/cjh/video-platform-go/internal/config" // 确认是新的模块路径
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	DB          *gorm.DB
	MinioClient *minio.Client
)

// InitMySQL 初始化数据库连接
func InitMySQL(cfg *config.Config) {
	var err error
	dsn := cfg.MySQL.DSN
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	log.Println("Database connection established")
}

// InitMinIO 初始化 MinIO 客户端
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
}