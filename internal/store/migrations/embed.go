package migrations

import (
	"embed"
	"fmt"
)

//go:embed sql/*.sql
var files embed.FS

func MustLoad(name string) string {
	b, err := files.ReadFile("sql/" + name)
	if err != nil {
		panic(fmt.Errorf("load migration %s: %w", name, err))
	}
	return string(b)
}
