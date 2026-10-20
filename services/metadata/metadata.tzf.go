//go:build tzf

package metadata

import (
	"sync"
	"time"
	_ "time/tzdata"

	"github.com/ringsaturn/tzf"
)

const GPSZones = true

var finder = sync.OnceValues(tzf.NewDefaultFinder)

func gpsZone(lat float64, lon float64) (string, bool) {
	f, err := finder()
	if err != nil {
		return "", false
	}

	name := f.GetTimezoneName(lon, lat)
	if name == "" {
		return "", false
	}
	if _, err = time.LoadLocation(name); err != nil {
		return "", false
	}

	return name, true
}
