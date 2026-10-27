//go:build linux || darwin || freebsd

package mediaroot

import "syscall"

func diskUsage(path string) (u Usage, err error) {
	var st syscall.Statfs_t
	if err = syscall.Statfs(path, &st); err != nil {
		return u, err
	}

	size := int64(st.Bsize)
	u.Total = int64(st.Blocks) * size
	u.Free = int64(st.Bfree) * size
	u.Available = int64(st.Bavail) * size

	return u, nil
}
