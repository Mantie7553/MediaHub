// Package migrations embeds the goose SQL migrations so the server can apply them on startup
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
