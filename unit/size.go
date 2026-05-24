package gw_unit

import "fmt"

func FormatFileSize(size int64) string {
	const (
		KB int64 = 1 << (10 * (iota + 1))
		MB
		GB
		TB
		PB
		EB
	)

	switch {
	case size >= EB:
		return fmt.Sprintf("%.2f EB", float64(size)/float64(EB))
	case size >= PB:
		return fmt.Sprintf("%.2f PB", float64(size)/float64(PB))
	case size >= TB:
		return fmt.Sprintf("%.2f TB", float64(size)/float64(TB))
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/float64(GB))
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/float64(MB))
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/float64(KB))
	default:
		return fmt.Sprintf("%d B", size)
	}
}
