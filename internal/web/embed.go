package web

import "embed"

//go:embed template/*.html
var TemplatesFS embed.FS
