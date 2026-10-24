package insights

import (
	"syscall"
	"unsafe"
)

var getDiskFreeSpaceEx = syscall.NewLazyDLL("kernel32.dll").NewProc("GetDiskFreeSpaceExW")

// the size of the volume path is on and the space left on it for MediaHub
func diskSpace(path string) (total, free int64, ok bool) {
	if path == "" {
		return 0, 0, false
	}
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, 0, false
	}
	var available, size, totalFree uint64
	r, _, _ := getDiskFreeSpaceEx.Call(
		uintptr(unsafe.Pointer(p)),
		uintptr(unsafe.Pointer(&available)),
		uintptr(unsafe.Pointer(&size)),
		uintptr(unsafe.Pointer(&totalFree)),
	)
	if r == 0 {
		return 0, 0, false
	}
	return int64(size), int64(available), true
}
