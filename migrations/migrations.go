package migrations

import (
	"embed"
	"io/fs"
)

//go:embed *.sql
var files embed.FS

// Files returns the embedded SQL migration files.
func Files() fs.FS {
	return files
}
