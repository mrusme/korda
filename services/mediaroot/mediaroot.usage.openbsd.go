//go:build openbsd

package mediaroot

import "syscall"

func diskUsage(path string) (u Usage, err error) {
	var st syscall.Statfs_t
	if err = syscall.Statfs(path, &st); err != nil {
		return u, err
	}

	size := int64(st.F_bsize)
	u.Total = int64(st.F_blocks) * size
	u.Free = int64(st.F_bfree) * size
	u.Available = int64(st.F_bavail) * size

	return u, nil
}
