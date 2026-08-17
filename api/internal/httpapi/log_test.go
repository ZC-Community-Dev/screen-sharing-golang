package httpapi

import "testing"

func TestRedactHidesSecrets(t *testing.T) {
	if redact("presenterToken=abc") != "[redacted]" {
		t.Fatal("token must be redacted")
	}
	if redact("LINK_ID_SALT") != "[redacted]" {
		t.Fatal("salt must be redacted")
	}
	if redact("/api/v1/links") != "/api/v1/links" {
		t.Fatal("safe path changed")
	}
}
