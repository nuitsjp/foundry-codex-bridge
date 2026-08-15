//go:build !windows

package platform

import (
	"os"
	"path/filepath"
)

const appName = "FoundryCodexBridge"

func AppDataDir() string {
	if value, err := os.UserConfigDir(); err == nil && value != "" {
		return filepath.Join(value, appName)
	}
	return appName
}
