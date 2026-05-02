package config

import (
	"runtime/debug"
)

var (
	AppVersion = ""
)

func init() {
	discoveredAppVersion := "v(devel)"

	bi, ok := debug.ReadBuildInfo()
	if !ok {
		goto done
	}

	discoveredAppVersion = bi.Main.Version

done:
	if AppVersion == "" {
		AppVersion = discoveredAppVersion
	}
}
