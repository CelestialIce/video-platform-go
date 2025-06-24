// cmd/admin/main.go
package main

import (
	"log"

	"github.com/GoAdminGroup/go-admin/context"
	"github.com/GoAdminGroup/go-admin/engine"
	"github.com/GoAdminGroup/go-admin/modules/config"
	"github.com/GoAdminGroup/go-admin/modules/language"
	"github.com/GoAdminGroup/go-admin/template"
	"github.com/GoAdminGroup/go-admin/template/chartjs"

	"github.com/GoAdminGroup/themes/adminlte"
	"github.com/gin-gonic/gin"

	// --- FIX: 导入所有缺失的包 ---
	"github.com/GoAdminGroup/go-admin/modules/db"
	"github.com/GoAdminGroup/go-admin/plugins/admin/modules/table"
	"github.com/GoAdminGroup/go-admin/template/types"
	"github.com/GoAdminGroup/go-admin/template/types/form"

	// ---------------------------

	app_config "github.com/cjh/video-platform-go/internal/config"
)

func main() {
	r := gin.Default()

	// 1. 初始化我们自己的配置
	app_config.Init()
	log.Println("Configuration loaded for admin panel")

	// 2. 创建 Go-Admin 引擎
	eng := engine.Default()

	// 3. 配置 Go-Admin 数据库连接
	cfg := config.Config{
		Databases: config.DatabaseList{
			"default": {
				Host:         app_config.AppConfig.MySQL.Host,
				Port:         app_config.AppConfig.MySQL.Port,
				User:         app_config.AppConfig.MySQL.User,
				Pwd:          app_config.AppConfig.MySQL.Password,
				Name:         app_config.AppConfig.MySQL.Database,
				MaxIdleConns: 50,
				MaxOpenConns: 150,
				Driver:       db.DriverMysql,
			},
		},
		UrlPrefix: "/admin",
		Store: config.Store{
			Path:   "./uploads",
			Prefix: "uploads",
		},
		Language:    language.CN,
		IndexUrl:    "/",
		LoginUrl:    "/login",
		Debug:       true,
		ColorScheme: adminlte.ColorschemeSkinBlue,
		Title:       "视频平台管理后台",
		Logo:        "视频平台",
		MiniLogo:    "VP",
	}

	// 设置模板引擎
	template.AddComp(chartjs.NewChart())

	// 4. 初始化引擎
	if err := eng.AddConfig(&cfg).
		AddGenerators(map[string]table.Generator{
			"users":         GetUserTable,
			"videos":        GetVideoTable,
			"comments":      GetCommentTable,
			"video_sources": GetVideoSourceTable,
		}).
		Use(r); err != nil {
		panic(err)
	}

	// 5. 启动服务
	log.Println("Admin panel server is running on http://127.0.0.1:8081/admin")
	r.Run(":8081")
}

// GetUserTable 用户表格生成器
func GetUserTable(ctx *context.Context) table.Table {
	userTable := table.NewDefaultTable(ctx, table.DefaultConfig())

	info := userTable.GetInfo().HideFilterArea()
	info.AddField("ID", "id", db.Bigint).FieldFilterable()
	info.AddField("昵称", "nickname", db.Varchar).FieldFilterable()
	info.AddField("邮箱", "email", db.Varchar).FieldFilterable()
	info.AddField("角色", "role", db.Varchar).FieldFilterable()
	info.AddField("创建时间", "created_at", db.Datetime).FieldSortable()
	info.AddField("更新时间", "updated_at", db.Datetime)

	info.SetTable("users").SetTitle("用户管理").SetDescription("用户管理")

	formList := userTable.GetForm()
	formList.AddField("ID", "id", db.Bigint, form.Default).FieldNotAllowEdit().FieldNotAllowAdd()
	formList.AddField("昵称", "nickname", db.Varchar, form.Text).FieldMust()
	formList.AddField("邮箱", "email", db.Varchar, form.Email).FieldMust()
	formList.AddField("密码", "hashed_password", db.Varchar, form.Password).FieldNotAllowEdit()
	formList.AddField("角色", "role", db.Varchar, form.SelectSingle).
		FieldOptions(types.FieldOptions{
			{Text: "用户", Value: "user"},
			{Text: "管理员", Value: "admin"},
			{Text: "审核员", Value: "auditor"},
		}).FieldDefault("user")

	formList.SetTable("users").SetTitle("用户管理").SetDescription("用户管理")

	return userTable
}

