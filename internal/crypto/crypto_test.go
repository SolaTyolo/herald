package crypto_test

import (
	"testing"

	"github.com/SolaTyolo/herald/internal/crypto"
)

func TestAPIKeyHashAndCompare(t *testing.T) {
	hash, err := crypto.HashAPIKey("gn_test_secret_key_12345")
	if err != nil {
		t.Fatal(err)
	}
	if !crypto.CompareAPIKey("gn_test_secret_key_12345", hash) {
		t.Fatal("compare should succeed")
	}
	if crypto.CompareAPIKey("wrong", hash) {
		t.Fatal("compare should fail")
	}
}
