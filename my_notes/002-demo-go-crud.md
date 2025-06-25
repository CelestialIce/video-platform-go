太棒了！你的开发环境已经完美启动，所有底层服务都在 Docker 中稳定运行。这是非常坚实的第一步。

现在，我们开始 **阶段 1：API 基础与用户管理**。我会为你提供精确到每一个文件和每一行代码的“手把手”指导。请跟着我的步骤操作。

**目标：** 构建一个可以处理用户注册、登录，并使用 JWT 进行接口认证的 Go Web 服务器。

---

### **第 1 步：安装 Go 依赖包**

在你的项目根目录 `~/go/video-platform-go` 下，打开终端，执行以下命令来下载我们本阶段需要的所有 Go 库：

```bash
# Gin 是我们的 Web 框架
go get -u github.com/gin-gonic/gin

# Viper 用于读取配置文件
go get -u github.com/spf13/viper

# GORM 是我们的数据库 ORM，以及它的 MySQL 驱动
go get -u gorm.io/gorm
go get -u gorm.io/driver/mysql

# 用于处理 JWT (JSON Web Tokens)
go get -u github.com/golang-jwt/jwt/v5

# 用于密码哈希加密
go get -u golang.org/x/crypto/bcrypt
```

---

### **第 2 步：创建项目目录和文件**

我们在上一个回复中规划了项目结构，现在我们来实际创建它们。

```bash
# 在 video-platform-go 根目录下执行
# -p 参数会帮助我们创建所有父目录
mkdir -p cmd/api
mkdir -p internal/api/handler
mkdir -p internal/api/middleware
mkdir -p internal/config
mkdir -p internal/dal/model
mkdir -p internal/service
mkdir -p configs

# 创建我们将要编辑的空文件
touch cmd/api/main.go
touch configs/config.yaml
touch internal/config/config.go
touch internal/dal/model/user.go
touch internal/dal/db.go
touch internal/service/user_service.go
touch internal/api/handler/user_handler.go
touch internal/api/middleware/auth.go
```

---

### **第 3 步：配置管理 (Viper)**

我们需要一个地方来存放数据库密码、JWT 密钥等敏感信息。

**3.1. 编写 `configs/config.yaml` 文件**

将以下内容粘贴到 `configs/config.yaml` 文件中。**注意：** `mysql.dsn` 里的密码需要和你 `docker-compose.yml` 文件中设置的 `MYSQL_ROOT_PASSWORD` 一致。

```yaml
# configs/config.yaml
server:
  port: ":8000" # API 服务的监听端口

mysql:
  dsn: "root:your_strong_password@tcp(127.0.0.1:3306)/video_platform_mvp?charset=utf8mb4&parseTime=True&loc=Local"

redis:
  addr: "127.0.0.1:6379"
  password: ""
  db: 0

jwt:
  secret: "your-very-secret-key-change-it" # 换成一个更复杂的密钥
  expire_hours: 72 # Token 有效期（小时）
```

**3.2. 编写 Go 代码来加载配置**

将以下代码粘贴到 `internal/config/config.go` 文件中。

```go
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
```

---

### **第 4 步：数据库模型 (GORM)**

我们将你的 SQL `users` 表定义为一个 Go `struct`。

将以下代码粘贴到 `internal/dal/model/user.go` 文件中。

```go
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
```

---

### **第 5 步：数据库连接 (DAL - Data Access Layer)**

我们来写连接到 MySQL 的代码。

将以下代码粘贴到 `internal/dal/db.go` 文件中。

```go
// internal/dal/db.go
package dal

import (
	"log"
	// -- FIX THIS LINE --
	"github.com/cjh/video-platform-go/internal/config"
	// -------------------
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// ... rest of the file remains the same ...
var DB *gorm.DB

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
```

---

### **第 6 步：业务逻辑 (Service Layer)**

这是处理核心业务逻辑的地方，例如密码如何加密，JWT 如何生成。

将以下代码粘贴到 `internal/service/user_service.go` 文件中。

```go
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
func Login(email, password string) (string, error) {
	var user model.User
	if err := dal.DB.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", errors.New("invalid credentials")
		}
		return "", err
	}

	// 比较哈希密码和输入的密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(password)); err != nil {
		return "", errors.New("invalid credentials")
	}

	// 生成 JWT Token
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"role":    user.Role,
		"exp":     time.Now().Add(time.Hour * time.Duration(config.AppConfig.JWT.ExpireHours)).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(config.AppConfig.JWT.Secret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
```

---

### **第 7 步：接口处理 (API Handler)**

Handler 负责解析 HTTP 请求，调用 Service，然后返回 HTTP 响应。

将以下代码粘贴到 `internal/api/handler/user_handler.go` 文件中。

```go
// internal/api/handler/user_handler.go
package handler

import (
	"net/http"
	// -- FIX THIS LINE --
	"github.com/cjh/video-platform-go/internal/service"
	// -------------------
	"github.com/gin-gonic/gin"
)
// ... rest of the file remains the same ...
type RegisterRequest struct {
	Nickname string `json:"nickname" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

func Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := service.Register(req.Nickname, req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "User registered successfully",
		"user_id": user.ID,
	})
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := service.Login(req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"token":   token,
	})
}
```

---

### **第 8 步：JWT 中间件**

中间件用于保护需要登录才能访问的接口。

将以下代码粘贴到 `internal/api/middleware/auth.go` 文件中。

```go
// internal/api/middleware/auth.go
package middleware

import (
	"errors"
	"net/http"
	"strings"
	// -- FIX THIS LINE --
	"github.com/cjh/video-platform-go/internal/config"
	// -------------------
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)
// ... rest of the file remains the same ...

func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header format must be Bearer {token}"})
			return
		}

		tokenString := parts[1]

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(config.AppConfig.JWT.Secret), nil
		})

		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token: " + err.Error()})
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			// 将用户信息存入 context，方便后续 handler 使用
			c.Set("user_id", claims["user_id"])
			c.Set("role", claims["role"])
		} else {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			return
		}

		c.Next()
	}
}
```

---

### **第 9 步：组装一切 (main.go)**

最后，我们把所有东西在 `main.go` 中串联起来，启动我们的 API 服务器。

将以下代码粘贴到 `cmd/api/main.go` 文件中。

```go
// cmd/api/main.go
package main


import (
	"log"
	"net/http"
	// -- FIX THESE LINES --
	"github.com/cjh/video-platform-go/internal/api/handler"
	"github.com/cjh/video-platform-go/internal/api/middleware"
	"github.com/cjh/video-platform-go/internal/config"
	"github.com/cjh/video-platform-go/internal/dal"
	// ---------------------
	"github.com/gin-gonic/gin"
)


