package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const startupValueName = "Logy"

// startupCommand builds the Run registry command for a hidden logy start.
// Prefer a VBScript launcher so Windows does not flash a console on login.
func startupCommand(exe, home, vbsPath string) string {
	exe = strings.TrimSpace(exe)
	home = strings.TrimSpace(home)
	vbsPath = strings.TrimSpace(vbsPath)
	if vbsPath != "" {
		return fmt.Sprintf("wscript.exe //B //Nologo %s", strconv.Quote(vbsPath))
	}
	quotedExe := strconv.Quote(exe)
	if home == "" {
		return quotedExe + " start"
	}
	return "cmd.exe /c \"set LOGY_HOME=" + home + "&& " + quotedExe + " start\""
}

func writeAutostartVBS(path, exe, home string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	exe = strings.ReplaceAll(exe, `"`, `""`)
	home = strings.ReplaceAll(home, `"`, `""`)
	content := "Set sh = CreateObject(\"WScript.Shell\")\r\n"
	if home != "" {
		content += "sh.Environment(\"Process\")(\"LOGY_HOME\") = \"" + home + "\"\r\n"
	}
	content += "sh.Run \"\"\"" + exe + "\"\"\" start\", 0, False\r\n"
	return os.WriteFile(path, []byte(content), 0644)
}
