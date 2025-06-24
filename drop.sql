
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