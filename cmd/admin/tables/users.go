package tables

import (
	"github.com/GoAdminGroup/go-admin/context"
	"github.com/GoAdminGroup/go-admin/modules/db"
	"github.com/GoAdminGroup/go-admin/plugins/admin/modules/table"
	"github.com/GoAdminGroup/go-admin/template/types/form"
)

func GetUsersTable(ctx *context.Context) table.Table {

	users := table.NewDefaultTable(ctx, table.DefaultConfigWithDriver("mysql").SetPrimaryKey("id", db.Bigint))

	info := users.GetInfo().HideFilterArea()

	info.AddField("Created_at", "created_at", db.Timestamp)
	info.AddField("Email", "email", db.Varchar)
	info.AddField("Hashed_password", "hashed_password", db.Varchar)
	info.AddField("Id", "id", db.Bigint).
		FieldFilterable()
	info.AddField("Nickname", "nickname", db.Varchar)
	info.AddField("Role", "role", db.Enum)
	info.AddField("Updated_at", "updated_at", db.Timestamp)

	info.SetTable("users").SetTitle("Users").SetDescription("Users")

	formList := users.GetForm()
	formList.AddField("Created_at", "created_at", db.Timestamp, form.Datetime)
	formList.AddField("Email", "email", db.Varchar, form.Email)
	formList.AddField("Hashed_password", "hashed_password", db.Varchar, form.Text)
	formList.AddField("Id", "id", db.Bigint, form.Default)
	formList.AddField("Nickname", "nickname", db.Varchar, form.Text)
	formList.AddField("Role", "role", db.Enum, form.Text)
	formList.AddField("Updated_at", "updated_at", db.Timestamp, form.Datetime)

	formList.SetTable("users").SetTitle("Users").SetDescription("Users")

	return users
}
