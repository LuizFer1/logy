package daemon

import (
	"io"
	"log"
	"os"
	"path/filepath"
)

const maxLogBytes = 1 << 20

// NewRotatingLogger writes daemon logs to path, rotating once the file exceeds 1 MiB.
func NewRotatingLogger(path string) (*log.Logger, io.Closer, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, nil, err
	}
	if info, err := os.Stat(path); err == nil && info.Size() >= maxLogBytes {
		_ = os.Rename(path, path+".old")
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return nil, nil, err
	}
	return log.New(file, "logy: ", log.LstdFlags|log.LUTC), file, nil
}
