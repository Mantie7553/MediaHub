package migrations

import (
	"io/fs"
	"strconv"
	"strings"
	"testing"
)

// goose refuses to start when two migrations share a version, which takes the whole server down.
// Branches that each add the next number pass on their own, so this catches the merge
func TestMigrationVersionsAreUnique(t *testing.T) {
	files, err := fs.Glob(FS, "*.sql")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no migrations embedded")
	}
	seen := map[int64]string{}
	for _, name := range files {
		prefix, _, ok := strings.Cut(name, "_")
		if !ok {
			t.Errorf("%s: name must start with a version number and an underscore", name)
			continue
		}
		version, err := strconv.ParseInt(prefix, 10, 64)
		if err != nil {
			t.Errorf("%s: %q is not a version number", name, prefix)
			continue
		}
		if other, dup := seen[version]; dup {
			t.Errorf("version %d is used by both %s and %s, give the newer one the next free number", version, other, name)
		}
		seen[version] = name
	}
}
