//go:build linux

package metadata

import (
	"time"

	"golang.org/x/sys/unix"
)

func birthTime(path string) time.Time {
	var st unix.Statx_t
	if err := unix.Statx(unix.AT_FDCWD, path, 0, unix.STATX_BTIME, &st); err != nil || st.Mask&unix.STATX_BTIME == 0 {
		return time.Time{}
	}

	return time.Unix(st.Btime.Sec, int64(st.Btime.Nsec))
}
