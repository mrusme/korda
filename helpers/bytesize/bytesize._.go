package bytesize

import (
	"math/big"
	"strconv"
)

const (
	KiB int64 = 1024
	MiB       = 1024 * KiB
	GiB       = 1024 * MiB
)

var units = []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}

func HumanReadable(bytes int64) string {
	magnitude := 0
	remainder := float64(bytes)
	for remainder >= 1024 && magnitude+1 < len(units) {
		magnitude++
		remainder /= 1024
	}

	precision := 1
	if magnitude == 0 {
		precision = 0
	}

	return ToFixed(remainder, precision) + " " + units[magnitude]
}

func ToFixed(x float64, digits int) string {
	scale := new(big.Float).SetPrec(256).SetInt64(1)
	ten := new(big.Float).SetPrec(256).SetInt64(10)
	for range digits {
		scale.Mul(scale, ten)
	}

	scaled := new(big.Float).SetPrec(256).SetFloat64(x)
	scaled.Mul(scaled, scale)
	scaled.Add(scaled, big.NewFloat(0.5).SetPrec(256))

	n, _ := scaled.Int(nil)
	result := new(big.Float).SetPrec(256).SetInt(n)
	result.Quo(result, scale)

	return result.Text('f', digits)
}

func Percentage(part int64, total int64) float64 {
	if total == 0 {
		return 0
	}

	fixed := ToFixed(float64(part)/float64(total)*100, 2)
	value, _ := strconv.ParseFloat(fixed, 64)

	return value
}
