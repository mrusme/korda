//go:build openbsd

package metadata

import (
	"syscall"
	"time"
)

func birthTime(path string) time.Time {
	var st syscall.Stat_t
	if err := syscall.Stat(path, &st); err != nil || st.X__st_birthtim.Sec <= 0 {
		return time.Time{}
	}

	return time.Unix(st.X__st_birthtim.Unix())
}
