package meta

import (
	"fmt"
	"runtime"
)

func MemInfo() string {
	stats := runtime.MemStats{}
	runtime.ReadMemStats(&stats)

	return fmt.Sprintf("%.1fMb", float64(stats.Alloc)/1024.0/1024.0)

}
