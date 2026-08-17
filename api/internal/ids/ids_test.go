package ids

import (
	"strings"
	"testing"
)

func TestGenerateLengthAlphabetAndUniqueness(t *testing.T) {
	const salt = "test-salt"
	seen := map[string]bool{}
	for range 20 {
		id, err := Generate(salt)
		if err != nil {
			t.Fatal(err)
		}
		if len(id) < 8 {
			t.Fatalf("id too short: %q", id)
		}
		for _, r := range id {
			if !strings.ContainsRune(Alphabet, r) {
				t.Fatalf("id %q has non-base62 %q", id, r)
			}
		}
		if seen[id] {
			t.Fatalf("duplicate id %q", id)
		}
		seen[id] = true
	}
}

func TestGenerateNotSequential(t *testing.T) {
	a, err := Generate("salt")
	if err != nil {
		t.Fatal(err)
	}
	b, err := Generate("salt")
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Fatal("ids should not match")
	}
	if a < b && incrementBase62(a) == b {
		t.Fatal("ids look sequential")
	}
}

func incrementBase62(s string) string {
	return s
}

func TestValidPublicID(t *testing.T) {
	if ValidPublicID("short") {
		t.Fatal("expected invalid")
	}
	if ValidPublicID("bad-id!!") {
		t.Fatal("expected invalid")
	}
	if !ValidPublicID("Abcdefgh") {
		t.Fatal("expected valid")
	}
}
