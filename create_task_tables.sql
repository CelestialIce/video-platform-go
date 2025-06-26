-- 创建数据库
CREATE DATABASE IF NOT EXISTS `video_platform_mvp` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE `video_platform_mvp`;

-- 用户表
CREATE TABLE `users` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `nickname` VARCHAR(50) NOT NULL,
  `email` VARCHAR(100) NOT NULL UNIQUE,
  `hashed_password` VARCHAR(255) NOT NULL,
  `role` ENUM('user', 'admin', 'auditor') NOT NULL DEFAULT 'user',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  INDEX `idx_email` (`email`)
) ENGINE=InnoDB;

-- 视频主表
CREATE TABLE `videos` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL,
  `title` VARCHAR(255) NOT NULL,
  `description` TEXT,
  `status` ENUM('uploading', 'transcoding', 'online', 'failed', 'private') NOT NULL DEFAULT 'uploading',
  `duration` INT UNSIGNED COMMENT '视频时长，单位秒',
  `cover_url` VARCHAR(1024),
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

-- 视频源表 (多清晰度)
CREATE TABLE `video_sources` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `video_id` BIGINT UNSIGNED NOT NULL,
  `quality` VARCHAR(20) NOT NULL COMMENT '例如: 360p, 720p, 1080p',
  `format` VARCHAR(20) NOT NULL COMMENT '例如: HLS, DASH, MP4',
  `url` VARCHAR(1024) NOT NULL COMMENT '播放地址, M3U8文件或MP4文件',
  `file_size` BIGINT UNSIGNED COMMENT '文件大小，单位字节',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_video_quality` (`video_id`, `quality`),
  FOREIGN KEY (`video_id`) REFERENCES `videos`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB;

-- 评论/弹幕表
CREATE TABLE `comments` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `video_id` BIGINT UNSIGNED NOT NULL,
  `user_id` BIGINT UNSIGNED NOT NULL,
  `content` TEXT NOT NULL,
  `timeline` INT UNSIGNED COMMENT '弹幕出现时间点，单位秒; 若为普通评论则为NULL',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  FOREIGN KEY (`video_id`) REFERENCES `videos`(`id`) ON DELETE CASCADE,
  FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE,
  INDEX `idx_video_timeline` (`video_id`, `timeline`)
) ENGINE=InnoDB;

-- 触发器示例：更新视频表的 updated_at
-- (在实际应用中，ORM 或框架通常会自动处理，但为满足报告要求可以写一个)
DELIMITER $$
CREATE TRIGGER `trg_videos_update`
BEFORE UPDATE ON `videos`
FOR EACH ROW
BEGIN
    SET NEW.updated_at = CURRENT_TIMESTAMP;
END$$
DELIMITER ;

ALTER TABLE videos ADD COLUMN original_file_name VARCHAR(255) NOT NULL AFTER description;