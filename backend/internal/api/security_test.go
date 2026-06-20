package api

import "testing"

func TestValidatePasswordStrength(t *testing.T) {
	cases := []struct {
		pw      string
		wantErr bool
	}{
		{"short1", true},            // too short
		{"alllettersonly", true},    // no digit
		{"1234567890", true},        // no letter
		{"goodpass123", false},      // ok
		{"Sup3rSecret!", false},     // ok
	}
	for _, tc := range cases {
		err := validatePasswordStrength(tc.pw)
		if (err != nil) != tc.wantErr {
			t.Errorf("validatePasswordStrength(%q) err=%v, wantErr=%v", tc.pw, err, tc.wantErr)
		}
	}
}

func TestTailLines(t *testing.T) {
	in := "a\nb\nc\nd\ne"
	if got := tailLines(in, 2); got != "d\ne" {
		t.Errorf("tailLines last 2 = %q, want %q", got, "d\ne")
	}
	if got := tailLines(in, 10); got != in {
		t.Errorf("tailLines more than available = %q, want full", got)
	}
}

func TestValidRole(t *testing.T) {
	for _, r := range []string{"admin", "operator", "viewer"} {
		if !validRole(r) {
			t.Errorf("validRole(%q) = false, want true", r)
		}
	}
	if validRole("root") {
		t.Error("validRole(\"root\") = true, want false")
	}
}
