//go:build !windows

package libraries

import "golang.org/x/sys/unix"

func readable(p string) bool {
	return unix.Access(p, unix.R_OK) == nil
}
