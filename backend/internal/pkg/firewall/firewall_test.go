package firewall

import "testing"

func TestSplitPortProto(t *testing.T) {
	cases := []struct {
		in    string
		port  int
		proto string
		ok    bool
	}{
		{"80/tcp", 80, "tcp", true},
		{"53/udp", 53, "udp", true},
		{"80", 0, "", false},
		{"abc/tcp", 0, "", false},
		{"80/sctp", 0, "", false},
	}
	for _, tc := range cases {
		port, proto, ok := splitPortProto(tc.in)
		if ok != tc.ok || port != tc.port || proto != tc.proto {
			t.Errorf("splitPortProto(%q) = (%d,%q,%v), want (%d,%q,%v)",
				tc.in, port, proto, ok, tc.port, tc.proto, tc.ok)
		}
	}
}

func TestValidate(t *testing.T) {
	if err := validate(0, "tcp"); err == nil {
		t.Error("expected error for port 0")
	}
	if err := validate(70000, "tcp"); err == nil {
		t.Error("expected error for port > 65535")
	}
	if err := validate(80, "icmp"); err == nil {
		t.Error("expected error for invalid proto")
	}
	if err := validate(443, "tcp"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
