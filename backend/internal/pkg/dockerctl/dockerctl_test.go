package dockerctl

import "testing"

func TestValidName(t *testing.T) {
	good := []string{"nginx", "my-container", "registry.io/app:1.2", "sha256:abcdef", "a_b.c"}
	for _, g := range good {
		if !validName(g) {
			t.Errorf("validName(%q) = false, want true", g)
		}
	}
	bad := []string{"", "bad name", "x;rm -rf", "$(whoami)", "a|b"}
	for _, b := range bad {
		if validName(b) {
			t.Errorf("validName(%q) = true, want false", b)
		}
	}
}
