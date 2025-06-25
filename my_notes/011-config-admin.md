# 011: 配置 GoAdmin 后台管理系统

本文档记录了为 `video-platform-go` 项目集成并配置 GoAdmin 后台管理系统的完整步骤，并重点说明了在此过程中遇到的关键问题及其解决方案。

## 1. 准备工作

### 1.1 环境要求
*   **Go 语言版本**: `1.11` 或更高版本。本项目已在 `go1.23` 环境下验证。
*   **数据库**: 本项目使用 MySQL 8.0，并通过 Docker Compose 进行管理。

### 1.2 安装命令行工具 `adm`
GoAdmin 提供了一个强大的命令行工具 `adm` 来帮助快速生成项目。如果你的环境中没有安装，请使用以下命令安装：

```bash
go install github.com/GoAdminGroup/adm@latest
```
安装成功后，`adm` 命令将被添加到你的 `$GOPATH/bin` 目录下。请确保此目录已在你的系统环境变量 `PATH` 中。

## 2. 初始化数据库

GoAdmin 框架需要一系列特定的数据表来存储菜单、权限、角色、用户等信息。这些表的结构定义在 `admin.sql` 文件中。

由于我们的 MySQL 实例运行在 Docker 容器中，我们需要使用 `docker exec` 命令将 `admin.sql` 导入到名为 `video-platform-mvp` 的数据库中。

在项目根目录下执行以下命令：

```bash
# 读取 admin.sql 文件内容，并通过管道传送给 Docker 容器内的 mysql 客户端执行
cat admin.sql | docker exec -i video-mysql mysql -u root -p'your_strong_password' video_platform_mvp
```

**注意：**
*   请将 `your_strong_password` 替换为你在 `docker-compose.yml` 文件中为 `MYSQL_ROOT_PASSWORD` 设置的真实密码。
*   `video-mysql` 是 `docker-compose.yml` 中定义的容器名称。
*   `video_platform_mvp` 是 `docker-compose.yml` 中定义的数据库名称。

## 3. 生成 GoAdmin 项目（已完成）

我们已经使用 `adm` 工具在 `cmd/admin/` 目录下生成了后台管理项目的基本骨架。这一步是记录性的，用于说明项目的由来。通常的命令如下：

```bash
# 切换到目标目录
cd cmd/admin/

# 执行初始化命令
adm init
```
这会在 `cmd/admin` 目录下生成包括 `main.go`, `config.yml`, `models`, `pages` 等在内的完整 GoAdmin 项目结构。

## 4. 关键配置与问题修复

在初始生成项目后，我们遇到了两个关键问题。这里的配置是确保后台能成功运行的核心。

### 4.1 数据库连接配置

**问题描述：**
启动服务时程序崩溃（panic），并报出 `Access denied for user 'root'@'...'` 错误。

**根本原因：**
`cmd/admin/config.yml` 文件中配置的数据库连接信息（主要是密码和数据库名）与 `docker-compose.yml` 中 MySQL 实例的实际信息不匹配。

**解决方案：**
编辑 `cmd/admin/config.yml` 文件，确保 `database` 部分的配置正确。

```yaml
# 文件路径: ./cmd/admin/config.yml

database:
  # 默认的数据库列表
  default:
    host: 127.0.0.1             # 主机上运行Go程序, 连接本地映射的端口
    port: '3306'                # MySQL 端口
    user: root
    pwd: 'your_strong_password' # <--- 关键：必须与 docker-compose.yml 中的密码一致
    name: 'video_platform_mvp'  # <--- 关键：必须与 docker-compose.yml 中的数据库名一致
    max_idle_con: 50
    max_open_con: 150
    driver: mysql
```

### 4.2 Go 模块导入路径修复

**问题描述：**
编译时报错，提示无法找到包，如 `cannot find package "admin/models"`。

**根本原因：**
Go 语言要求项目内部的包导入使用基于 `go.mod` 文件中定义的模块路径的**绝对路径**，而不是相对路径。

**项目模块路径（来自 `go.mod`）：**
```
module github.com/cjh/video-platform-go
```

**解决方案：**
编辑 `cmd/admin/main.go` 文件，将所有项目内部的包导入路径修改为完整的绝对路径。

```go
// 文件路径: ./cmd/admin/main.go

// ...
import (
    // ... 其他第三方库导入

    // --- 错误的相对路径导入 (需要修改) ---
    // "admin/models"
    // "admin/pages"
    // "admin/tables"

    // --- 正确的绝对路径导入 ---
    "github.com/cjh/video-platform-go/cmd/admin/models"
    "github.com/cjh/video-platform-go/cmd/admin/pages"
    "github.com/cjh/video-platform-go/cmd/admin/tables"
)
// ...
```

## 5. 启动与访问

完成以上所有配置后，即可启动后台管理服务。

1.  **启动服务**
    在 `cmd/admin` 目录下，通常有一个 `Makefile` 文件。使用 `make` 命令启动服务：

    ```bash
    cd cmd/admin
    make serve
    ```
    或者直接使用 `go run`：
    ```bash
    cd cmd/admin
    go run .
    ```

2.  **访问后台**
    *   **登录地址**：[http://127.0.0.1:9033/admin/login](http://127.0.0.1:9033/admin/login)
        *(注意：端口号请以 `cmd/admin/config.yml` 中的 `port` 设置为准)*
    *   **默认账号**：`admin`
    *   **默认密码**：`admin`

至此，GoAdmin 后台管理系统已成功配置并运行。