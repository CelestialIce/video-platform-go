package tables

import (
	"github.com/GoAdminGroup/go-admin/context"
	"github.com/GoAdminGroup/go-admin/modules/db"
	"github.com/GoAdminGroup/go-admin/plugins/admin/modules/table"
	"github.com/GoAdminGroup/go-admin/template/types/form"
)

func GetVideosourcesTable(ctx *context.Context) table.Table {

	videoSources := table.NewDefaultTable(ctx, table.DefaultConfigWithDriver("mysql").SetPrimaryKey("id", db.Bigint))

	info := videoSources.GetInfo().HideFilterArea()

	info.AddField("Created_at", "created_at", db.Timestamp)
	info.AddField("文件大小，单位字节", "file_size", db.Bigint)
	info.AddField("例如: Hls, Dash, Mp4", "format", db.Varchar)
	info.AddField("Id", "id", db.Bigint).
		FieldFilterable()
	info.AddField("例如: 360P, 720P, 1080P", "quality", db.Varchar)
	info.AddField("播放地址, M3u8文件或Mp4文件", "url", db.Varchar)
	info.AddField("Video_id", "video_id", db.Bigint)

	info.SetTable("video_sources").SetTitle("Videosources").SetDescription("Videosources")

	formList := videoSources.GetForm()
	formList.AddField("Created_at", "created_at", db.Timestamp, form.Datetime)
	formList.AddField("文件大小，单位字节", "file_size", db.Bigint, form.Number)
	formList.AddField("例如: Hls, Dash, Mp4", "format", db.Varchar, form.Text)
	formList.AddField("Id", "id", db.Bigint, form.Default)
	formList.AddField("例如: 360P, 720P, 1080P", "quality", db.Varchar, form.Text)
	formList.AddField("播放地址, M3u8文件或Mp4文件", "url", db.Varchar, form.Text)
	formList.AddField("Video_id", "video_id", db.Bigint, form.Number)

	formList.SetTable("video_sources").SetTitle("Videosources").SetDescription("Videosources")

	return videoSources
}
