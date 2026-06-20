// Package osinfo detects the host operating system and its distribution family
// so that distro-specific operations (package management, firewalls) can be
// dispatched correctly across Debian/Ubuntu and RHEL/CentOS/Rocky.
package osinfo

import (
	"bufio"
	"os"
	"strings"
)

// Family identifies a group of related distributions sharing tooling.
type Family string

const (
	FamilyDebian  Family = "debian" // Debian, Ubuntu, ...
	FamilyRHEL    Family = "rhel"   // RHEL, CentOS, Rocky, AlmaLinux, Fedora
	FamilyUnknown Family = "unknown"
)

// Info describes the detected operating system.
type Info struct {
	ID      string `json:"id"`      // e.g. "ubuntu", "rocky"
	IDLike  string `json:"id_like"` // e.g. "debian", "rhel fedora"
	Name    string `json:"name"`    // pretty name
	Version string `json:"version"` // version id
	Family  Family `json:"family"`  // resolved family
}

const osReleasePath = "/etc/os-release"

// Detect reads /etc/os-release and resolves the distribution family.
func Detect() Info {
	return parse(readFile(osReleasePath))
}

func readFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// parse turns the contents of an os-release file into an Info value. It is
// separated from Detect so it can be unit tested without touching the host.
func parse(content string) Info {
	fields := map[string]string{}
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		fields[strings.TrimSpace(key)] = unquote(strings.TrimSpace(value))
	}

	info := Info{
		ID:      fields["ID"],
		IDLike:  fields["ID_LIKE"],
		Name:    fields["PRETTY_NAME"],
		Version: fields["VERSION_ID"],
	}
	info.Family = resolveFamily(info.ID, info.IDLike)
	return info
}

func resolveFamily(id, idLike string) Family {
	candidates := strings.Fields(strings.ToLower(id + " " + idLike))
	for _, c := range candidates {
		switch c {
		case "debian", "ubuntu", "linuxmint", "raspbian", "pop":
			return FamilyDebian
		case "rhel", "centos", "rocky", "almalinux", "fedora", "ol", "oracle":
			return FamilyRHEL
		}
	}
	return FamilyUnknown
}

func unquote(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}
