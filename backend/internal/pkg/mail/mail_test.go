package mail

import (
	"strings"
	"testing"
)

func TestRenderVmailbox(t *testing.T) {
	out := RenderVmailbox([]Account{{Address: "bob@example.com", Domain: "example.com"}})
	if !strings.Contains(out, "bob@example.com example.com/bob/Maildir/") {
		t.Errorf("vmailbox = %q", out)
	}
}

func TestRenderDovecotUsers(t *testing.T) {
	out := RenderDovecotUsers([]Account{{Address: "bob@example.com", Domain: "example.com", PasswordHash: "{SHA512-CRYPT}$6$abc"}})
	if !strings.Contains(out, "bob@example.com:{SHA512-CRYPT}$6$abc::::") {
		t.Errorf("dovecot users = %q", out)
	}
}

func TestRenderVdomains(t *testing.T) {
	out := RenderVdomains([]string{"b.com", "a.com"})
	// sorted
	if !strings.Contains(out, "a.com\nb.com") {
		t.Errorf("vdomains not sorted: %q", out)
	}
}
