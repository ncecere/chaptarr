//go:build windows

package service

// Free-space reporting is not implemented on Windows.
func diskUsage(string) (free, total uint64) { return 0, 0 }