func main() {
	// 1. 初始化配置
	config.Init()
	log.Println("Configuration loaded")

	// 2. 初始化数据库
	dal.InitMySQL(&config.AppConfig)
	log.Println("Database initialized")

	// 3. 设置 Gin 引擎
	r := gin.Default()

	// 4. 设置路由
	// API 版本分组
	apiV1 := r.Group("/api/v1")
	{
		// 用户路由 (无需认证)
		userRoutes := apiV1.Group("/users")
		{
			userRoutes.POST("/register", handler.Register)
			userRoutes.POST("/login", handler.Login)
		}

		// 需要认证的路由组
		authed := apiV1.Group("/")
		authed.Use(middleware.JWTAuthMiddleware()) // 应用 JWT 中间件
		{
			// 测试路由，用于验证 token
			authed.GET("/me", func(c *gin.Context) {
				userID, _ := c.Get("user_id")
				role, _ := c.Get("role")
				c.JSON(http.StatusOK, gin.H{
					"message": "Token is valid",
					"user_id": userID,
					"role":    role,
				})
			})
		}
	}

	// 5. 启动服务器
	log.Printf("Starting server on port %s", config.AppConfig.Server.Port)
	if err := r.Run(config.AppConfig.Server.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
```

---

### **第 10 步：运行与测试**

你已经完成了阶段 1 的所有编码工作！现在来验证一下成果。

**1. 启动 API 服务器**
在项目根目录 `~/go/video-platform-go` 下，运行：

```bash
go run ./cmd/api/main.go
```

你应该会看到日志输出，表明配置加载成功、数据库连接成功，并且服务器正在监听 8000 端口。

**2. 使用 `curl` 或 Postman 进行测试**

打开一个新的终端。

**测试注册 (POST /api/v1/users/register):**
```bash
curl -X POST http://localhost:8000/api/v1/users/register \
-H "Content-Type: application/json" \
-d '{
    "nickname": "testuser",
    "email": "test@example.com",
    "password": "password123"
}'
```
你应该会收到 `201 Created` 的响应。

**测试登录 (POST /api/v1/users/login):**
```bash
curl -X POST http://localhost:8000/api/v1/users/login \
-H "Content-Type: application/json" \
-d '{
    "email": "test@example.com",
    "password": "password123"
}'
```
你应该会收到 `200 OK` 的响应，其中包含一个 JWT token。**复制这个 token**，我们下一步要用。

**测试受保护的路由 (GET /api/v1/me):**
将上一步复制的 `YOUR_JWT_TOKEN` 替换到下面的命令中。

```bash
curl -X GET http://localhost:8000/api/v1/me \
-H "Authorization: Bearer YOUR_JWT_TOKEN"
```
如果你收到了包含 `user_id` 和 `role` 的 `200 OK` 响应，**恭喜你！** 你已经成功完成了阶段 1！你的后端服务现在拥有了完整的用户认证和授权基础。

随时可以进入 **阶段 2：视频上传与转码**。

好的，原来如此。非常抱歉，我之前的指导让你陷入了这个非常经典的 Go 语言模块（Go Modules）的“陷阱”里。你遇到的问题100%是 **模块路径解析** 的问题，而不是你的代码或者环境有问题。

我来用中文为你彻底讲清楚这个**原理**，你马上就会明白，并且以后再也不会被它困扰。

### **核心原理：Go 编译器如何“找代码”？**

想象一下，你的 `go.mod` 文件就是这个项目的“**身份证**”，上面写着项目的“**全名**”。

当你在代码里写 `import "xxx"` 时，Go 编译器需要决定去哪里找 `xxx` 这个包。它主要在两个地方找：

1.  **标准库 (Standard Library)**: 像 `fmt`, `log`, `net/http` 这些 Go 自带的包。
2.  **第三方或你自己的包**:
    *   **去哪里找？** 它会先看你的 `go.mod` 文件。
    *   **如何找？**
        *   如果 `import "github.com/gin-gonic/gin"`，编译器一看就知道，"哦，这不是我的项目名，我需要去网上（通过代理）把它下载到我的本地缓存里（就是你看到的 `GOPATH/pkg/mod` 目录）"。
        *   如果 `import "video-platform-go/internal/dal"`，编译器会想：“`video-platform-go` 是谁？” 它会拿这个名字和你 `go.mod` 里声明的模块名（`module video-platform-go`）做对比。

**问题的根源就在这里：**

你的模块名叫 `video-platform-go`，而你的导入路径也以 `video-platform-go/` 开头。这本身是正确的，但因为你的模块名**太简单了，看起来不像一个网络路径**（比如 `github.com/xxx/yyy`），Go 的工具链在某些情况下会产生“误判”。

它错误地认为 `video-platform-go` 是一个需要从**网上下载**的第三方库，于是它跑去**全局缓存目录** (`/home/cjh/go/pkg/mod/...`) 里去找你的 `internal` 文件夹，结果当然是找不到！它没有意识到，`video-platform-go` 就是**当前你所在的这个项目**。

### **解决方案：给你的项目一个“不会混淆”的全名**

我们要做的就是给你的项目起一个规范的、不会引起误会的名字。Go 社区的最佳实践是使用类似代码仓库的路径作为模块名，即使你暂时不打算上传它。

这能 100% 解决你的问题。

---

### **手把手修复步骤**

#### **第一步：修改 `go.mod` 文件**

打开你项目根目录下的 `go.mod` 文件。把第一行：

```
module video-platform-go
```

修改为：

```
module github.com/cjh/video-platform-go
```

**说明：**
*   这里的 `github.com` 只是一个惯例，你也可以用 `gitee.com` 或者其他。
*   这里的 `cjh` 是你的用户名，你可以换成任何你喜欢的名字。
*   **你完全不需要真的在 GitHub 上创建这个仓库**，这只是给项目一个唯一的、不会混淆的“全名”。

#### **第二步：全局替换项目内的 `import` 路径**

现在你项目的“全名”变了，所有引用到内部包的地方也必须跟着改。

这是最关键的一步。你需要把所有文件里，`import` 语句中 `video-platform-go/` 的前缀，全部替换成新的模块名 `github.com/cjh/video-platform-go/`。

**你需要修改以下所有文件：**

1.  `cmd/api/main.go`
2.  `internal/dal/db.go`
3.  `internal/service/user_service.go`
4.  `internal/api/handler/user_handler.go`
5.  `internal/api/middleware/auth.go`

**快速操作：**
如果你使用 VS Code，可以按 `Ctrl + Shift + F` (或者 `Cmd + Shift + F` 在 Mac 上) 进行全局搜索和替换：
*   **Find (查找):** `video-platform-go/`
*   **Replace (替换):** `github.com/cjh/video-platform-go/`
*   然后点击“全部替换”按钮。

**例如，`cmd/api/main.go` 的 import 部分会变成这样：**
```go
import (
	"log"
	"net/http"
	"github.com/cjh/video-platform-go/internal/api/handler"
	"github.com/cjh/video-platform-go/internal/api/middleware"
	"github.com/cjh/video-platform-go/internal/config"
	"github.com/cjh/video-platform-go/internal/dal"
	"github.com/gin-gonic/gin"
)
```
其他文件依此类推。

#### **第三步：清理并再次运行**

1.  **回到你的终端**，在项目根目录下，先运行 `go mod tidy`。这个命令会根据你修改后的 `go.mod` 和 `import` 路径，重新整理依赖关系。

    ```bash
    go mod tidy
    ```

2.  **最后，再次运行你的程序！**

    ```bash
    go run ./cmd/api/main.go
    ```

**相信我，这一次一定能成功！** 你会看到熟悉的启动日志。

因为现在，当编译器看到 `import "github.com/cjh/video-platform-go/internal/dal"`，它会和 `go.mod` 里的 `module github.com/cjh/video-platform-go` 一对比，立刻就明白了：“哦！这就是当前项目内部的包！我应该直接去当前目录下的 `internal/dal` 文件夹里找它。” 误会就此解除。