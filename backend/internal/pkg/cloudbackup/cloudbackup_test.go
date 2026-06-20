package cloudbackup

import (
	"encoding/hex"
	"testing"
)

// TestDeriveSigningKey checks the AWS SigV4 key derivation against the documented
// test vector (Signature Version 4 — "Examples of how to derive a signing key").
func TestDeriveSigningKey(t *testing.T) {
	key := deriveSigningKey(
		"wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY",
		"20150830", "us-east-1", "iam",
	)
	got := hex.EncodeToString(key)
	want := "c4afb1cc5771d871763a393e44b703571b55cc28424d1a5e86da6ed3c154a4b9"
	if got != want {
		t.Fatalf("signing key mismatch:\n got %s\nwant %s", got, want)
	}
}

func TestUploadUnknownType(t *testing.T) {
	err := Upload(nil, Destination{Type: "ftp"}, "/tmp/x", "x") //nolint:staticcheck
	if err == nil {
		t.Fatal("expected error for unknown destination type")
	}
}
