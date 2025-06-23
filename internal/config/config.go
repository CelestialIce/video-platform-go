// internal/config/config.go
package config

import (
	"log"
	"github.com/spf13/viper"
)

// 全局配置变量
var AppConfig Config

// Config 结构体，与 config.yaml 文件对应
type Config struct {
	Server struct {
		Port string `mapstructure:"port"`
	} `mapstructure:"server"`
	MySQL struct {
		DSN string `mapstructure:"dsn"`
	} `mapstructure:"mysql"`
	Redis struct {
		Addr     string `mapstructure:"addr"`
		Password string `mapstructure:"password"`
		DB       int    `mapstructure:"db"`
	} `mapstructure:"redis"`
	JWT struct {
		Secret      string `mapstructure:"secret"`
		ExpireHours int    `mapstructure:"expire_hours"`
	} `mapstructure:"jwt"`
	MinIO struct {
		Endpoint        string `mapstructure:"endpoint"`
		AccessKeyID     string `mapstructure:"access_key_id"`
		SecretAccessKey string `mapstructure:"secret_access_key"`
		UseSSL          bool   `mapstructure:"use_ssl"`
		BucketName      string `mapstructure:"bucket_name"`
	} `mapstructure:"minio"`
	RabbitMQ struct {
		URL            string `mapstructure:"url"`
		TranscodeQueue string `mapstructure:"transcode_queue"`
	} `mapstructure:"rabbitmq"`
}

// Init 函数用于初始化配置加载
func Init() {
	viper.SetConfigName("config")    // 配置文件名 (不带扩展名)
	viper.SetConfigType("yaml")      // 配置文件类型
	viper.AddConfigPath("./configs") // 配置文件路径

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}

	if err := viper.Unmarshal(&AppConfig); err != nil {
		log.Fatalf("Unable to decode into struct, %v", err)
	}
}