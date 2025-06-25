package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/cjh/video-platform-go/internal/service"
)

// ---------- 请求 / 响应 DTO ----------

// RegisterRequest 注册请求体
type RegisterRequest struct {
	Nickname string `json:"nickname" binding:"required"    example:"Tom"`
	Email    string `json:"email"    binding:"required,email" example:"tom@example.com"`
	Password string `json:"password" binding:"required,min=6" example:"secret123"`
}

// RegisterResponse 注册成功响应
type RegisterResponse struct {
	Message string `json:"message" example:"User registered successfully"`
	UserID  uint64 `json:"user_id" example:"1"`
}

// LoginRequest 登录请求体
type LoginRequest struct {
	Email    string `json:"email"    binding:"required,email" example:"tom@example.com"`
	Password string `json:"password" binding:"required"       example:"secret123"`
}

// LoginResponse 登录成功响应
type LoginResponse struct {
	Message string `json:"message" example:"Login successful"`
	Token   string `json:"token"   example:"<jwt>"`
}

// ---------- 处理器 ----------

// Register godoc
// @Summary      用户注册
// @Description  根据用户提供的昵称、邮箱和密码进行注册
// @Tags         用户
// @Accept       json
// @Produce      json
// @Param        body  body      RegisterRequest   true  "注册请求体"
// @Success      201   {object}  RegisterResponse
// @Failure      400   {object}  ErrorResponse
// @Failure      409   {object}  ErrorResponse
// @Router       /users/register [post]
func Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	user, err := service.Register(req.Nickname, req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusConflict, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, RegisterResponse{
		Message: "User registered successfully",
		UserID:  user.ID,
	})
}

// Login godoc
// @Summary      用户登录
// @Description  根据邮箱和密码进行登录，成功后返回 JWT Token
// @Tags         用户
// @Accept       json
// @Produce      json
// @Param        body  body      LoginRequest   true  "登录请求体"
// @Success      200   {object}  LoginResponse
// @Failure      400   {object}  ErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Router       /users/login [post]
func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	token, err := service.Login(req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		Message: "Login successful",
		Token:   token,
	})
}
