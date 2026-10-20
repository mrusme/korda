//go:build !tzf

package metadata

const GPSZones = false

func gpsZone(float64, float64) (string, bool) {
	return "", false
}
