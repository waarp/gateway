package gui

import "embed"

//go:embed front-end/html/*.html
//go:embed front-end/css/*.css
//go:embed front-end/javaScript/*.js
//go:embed front-end/assets/*.png
//go:embed front-end/multi_language/*.json

var webFS embed.FS
