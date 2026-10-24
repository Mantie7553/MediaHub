package insights

import "syscall"

// the size of the volume path is on and the space left on it for MediaHub
func diskSpace(path string) (total, free int64, ok bool) {
	var st syscall.Statfs_t
	if path == "" || syscall.Statfs(path, &st) != nil {
		return 0, 0, false
	}
	return int64(st.Blocks) * st.Bsize, int64(st.Bavail) * st.Bsize, true
}
