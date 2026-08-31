package update

import (
	"fmt"
	"strings"
)

// AssetName returns the release archive name for version/GOOS/GOARCH.
// Windows uses .zip; darwin/linux use .tar.gz. Only amd64 and arm64 are supported.
func AssetName(version, goos, goarch string) (string, error) {
	v := NormalizeVersion(version)
	switch goos {
	case "windows", "darwin", "linux":
	default:
		return "", fmt.Errorf("unsupported GOOS %q", goos)
	}
	switch goarch {
	case "amd64", "arm64":
	default:
		return "", fmt.Errorf("unsupported GOARCH %q", goarch)
	}
	ext := "tar.gz"
	if goos == "windows" {
		ext = "zip"
	}
	return fmt.Sprintf("logy_%s_%s_%s.%s", v, goos, goarch, ext), nil
}

// ParseChecksums parses a checksums.txt body into filename → hex digest.
// Lines are "digest  filename" (two or more spaces/tabs between fields).
func ParseChecksums(text string) (map[string]string, error) {
	out := make(map[string]string)
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return nil, fmt.Errorf("invalid checksum line: %q", line)
		}
		digest, name := fields[0], fields[len(fields)-1]
		out[name] = digest
	}
	return out, nil
}
