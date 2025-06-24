-- Active: 1750660870702@@127.0.0.1@3306@video_platform_mvp
-- Go-Admin所需的系统表
CREATE TABLE IF NOT EXISTS `goadmin_session` (
  `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
  `sid` varchar(50) NOT NULL DEFAULT '',
  `values` varchar(3000) NOT NULL DEFAULT '',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `sid` (`sid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `goadmin_users` (
  `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
  `username` varchar(100) NOT NULL DEFAULT '',
  `password` varchar(100) NOT NULL DEFAULT '',
  `name` varchar(100) NOT NULL DEFAULT '',
  `avatar` varchar(255) DEFAULT NULL,
  `remember_token` varchar(100) DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `username` (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `goadmin_roles` (
  `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(50) NOT NULL DEFAULT '',
  `slug` varchar(50) NOT NULL DEFAULT '',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `slug` (`slug`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `goadmin_role_users` (
  `role_id` int(11) NOT NULL,
  `user_id` int(11) NOT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX (`role_id`),
  INDEX (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `goadmin_permissions` (
  `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(50) NOT NULL DEFAULT '',
  `slug` varchar(50) NOT NULL DEFAULT '',
  `http_method` varchar(255) DEFAULT NULL,
  `http_path` text,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `slug` (`slug`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `goadmin_role_permissions` (
  `role_id` int(11) NOT NULL,
  `permission_id` int(11) NOT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX (`role_id`),
  INDEX (`permission_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `goadmin_user_permissions` (
  `user_id` int(11) NOT NULL,
  `permission_id` int(11) NOT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX (`user_id`),
  INDEX (`permission_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `goadmin_menu` (
  `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
  `parent_id` int(11) NOT NULL DEFAULT '0',
  `type` tinyint(4) NOT NULL DEFAULT '0',
  `order` int(11) NOT NULL DEFAULT '0',
  `title` varchar(50) NOT NULL DEFAULT '',
  `icon` varchar(50) NOT NULL DEFAULT '',
  `uri` varchar(3000) NOT NULL DEFAULT '',
  `header` varchar(150) DEFAULT NULL,
  `plugin_name` varchar(150) NOT NULL DEFAULT '',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `goadmin_operation_log` (
  `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` int(11) NOT NULL,
  `path` varchar(255) NOT NULL DEFAULT '',
  `method` varchar(10) NOT NULL DEFAULT '',
  `ip` varchar(15) NOT NULL DEFAULT '',
  `input` text NOT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  INDEX (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `goadmin_site` (
  `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
  `key` varchar(100) NOT NULL DEFAULT '',
  `value` longtext,
  `description` varchar(3000) DEFAULT NULL,
  `state` tinyint(4) NOT NULL DEFAULT '0',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `key` (`key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 插入默认管理员用户 (用户名: admin, 密码: admin)
INSERT IGNORE INTO `goadmin_users` (`id`, `username`, `password`, `name`, `avatar`, `remember_token`) 
VALUES (1, 'admin', '$2a$10$2fqG/7sLF8kAVSR2pDONQOI3rkZUf6YT8dbNdIj4YjfOVCMbas9Ne', 'Administrator', '', '');

-- 插入默认角色
INSERT IGNORE INTO `goadmin_roles` (`id`, `name`, `slug`) VALUES 
(1, 'Administrator', 'administrator');

-- 关联用户和角色
INSERT IGNORE INTO `goadmin_role_users` (`role_id`, `user_id`) VALUES (1, 1);

-- 插入权限
INSERT IGNORE INTO `goadmin_permissions` (`id`, `name`, `slug`, `http_method`, `http_path`) VALUES 
(1, 'All permission', '*', '', '*');

-- 关联角色和权限
INSERT IGNORE INTO `goadmin_role_permissions` (`role_id`, `permission_id`) VALUES (1, 1);

-- 插入菜单
INSERT IGNORE INTO `goadmin_menu` (`id`, `parent_id`, `type`, `order`, `title`, `icon`, `uri`, `header`, `plugin_name`) VALUES
(1, 0, 1, 2, 'Admin', 'fa-tasks', '', '', ''),
(2, 1, 1, 2, 'Users', 'fa-users', '/info/users', '', ''),
(3, 1, 1, 3, 'Videos', 'fa-file-video-o', '/info/videos', '', ''),
(4, 1, 1, 4, 'Comments', 'fa-comments', '/info/comments', '', ''),
(5, 1, 1, 5, 'Video Sources', 'fa-video-camera', '/info/video_sources', '', '');

-- 插入站点配置
INSERT IGNORE INTO `goadmin_site` (`key`, `value`, `description`, `state`) VALUES
('site_title', '视频平台管理后台', '网站标题', 1),
('site_logo', '', '网站Logo', 1),
('site_mini_logo', 'VP', '网站小Logo', 1),
('theme', 'adminlte', '主题', 1),
('animation_type', '', '动画类型', 1),
('custom_head_html', '', '自定义头部HTML', 1),
('custom_foot_html', '', '自定义底部HTML', 1),
('footer_info', 'Powered by GoAdmin', '底部信息', 1),
('login_title', '视频平台管理后台', '登录页标题', 1),
('login_logo', '', '登录页Logo', 1),
('auth_user_table', 'goadmin_users', '用户表名', 1);


USE video_platform_mvp;
SHOW TABLES LIKE 'goadmin%';

DROP TABLE IF EXISTS goadmin_users;
DROP TABLE IF EXISTS goadmin_menu;
DROP TABLE IF EXISTS goadmin_roles;
DROP TABLE IF EXISTS goadmin_permissions;
DROP TABLE IF EXISTS goadmin_role_users;
DROP TABLE IF EXISTS goadmin_role_menu;
DROP TABLE IF EXISTS goadmin_role_permissions;
DROP TABLE IF EXISTS goadmin_user_permissions;
DROP TABLE IF EXISTS goadmin_session;
DROP TABLE IF EXISTS goadmin_operation_log;
DROP TABLE IF EXISTS goadmin_site 
-- 把 SHOW TABLES LIKE 'goadmin%'; 列出的所有表都用 DROP TABLE 命令删掉。
-- 使用 IF EXISTS 可以防止表不存在时报错。