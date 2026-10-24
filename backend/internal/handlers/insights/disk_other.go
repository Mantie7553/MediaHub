//go:build !linux && !windows

package insights

// free space isn't read on other systems; the page leaves it out
func diskSpace(path string) (total, free int64, ok bool) {
	return 0, 0, false
}
