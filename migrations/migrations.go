package migrations

import "embed"

const Path string = "."

//go:embed *.sql
var FS embed.FS
