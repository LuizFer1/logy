package control

import (
	"os"
	"os/user"
	"runtime"
	"strings"
)

// PipeName returns the user-local control endpoint.
// On Windows this is a named pipe; elsewhere it is a Unix domain socket path.
func PipeName() string {
	name := "logy"
	if u, err := user.Current(); err == nil {
		if u.Username != "" {
			name = "logy-" + sanitizeUser(u.Username)
		}
	} else if env := strings.TrimSpace(os.Getenv("USERNAME")); env != "" {
		name = "logy-" + sanitizeUser(env)
	}
	if runtime.GOOS == "windows" {
		return `\\.\pipe\` + name
	}
	return os.TempDir() + "/" + name + ".sock"
}

func sanitizeUser(value string) string {
	value = strings.ReplaceAll(value, `\`, "-")
	value = strings.ReplaceAll(value, "/", "-")
	return value
}
