//go:build windows

package mediaroot

import "golang.org/x/sys/windows"

func diskUsage(path string) (u Usage, err error) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return u, err
	}

	var available, total, free uint64
	if err = windows.GetDiskFreeSpaceEx(p, &available, &total, &free); err != nil {
		return u, err
	}

	u.Total = int64(total)
	u.Free = int64(free)
	u.Available = int64(available)

	return u, nil
}
