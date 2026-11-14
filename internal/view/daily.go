package view

import "html/template"

var FreezeFrameDaily = template.Must(template.ParseFS(TemplatesFS, "template/base.html", "template/freeze_frame_daily.html"))
var FreezeFrameAccount = template.Must(template.ParseFS(TemplatesFS, "template/base.html", "template/freeze_frame_account.html"))
var FreezeFrameAdmin = template.Must(template.ParseFS(TemplatesFS, "template/base.html", "template/freeze_frame_admin.html"))
