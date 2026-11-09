package analytics

import "github.com/shirou/gopsutil/v4/mem"

// TotalMemory is the memory of the machine, in bytes, and whether it could
// be read.
func TotalMemory() (uint64, bool) {
	stat, err := mem.VirtualMemory()
	if err != nil {
		return 0, false
	}

	return stat.Total, true
}
