package gui

import (
    "html/template"
)

var templates = template.Must(
    template.ParseFS(webFS, "front_end/html/*.html"),
)
