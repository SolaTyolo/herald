package wasm

import "testing"

func TestAllowsNetwork(t *testing.T) {
	perms := []string{"network:https://api.sendgrid.com"}
	if !allowsNetwork(perms, "https://api.sendgrid.com/v3/mail/send") {
		t.Fatal("expected allowed")
	}
	if allowsNetwork(perms, "https://evil.com") {
		t.Fatal("expected denied")
	}
}

func TestValidateManifestPermissions(t *testing.T) {
	if err := validateManifestPermissions([]string{"network:https://x.com"}); err != nil {
		t.Fatal(err)
	}
	if err := validateManifestPermissions([]string{"network:evil.com"}); err == nil {
		t.Fatal("expected error for missing scheme")
	}
}
