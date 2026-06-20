// Package dns generates BIND zone files and wires them into the named
// configuration across the Debian (bind9) and RHEL (named) families.
package dns

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/jud3ns0hn/digitarlopanel/backend/internal/pkg/osinfo"
)

// Record is a single resource record.
type Record struct {
	Name     string // "@" or label
	Type     string // A, AAAA, CNAME, MX, TXT, NS
	Value    string
	TTL      int
	Priority int
}

// Zone is the input to a zone file.
type Zone struct {
	Domain  string
	NS      string
	Admin   string
	Serial  uint32
	Records []Record
}

// paths returns the zone directory, managed include file, main config file and
// systemd unit for the distribution family.
func paths(family osinfo.Family) (zoneDir, managedConf, mainConf, unit string) {
	if family == osinfo.FamilyDebian {
		return "/etc/bind/zones", "/etc/bind/named.conf.digitarlopanel", "/etc/bind/named.conf.local", "bind9"
	}
	return "/var/named", "/etc/named.digitarlopanel.conf", "/etc/named.conf", "named"
}

// ReloadUnit returns the systemd unit to reload after a change.
func ReloadUnit(family osinfo.Family) string {
	_, _, _, unit := paths(family)
	return unit
}

const zoneTemplate = `$TTL {{ .DefaultTTL }}
@ IN SOA {{ .NSFQDN }} {{ .AdminFQDN }} (
    {{ .Serial }} ; serial
    3600       ; refresh
    1800       ; retry
    604800     ; expire
    86400 )    ; minimum
@ IN NS {{ .NSFQDN }}
{{- range .Records }}
{{ .Name }} {{ .TTL }} IN {{ .Type }} {{ if eq .Type "MX" }}{{ .Priority }} {{ end }}{{ .Value }}
{{- end }}
`

var tmpl = template.Must(template.New("zone").Parse(zoneTemplate))

func fqdn(s string) string {
	if strings.HasSuffix(s, ".") {
		return s
	}
	return s + "."
}

// adminToRname converts an email like admin@example.com to the SOA rname
// admin.example.com.
func adminToRname(admin, domain string) string {
	if admin == "" {
		admin = "hostmaster@" + domain
	}
	at := strings.Index(admin, "@")
	if at < 0 {
		return fqdn(admin)
	}
	return fqdn(admin[:at] + "." + admin[at+1:])
}

// Render produces the zone file text for a zone.
func Render(z Zone) (string, error) {
	ns := z.NS
	if ns == "" {
		ns = "ns1." + z.Domain
	}
	data := struct {
		Zone
		DefaultTTL int
		NSFQDN     string
		AdminFQDN  string
	}{
		Zone:       z,
		DefaultTTL: 3600,
		NSFQDN:     fqdn(ns),
		AdminFQDN:  adminToRname(z.Admin, z.Domain),
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// Sync writes every zone file, regenerates the managed include with all zone
// blocks and ensures the main config includes it. It is a no-op when BIND is
// not installed (the configuration base directory is absent).
func Sync(family osinfo.Family, zones []Zone) error {
	zoneDir, managedConf, mainConf, _ := paths(family)
	if !dirExists(filepath.Dir(zoneDir)) {
		return nil // BIND not installed; skip silently
	}
	if err := os.MkdirAll(zoneDir, 0o755); err != nil {
		return err
	}

	var blocks bytes.Buffer
	blocks.WriteString("# Managed by DigitarloPanel - do not edit by hand\n")
	for _, z := range zones {
		content, err := Render(z)
		if err != nil {
			return err
		}
		zoneFile := filepath.Join(zoneDir, "db."+z.Domain)
		if err := os.WriteFile(zoneFile, []byte(content), 0o644); err != nil {
			return err
		}
		fmt.Fprintf(&blocks, "zone \"%s\" {\n    type master;\n    file \"%s\";\n};\n", z.Domain, zoneFile)
	}
	if err := os.WriteFile(managedConf, blocks.Bytes(), 0o644); err != nil {
		return err
	}
	return ensureInclude(mainConf, managedConf)
}

// ensureInclude appends an include directive to mainConf if not already present.
func ensureInclude(mainConf, managedConf string) error {
	include := fmt.Sprintf("include \"%s\";", managedConf)
	data, err := os.ReadFile(mainConf)
	if err != nil {
		if os.IsNotExist(err) {
			// named not installed yet; skip silently so the panel stays usable.
			return nil
		}
		return err
	}
	if strings.Contains(string(data), include) {
		return nil
	}
	f, err := os.OpenFile(mainConf, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString("\n" + include + "\n")
	return err
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
