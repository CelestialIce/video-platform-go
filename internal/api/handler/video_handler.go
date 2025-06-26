package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/cjh/video-platform-go/internal/service"
	"github.com/gin-gonic/gin"
)

// ---------- 请求 / 响应 DTO ----------

// InitiateUploadRequest 初始化上传请求体
// after
type InitiateUploadRequest struct {
    FileName    string `json:"file_name"    binding:"required" example:"movie.mp4"`
    Title       string `json:"title"        binding:"required" example:"My Movie"`
    Description string `json:"description"                     example:"A funny video"`
}

// InitiateUploadResponse 初始化上传成功响应
type InitiateUploadResponse struct {
	UploadURL string `json:"upload_url" example:"https://minio.local/presigned-url"`
	VideoID   uint64 `json:"video_id"   example:"123"`
}

// CompleteUploadRequest 完成上传请求体
type CompleteUploadRequest struct {
	VideoID uint64 `json:"video_id" binding:"required" example:"123"`
}

// MessageResponse 通用消息响应
type MessageResponse struct {
	Message string `json:"message" example:"Transcoding task has been submitted"`
}

// VideoInfo 与 model.Video 字段一一对应（仅保留需要给前端看的字段）
type VideoInfo struct {
	ID          uint64    `json:"id"          example:"123"`
	Title       string    `json:"title"       example:"My Holiday"`
	Description string    `json:"description" example:"A short description"`
	CoverURL    string    `json:"cover_url"   example:"https://example.com/cover.jpg"`
	Status      string    `json:"status"      example:"online"`
	Duration    uint      `json:"duration"    example:"3600"`
	CreatedAt   time.Time `json:"created_at"  example:"2025-06-20T09:00:00Z"`
}

// ListVideosResponse 视频列表响应
type ListVideosResponse struct {
	Videos []VideoInfo `json:"videos"`
	Total  int64       `json:"total"  example:"100"`
}

// VideoDetailsResponse 视频详情响应
type VideoDetailsResponse struct {
	Video   any `json:"video"`
	Sources any `json:"sources"`
}

// ---------- 处理器 ----------

// InitiateUpload godoc
// @Summary      初始化视频上传
// @Description  生成预签名上传 URL
// @Tags         视频
// @Security     ApiKeyAuth
// @Accept       json
// @Produce      json
// @Param 		 body  body		 InitiateUploadRequest true "上传元数据"
// @Success      200   {object}  InitiateUploadResponse
// @Failure      400   {object}  ErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Failure      500   {object}  ErrorResponse
// @Router       /videos/upload/initiate [post]
func InitiateUpload(c *gin.Context) {
    var req InitiateUploadRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
        return
    }

    userIDVal, _ := c.Get("user_id")
    userID, ok := userIDVal.(float64)
    if !ok {
        c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Invalid user ID in token"})
        return
    }

    // *** 传 4 个参数 ***
    url, video, err := service.InitiateUploadService(
        uint64(userID),
        req.FileName,
        req.Title,
        req.Description,
    )
    if err != nil {
        c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
        return
    }

    c.JSON(http.StatusOK, InitiateUploadResponse{
        UploadURL: url,
        VideoID:   video.ID,
    })
}

// CompleteUpload godoc
// @Summary      完成视频上传
// @Description  通知后端开始转码
// @Tags         视频
// @Security     ApiKeyAuth
// @Accept       json
// @Produce      json
// @Param        body  body      CompleteUploadRequest  true  "视频 ID"
// @Success      200   {object}  MessageResponse
// @Failure      400   {object}  ErrorResponse
// @Failure      500   {object}  ErrorResponse
// @Router       /videos/upload/complete [post]
func CompleteUpload(c *gin.Context) {
	var req CompleteUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid video ID"})
		return
	}

	if err := service.CompleteUploadService(req.VideoID); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, MessageResponse{Message: "Transcoding task has been submitted"})
}

// ListVideos godoc
// @Summary      获取视频列表
// @Tags         视频
// @Produce      json
// @Param        limit   query     int  false  "每页数量"  default(10)
// @Param        offset  query     int  false  "偏移量"   default(0)
// @Success      200     {object}  ListVideosResponse
// @Failure      500     {object}  ErrorResponse
// @Router       /videos [get]
func ListVideos(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	videos, total, err := service.ListVideosService(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	// 转换 []model.Video -> []VideoInfo
	respVideos := make([]VideoInfo, 0, len(videos))
	for _, v := range videos {
		respVideos = append(respVideos, VideoInfo{
			ID:          v.ID,
			Title:       v.Title,
			Description: v.Description,
			CoverURL:    v.CoverURL,
			Status:      v.Status,
			Duration:    v.Duration,
			CreatedAt:   v.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, ListVideosResponse{
		Videos: respVideos,
		Total:  total,
	})
}

// GetVideoDetails godoc
// @Summary      获取视频详情
// @Tags         视频
// @Produce      json
// @Param        id   path      int64  true  "视频 ID"
// @Success      200  {object}  VideoDetailsResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Router       /videos/{id} [get]
func GetVideoDetails(c *gin.Context) {
	videoID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid video ID"})
		return
	}

	video, sources, err := service.GetVideoDetailsService(videoID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, VideoDetailsResponse{
		Video:   *video,
		Sources: sources,
	})
}
