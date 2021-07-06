package unit

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// See: http://en.wikipedia.org/wiki/Binary_prefix
const (
	// Binary
	KiB = 1024
	MiB = 1024 * KiB
	GiB = 1024 * MiB
	TiB = 1024 * GiB
	PiB = 1024 * TiB
)

type unitMap map[string]int64

var (
	binaryMap = unitMap{"k": KiB, "m": MiB, "g": GiB, "t": TiB, "p": PiB}
	sizeRegex = regexp.MustCompile(`^(\d+(\.\d+)*) ?([kKmMgGtTpP])?[iI]?[bB]?$`)
)

var binaryAbbrs = []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB", "ZiB", "YiB"}

func getSizeAndUnit(size float64, base float64, _map []string) (float64, string) {
	i := 0
	unitsLimit := len(_map) - 1
	for size >= base && i < unitsLimit {
		size = size / base
		i++
	}
	return size, _map[i]
}

// CustomSize returns a human-readable approximation of a size
// using custom format.
func CustomSize(format string, size float64, base float64, _map []string) string {
	size, unit := getSizeAndUnit(size, base, _map)
	return fmt.Sprintf(format, size, unit)
}

// BytesSize returns a human-readable size in bytes, kibibytes,
// mebibytes, gibibytes, or tebibytes (eg. "44kiB", "17MiB").
func BytesSize(size float64) string {
	return CustomSize("%.4g%s", size, 1024.0, binaryAbbrs)
}

// RAMInBytes parses a human-readable string representing an amount of RAM
// in bytes, kibibytes, mebibytes, gibibytes, or tebibytes and
// returns the number of bytes, or -1 if the string is unparseable.
// Units are case-insensitive, and the 'b' suffix is optional.
func RAMInBytes(size string) (int64, error) {
	return parseSize(size, binaryMap)
}

// Parses the human-readable size string into the amount it represents.
func parseSize(sizeStr string, uMap unitMap) (int64, error) {
	matches := sizeRegex.FindStringSubmatch(sizeStr)
	if len(matches) != 4 {
		return -1, fmt.Errorf("invalid size: '%s'", sizeStr)
	}

	size, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return -1, err
	}

	unitPrefix := strings.ToLower(matches[3])
	if mul, ok := uMap[unitPrefix]; ok {
		size *= float64(mul)
	}

	return int64(size), nil
}

type Unit string

func (u Unit) Bytes() (int64, error) {
	if u == "" {
		return 0, nil
	}
	return RAMInBytes(string(u))
}
