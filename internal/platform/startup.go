package platform

import (
	"strconv"
	"strings"
)

const startupValueName = "Logy"

// startupCommand builds the Run registry / launch command for logy start.
// When home is set, LOGY_HOME is exported via cmd.exe so the daemon uses that data dir.
func startupCommand(exe, home string) string {
	exe = strings.TrimSpace(exe)
	quotedExe := strconv.Quote(exe)
	home = strings.TrimSpace(home)
	if home == "" {
		return quotedExe + " start"
	}
	return "cmd.exe /c \"set LOGY_HOME=" + home + "&& " + quotedExe + " start\""
}
