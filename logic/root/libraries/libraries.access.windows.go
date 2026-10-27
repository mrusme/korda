//go:build windows

package libraries

import "os"

func readable(p string) bool {
	f, err := os.Open(p)
	if err != nil {
		return false
	}

	return f.Close() == nil
}
