package tables

import (
	"github.com/GoAdminGroup/go-admin/context"
	"github.com/GoAdminGroup/go-admin/modules/db"
	"github.com/GoAdminGroup/go-admin/plugins/admin/modules/table"
	"github.com/GoAdminGroup/go-admin/template/types/form"
)

func GetCommentsTable(ctx *context.Context) table.Table {

	comments := table.NewDefaultTable(ctx, table.DefaultConfigWithDriver("mysql").SetPrimaryKey("id", db.Bigint))

	info := comments.GetInfo().HideFilterArea()

	info.AddField("Content", "content", db.Text)
	info.AddField("Created_at", "created_at", db.Timestamp)
	info.AddField("Id", "id", db.Bigint).
		FieldFilterable()
	info.AddField("弹幕出现时间点，单位秒; 若为普通评论则为Null", "timeline", db.Int)
	info.AddField("User_id", "user_id", db.Bigint)
	info.AddField("Video_id", "video_id", db.Bigint)

	info.SetTable("comments").SetTitle("Comments").SetDescription("Comments")

	formList := comments.GetForm()
	formList.AddField("Content", "content", db.Text, form.RichText)
	formList.AddField("Created_at", "created_at", db.Timestamp, form.Datetime)
	formList.AddField("Id", "id", db.Bigint, form.Default)
	formList.AddField("弹幕出现时间点，单位秒; 若为普通评论则为Null", "timeline", db.Int, form.Number)
	formList.AddField("User_id", "user_id", db.Bigint, form.Number)
	formList.AddField("Video_id", "video_id", db.Bigint, form.Number)

	formList.SetTable("comments").SetTitle("Comments").SetDescription("Comments")

	return comments
}
