// Package php discovers installed PHP-FPM versions and resolves their FastCGI
// sockets so websites can be wired to a specific PHP version.
package php

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/jud3ns0hn/digitarlopanel/backend/internal/pkg/osinfo"
)

// Version describes an installed PHP-FPM version.
type Version struct {
	Version string `json:"version"` // e.g. "8.3"
	Socket  string `json:"socket"`  // FastCGI unix socket path
	Active  bool   `json:"active"`  // socket currently present
}

var versionDir = regexp.MustCompile(`^\d+\.\d+$`)

// DebianSocketPath returns the conventional php-fpm socket for a version on the
// Debian family.
func DebianSocketPath(version string) string {
	return "/run/php/php" + version + "-fpm.sock"
}

// PackageName returns the php-fpm package name for a version and distro family.
func PackageName(family osinfo.Family, version string) string {
	if family == osinfo.FamilyDebian {
		return "php" + version + "-fpm"
	}
	// RHEL/Remi naming, e.g. php83-php-fpm for 8.3.
	return "php" + strip(version) + "-php-fpm"
}

// strip removes the dot from a version, e.g. "8.3" -> "83".
func strip(v string) string {
	out := make([]rune, 0, len(v))
	for _, r := range v {
		if r != '.' {
			out = append(out, r)
		}
	}
	return string(out)
}

// ListVersions discovers installed PHP versions. On the Debian family it reads
// /etc/php/<version>; on the RHEL family it reports the system php-fpm if present.
func ListVersions(family osinfo.Family) []Version {
	switch family {
	case osinfo.FamilyDebian:
		return listDebian("/etc/php")
	default:
		return listRHEL()
	}
}

func listDebian(root string) []Version {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var versions []Version
	for _, e := range entries {
		if !e.IsDir() || !versionDir.MatchString(e.Name()) {
			continue
		}
		// Only count versions that have an fpm configuration.
		if _, err := os.Stat(filepath.Join(root, e.Name(), "fpm")); err != nil {
			continue
		}
		socket := DebianSocketPath(e.Name())
		versions = append(versions, Version{
			Version: e.Name(),
			Socket:  socket,
			Active:  fileExists(socket),
		})
	}
	sort.Slice(versions, func(i, j int) bool { return versions[i].Version < versions[j].Version })
	return versions
}

func listRHEL() []Version {
	// Common default socket locations for the system php-fpm.
	for _, socket := range []string{"/run/php-fpm/www.sock", "/var/run/php-fpm/www.sock"} {
		if fileExists(socket) {
			return []Version{{Version: "system", Socket: socket, Active: true}}
		}
	}
	if _, err := os.Stat("/etc/php-fpm.d"); err == nil {
		return []Version{{Version: "system", Socket: "/run/php-fpm/www.sock", Active: false}}
	}
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
