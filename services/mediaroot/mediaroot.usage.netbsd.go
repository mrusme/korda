//go:build netbsd

package mediaroot

import "golang.org/x/sys/unix"

func diskUsage(path string) (u Usage, err error) {
	var st unix.Statvfs_t
	if err = unix.Statvfs(path, &st); err != nil {
		return u, err
	}

	size := int64(st.Frsize)
	u.Total = int64(st.Blocks) * size
	u.Free = int64(st.Bfree) * size
	u.Available = int64(st.Bavail) * size

	return u, nil
}