// GetVideoTable 视频表格生成器
func GetVideoTable(ctx *context.Context) table.Table {
	videoTable := table.NewDefaultTable(ctx, table.DefaultConfig())

	info := videoTable.GetInfo().HideFilterArea()
	info.AddField("ID", "id", db.Bigint).FieldFilterable()
	info.AddField("用户ID", "user_id", db.Bigint).FieldFilterable()
	info.AddField("标题", "title", db.Varchar).FieldFilterable()
	info.AddField("描述", "description", db.Text)
	info.AddField("状态", "status", db.Varchar).FieldFilterable()
	info.AddField("时长(秒)", "duration", db.Int)
	info.AddField("封面地址", "cover_url", db.Varchar)
	info.AddField("创建时间", "created_at", db.Datetime).FieldSortable()
	info.AddField("更新时间", "updated_at", db.Datetime)

	info.SetTable("videos").SetTitle("视频管理").SetDescription("视频管理")

	formList := videoTable.GetForm()
	formList.AddField("ID", "id", db.Bigint, form.Default).FieldNotAllowEdit().FieldNotAllowAdd()
	formList.AddField("用户ID", "user_id", db.Bigint, form.Number).FieldMust()
	formList.AddField("标题", "title", db.Varchar, form.Text).FieldMust()
	formList.AddField("描述", "description", db.Text, form.TextArea)
	formList.AddField("状态", "status", db.Varchar, form.SelectSingle).
		FieldOptions(types.FieldOptions{
			{Text: "上传中", Value: "uploading"},
			{Text: "转码中", Value: "transcoding"},
			{Text: "在线", Value: "online"},
			{Text: "失败", Value: "failed"},
			{Text: "私有", Value: "private"},
		}).FieldDefault("uploading")
	formList.AddField("时长(秒)", "duration", db.Int, form.Number)
	formList.AddField("封面地址", "cover_url", db.Varchar, form.Text)

	formList.SetTable("videos").SetTitle("视频管理").SetDescription("视频管理")

	return videoTable
}

// GetCommentTable 评论表格生成器
func GetCommentTable(ctx *context.Context) table.Table {
	commentTable := table.NewDefaultTable(ctx, table.DefaultConfig())

	info := commentTable.GetInfo().HideFilterArea()
	info.AddField("ID", "id", db.Bigint).FieldFilterable()
	info.AddField("视频ID", "video_id", db.Bigint).FieldFilterable()
	info.AddField("用户ID", "user_id", db.Bigint).FieldFilterable()
	info.AddField("内容", "content", db.Text)
	info.AddField("时间轴(秒)", "timeline", db.Int)
	info.AddField("创建时间", "created_at", db.Datetime).FieldSortable()

	info.SetTable("comments").SetTitle("评论管理").SetDescription("评论管理")

	formList := commentTable.GetForm()
	formList.AddField("ID", "id", db.Bigint, form.Default).FieldNotAllowEdit().FieldNotAllowAdd()
	formList.AddField("视频ID", "video_id", db.Bigint, form.Number).FieldMust()
	formList.AddField("用户ID", "user_id", db.Bigint, form.Number).FieldMust()
	formList.AddField("内容", "content", db.Text, form.TextArea).FieldMust()
	formList.AddField("时间轴(秒)", "timeline", db.Int, form.Number)

	formList.SetTable("comments").SetTitle("评论管理").SetDescription("评论管理")

	return commentTable
}

// GetVideoSourceTable 视频源表格生成器
func GetVideoSourceTable(ctx *context.Context) table.Table {
	sourceTable := table.NewDefaultTable(ctx, table.DefaultConfig())

	info := sourceTable.GetInfo().HideFilterArea()
	info.AddField("ID", "id", db.Bigint).FieldFilterable()
	info.AddField("视频ID", "video_id", db.Bigint).FieldFilterable()
	info.AddField("质量", "quality", db.Varchar).FieldFilterable()
	info.AddField("格式", "format", db.Varchar).FieldFilterable()
	info.AddField("播放地址", "url", db.Varchar)
	info.AddField("文件大小", "file_size", db.Bigint)
	info.AddField("创建时间", "created_at", db.Datetime).FieldSortable()

	info.SetTable("video_sources").SetTitle("视频源管理").SetDescription("视频源管理")

	formList := sourceTable.GetForm()
	formList.AddField("ID", "id", db.Bigint, form.Default).FieldNotAllowEdit().FieldNotAllowAdd()
	formList.AddField("视频ID", "video_id", db.Bigint, form.Number).FieldMust()
	formList.AddField("质量", "quality", db.Varchar, form.SelectSingle).
		FieldOptions(types.FieldOptions{
			{Text: "360p", Value: "360p"},
			{Text: "720p", Value: "720p"},
			{Text: "1080p", Value: "1080p"},
		})
	formList.AddField("格式", "format", db.Varchar, form.SelectSingle).
		FieldOptions(types.FieldOptions{
			{Text: "HLS", Value: "HLS"},
			{Text: "DASH", Value: "DASH"},
			{Text: "MP4", Value: "MP4"},
		})
	formList.AddField("播放地址", "url", db.Varchar, form.Text).FieldMust()
	formList.AddField("文件大小", "file_size", db.Bigint, form.Number)

	formList.SetTable("video_sources").SetTitle("视频源管理").SetDescription("视频源管理")

	return sourceTable
}
