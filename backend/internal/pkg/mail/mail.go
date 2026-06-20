// Package mail generates Postfix virtual maps and a Dovecot passwd-file for
// virtual mail domains and mailboxes. It writes the maps only when the
// respective service configuration directories exist, so it is a no-op on hosts
// without a mail server installed.
package mail

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Account is a virtual mailbox with a pre-hashed password (dovecot scheme).
type Account struct {
	Address      string
	Domain       string
	PasswordHash string
}

// File locations (Postfix/Dovecot conventions).
const (
	VmailboxPath = "/etc/postfix/vmailbox"
	VdomainsPath = "/etc/postfix/vdomains"
	DovecotUsers = "/etc/dovecot/users"
	MailRoot     = "/var/mail/vhosts"
)

// RenderVdomains returns the virtual domains file (one domain per line).
func RenderVdomains(domains []string) string {
	sorted := append([]string(nil), domains...)
	sort.Strings(sorted)
	var b strings.Builder
	b.WriteString("# Managed by DigitarloPanel\n")
	for _, d := range sorted {
		b.WriteString(d + "\n")
	}
	return b.String()
}

// RenderVmailbox returns the Postfix virtual_mailbox_maps source mapping each
// address to a Maildir path under MailRoot.
func RenderVmailbox(accounts []Account) string {
	var b strings.Builder
	b.WriteString("# Managed by DigitarloPanel\n")
	for _, a := range accounts {
		local := localPart(a.Address)
		fmt.Fprintf(&b, "%s %s/%s/Maildir/\n", a.Address, a.Domain, local)
	}
	return b.String()
}

// RenderDovecotUsers returns a Dovecot passwd-file with hashed passwords.
func RenderDovecotUsers(accounts []Account) string {
	var b strings.Builder
	b.WriteString("# Managed by DigitarloPanel\n")
	for _, a := range accounts {
		// passwd-file format: user:password:uid:gid::home
		fmt.Fprintf(&b, "%s:%s::::%s/%s/%s\n",
			a.Address, a.PasswordHash, MailRoot, a.Domain, localPart(a.Address))
	}
	return b.String()
}

func localPart(addr string) string {
	if at := strings.Index(addr, "@"); at >= 0 {
		return addr[:at]
	}
	return addr
}

// Sync writes the map files when their parent directories exist. It returns the
// path of the Postfix vmailbox source that must be compiled with postmap (empty
// if Postfix is not present).
func Sync(domains []string, accounts []Account) (postmapTarget string, err error) {
	if dirExists(filepath.Dir(VdomainsPath)) {
		if err := os.WriteFile(VdomainsPath, []byte(RenderVdomains(domains)), 0o644); err != nil {
			return "", err
		}
		if err := os.WriteFile(VmailboxPath, []byte(RenderVmailbox(accounts)), 0o644); err != nil {
			return "", err
		}
		postmapTarget = VmailboxPath
	}
	if dirExists(filepath.Dir(DovecotUsers)) {
		if err := os.WriteFile(DovecotUsers, []byte(RenderDovecotUsers(accounts)), 0o640); err != nil {
			return postmapTarget, err
		}
	}
	return postmapTarget, nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
