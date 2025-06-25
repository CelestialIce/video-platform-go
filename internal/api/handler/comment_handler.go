package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/cjh/video-platform-go/internal/service"
	"github.com/gin-gonic/gin"
)

// ---------- 请求 / 响应 DTO ----------

// CreateCommentRequest 创建评论 / 弹幕请求体
type CreateCommentRequest struct {
	Content  string `json:"content"  binding:"required" example:"Great video!"`
	Timeline *uint  `json:"timeline" example:"15"` // 可选弹幕时间点（秒）
}

// CommentInfo 评论信息（用于列表和单条返回）
type CommentInfo struct {
	ID        uint64    `json:"id"         example:"1"`
	Content   string    `json:"content"    example:"Great video!"`
	Timeline  *uint     `json:"timeline,omitempty" example:"15"`
	CreatedAt time.Time `json:"created_at" example:"2025-06-25T11:34:00Z"`
	User      struct {
		ID       uint64 `json:"id"       example:"2"`
		Nickname string `json:"nickname" example:"Tom"`
	} `json:"user"`
}

// ---------- 处理器 ----------

// CreateComment godoc
// @Summary      创建评论 / 弹幕
// @Description  需要登录。根据视频 ID 创建评论或弹幕
// @Tags         评论
// @Security     ApiKeyAuth
// @Accept       json
// @Produce      json
// @Param        id    path      int64                true  "视频 ID"
// @Param        body  body      CreateCommentRequest true  "评论内容"
// @Success      201   {object}  CommentInfo
// @Failure      400   {object}  ErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Failure      500   {object}  ErrorResponse
// @Router       /videos/{id}/comments [post]
func CreateComment(c *gin.Context) {
	videoID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid video ID"})
		return
	}

	var req CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	userIDVal, _ := c.Get("user_id")
	userID := uint64(userIDVal.(float64))

	comment, err := service.CreateCommentService(userID, videoID, req.Content, req.Timeline)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	resp := CommentInfo{
		ID:        comment.ID,
		Content:   comment.Content,
		Timeline:  comment.Timeline,
		CreatedAt: comment.CreatedAt,
	}
	resp.User.ID = comment.User.ID
	resp.User.Nickname = comment.User.Nickname

	c.JSON(http.StatusCreated, resp)
}

// ListComments godoc
// @Summary      获取评论列表
// @Description  根据视频 ID 获取评论 / 弹幕列表
// @Tags         评论
// @Produce      json
// @Param        id  path      int64  true  "视频 ID"
// @Success      200 {array}   CommentInfo
// @Failure      400 {object}  ErrorResponse
// @Failure      500 {object}  ErrorResponse
// @Router       /videos/{id}/comments [get]
func ListComments(c *gin.Context) {
	videoID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid video ID"})
		return
	}

	comments, err := service.ListCommentsService(videoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	var resp []CommentInfo
	for _, comment := range comments {
		info := CommentInfo{
			ID:        comment.ID,
			Content:   comment.Content,
			Timeline:  comment.Timeline,
			CreatedAt: comment.CreatedAt,
		}
		info.User.ID = comment.User.ID
		info.User.Nickname = comment.User.Nickname
		resp = append(resp, info)
	}

	c.JSON(http.StatusOK, resp)
}
