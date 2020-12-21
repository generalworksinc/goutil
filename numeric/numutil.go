package gw_numeric

import (
	"encoding/json"
	"fmt"
	"strconv"
)

func CNullUInt(num json.Number) uint {

	if i, err := num.Int64(); err == nil {
		return uint(i)
	} else {
		return 0
	}
}

func CNulStrlUInt(num string) uint {

	if i, err := strconv.Atoi(num); err == nil {
		return uint(i)
	} else {
		return 0
	}
}
func CNullStrInt(num string) int {

	if i, err := strconv.Atoi(num); err == nil {
		return i
	} else {
		return 0
	}
}

func zip_string(a, b []string) ([][]string, error) {

	if len(a) != len(b) {
		return nil, fmt.Errorf("zip: arguments must be of same length")
	}

	r := make([][]string, len(a), len(a))

	for i, e := range a {
		r[i] = []string{e, b[i]}
	}

	return r, nil
}
