package php

import (
	"os"
	"path/filepath"
)

func mkDir(root, name string) error {
	return os.MkdirAll(filepath.Join(root, name), 0o755)
}

func mkFPM(root, version string) error {
	return os.MkdirAll(filepath.Join(root, version, "fpm"), 0o755)
}
