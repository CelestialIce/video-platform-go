package tables

import (
	"github.com/GoAdminGroup/go-admin/context"
	"github.com/GoAdminGroup/go-admin/modules/db"
	"github.com/GoAdminGroup/go-admin/plugins/admin/modules/table"
	"github.com/GoAdminGroup/go-admin/template/types/form"
)

func GetVideosTable(ctx *context.Context) table.Table {

	videos := table.NewDefaultTable(ctx, table.DefaultConfigWithDriver("mysql").SetPrimaryKey("id", db.Bigint))

	info := videos.GetInfo().HideFilterArea()

	info.AddField("Cover_url", "cover_url", db.Varchar)
	info.AddField("Created_at", "created_at", db.Timestamp)
	info.AddField("Description", "description", db.Text)
	info.AddField("视频时长，单位秒", "duration", db.Int)
	info.AddField("Id", "id", db.Bigint).
		FieldFilterable()
	info.AddField("Status", "status", db.Enum)
	info.AddField("Title", "title", db.Varchar)
	info.AddField("Updated_at", "updated_at", db.Timestamp)
	info.AddField("User_id", "user_id", db.Bigint)

	info.SetTable("videos").SetTitle("Videos").SetDescription("Videos")

	formList := videos.GetForm()
	formList.AddField("Cover_url", "cover_url", db.Varchar, form.Text)
	formList.AddField("Created_at", "created_at", db.Timestamp, form.Datetime)
	formList.AddField("Description", "description", db.Text, form.RichText)
	formList.AddField("视频时长，单位秒", "duration", db.Int, form.Number)
	formList.AddField("Id", "id", db.Bigint, form.Default)
	formList.AddField("Status", "status", db.Enum, form.Text)
	formList.AddField("Title", "title", db.Varchar, form.Text)
	formList.AddField("Updated_at", "updated_at", db.Timestamp, form.Datetime)
	formList.AddField("User_id", "user_id", db.Bigint, form.Number)

	formList.SetTable("videos").SetTitle("Videos").SetDescription("Videos")

	return videos
}
