package api

import (
	"errors"
	"path/filepath"
	"strings"
)

// errOutsideRoot is returned when a requested path escapes the file root.
var errOutsideRoot = errors.New("path is outside the permitted root")

// resolvePath cleans a user-supplied path and ensures it stays within root.
// root of "/" permits the whole filesystem. It returns the absolute, cleaned
// path safe to use for filesystem operations.
func resolvePath(root, p string) (string, error) {
	if root == "" {
		root = "/"
	}
	root = filepath.Clean(root)

	if p == "" {
		p = root
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(root, p)
	}
	clean := filepath.Clean(p)

	if root == "/" {
		return clean, nil
	}
	if clean == root || strings.HasPrefix(clean, root+string(filepath.Separator)) {
		return clean, nil
	}
	return "", errOutsideRoot
}
