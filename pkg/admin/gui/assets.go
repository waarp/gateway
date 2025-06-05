package gui

import "embed"

//go:embed front_end/html/*.html
//go:embed front_end/css/*.css
//go:embed front_end/javaScript/*.js
//go:embed front_end/assets/*.png
//go:embed front_end/multi_language/*.json

var webFS embed.FS
