//go:build windows

package metadata

import (
	"os"
	"syscall"
	"time"
)

func birthTime(path string) time.Time {
	st, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}

	attrs, ok := st.Sys().(*syscall.Win32FileAttributeData)
	if !ok {
		return time.Time{}
	}

	return time.Unix(0, attrs.CreationTime.Nanoseconds())
}
