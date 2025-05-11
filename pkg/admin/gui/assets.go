package gui

import "embed"

//go:embed front_end/html/*.html
//go:embed front_end/css/*.css
var webFS embed.FS
