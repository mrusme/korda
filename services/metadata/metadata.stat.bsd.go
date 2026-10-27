//go:build darwin || freebsd || netbsd

package metadata

import (
	"syscall"
	"time"
)

func birthTime(path string) time.Time {
	var st syscall.Stat_t
	if err := syscall.Stat(path, &st); err != nil || st.Birthtimespec.Sec <= 0 {
		return time.Time{}
	}

	return time.Unix(st.Birthtimespec.Unix())
}
