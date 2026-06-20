package dns

import (
	"strings"
	"testing"
)

func TestAdminToRname(t *testing.T) {
	if got := adminToRname("admin@example.com", "example.com"); got != "admin.example.com." {
		t.Errorf("adminToRname = %q", got)
	}
	if got := adminToRname("", "example.com"); got != "hostmaster.example.com." {
		t.Errorf("default rname = %q", got)
	}
}

func TestRenderZone(t *testing.T) {
	out, err := Render(Zone{
		Domain: "example.com",
		Serial: 2026010101,
		Records: []Record{
			{Name: "@", Type: "A", Value: "1.2.3.4", TTL: 3600},
			{Name: "mail", Type: "MX", Value: "mail.example.com.", TTL: 3600, Priority: 10},
			{Name: "www", Type: "CNAME", Value: "example.com.", TTL: 3600},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"IN SOA ns1.example.com. hostmaster.example.com.",
		"@ 3600 IN A 1.2.3.4",
		"mail 3600 IN MX 10 mail.example.com.",
		"www 3600 IN CNAME example.com.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("zone missing %q in:\n%s", want, out)
		}
	}
}
