//go:build windows

package platform

import "os"

const appName = "FoundryCodexBridge"

// AppDataDir is deliberately based on LOCALAPPDATA rather than the roaming
// profile. Provider mappings and the opencodex installation are device-local.
func AppDataDir() string {
	if value := os.Getenv("LOCALAPPDATA"); value != "" {
		return value + "\\" + appName
	}
	return appName
}
