package gui

import "embed"

//go:embed front_end/html/*.html
//go:embed front_end/css/*.css
//go:embed front_end/assets/*.png
var webFS embed.FS
