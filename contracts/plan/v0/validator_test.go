package v0

import (
	"bytes"
	"strings"
	"testing"
)

const frozenDigest = "sha256:7582512f122d2f2a9c4461facc7541c9887053f137260d6ebe9c6dea611d039d"

func TestEmbeddedContractInfo(t *testing.T) {
	version, digest, err := ContractInfo()
	if err != nil {
		t.Fatal(err)
	}
	if version != "0.0.1" {
		t.Fatalf("contract version = %q, want 0.0.1", version)
	}
	if digest != frozenDigest {
		t.Fatalf("contract digest = %q, want %q", digest, frozenDigest)
	}
}

func TestFixtureConformance(t *testing.T) {
	fixtures, err := Fixtures()
	if err != nil {
		t.Fatal(err)
	}
	if len(fixtures) != 60 {
		t.Fatalf("fixture count = %d, want 60", len(fixtures))
	}
	valid, invalid := 0, 0
	for _, fixture := range fixtures {
		fixture := fixture
		t.Run(fixture.ID, func(t *testing.T) {
			data, err := FixtureData(fixture.Path)
			if err != nil {
				t.Fatal(err)
			}
			result := Validate(data)
			if result.Valid != fixture.Valid {
				t.Fatalf("valid = %t, want %t; findings: %+v", result.Valid, fixture.Valid, result.Findings)
			}
			if result.AuthorityAuthenticated {
				t.Fatal("validator must never claim authority authentication")
			}
			if result.ContractVersion != "0.0.1" || result.ContractDigest != frozenDigest {
				t.Fatalf("result contract identity = %s %s", result.ContractVersion, result.ContractDigest)
			}
		})
		if fixture.Valid {
			valid++
		} else {
			invalid++
		}
	}
	if valid != 21 || invalid != 39 {
		t.Fatalf("fixture counts = %d valid, %d invalid; want 21/39", valid, invalid)
	}
}

func TestRejectDuplicateJSONKeys(t *testing.T) {
	result := Validate([]byte(`{"contractVersion":"0.0.1","contractVersion":"0.0.1"}`))
	if result.Valid || len(result.Findings) == 0 || result.Findings[0].Code != "invalid_json" {
		t.Fatalf("duplicate key result = %+v", result)
	}
}

func TestRejectOversizedInput(t *testing.T) {
	result := Validate(bytes.Repeat([]byte(" "), maxInputBytes+1))
	if result.Valid || len(result.Findings) == 0 || result.Findings[0].Code != "input_too_large" {
		t.Fatalf("oversized result = %+v", result)
	}
}

func TestRejectExcessiveJSONDepth(t *testing.T) {
	data := []byte(strings.Repeat("[", maxJSONDepth+2) + "null" + strings.Repeat("]", maxJSONDepth+2))
	if err := rejectDuplicateKeys(data); err == nil || !strings.Contains(err.Error(), "nesting") {
		t.Fatalf("depth error = %v", err)
	}
}

func TestSafeSourceRef(t *testing.T) {
	cases := []struct {
		name string
		ref  string
		want bool
	}{
		{name: "relative", ref: "docs/plan.md", want: true},
		{name: "logical urn", ref: "urn:example:receipt:1", want: true},
		{name: "public https", ref: "https://example.org/receipt/1", want: true},
		{name: "parent traversal", ref: "../private.md"},
		{name: "absolute path", ref: "/Users/private/plan.md"},
		{name: "file URL", ref: "file:///tmp/plan.md"},
		{name: "credentials", ref: "https://alice:secret@example.org/receipt"},
		{name: "loopback", ref: "https://127.0.0.1/receipt"},
		{name: "private IPv6", ref: "https://[fd00::1]/receipt"},
		{name: "local hostname", ref: "https://service.local/receipt"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := safeSourceRef(tc.ref); got != tc.want {
				t.Fatalf("safeSourceRef(%q) = %t, want %t", tc.ref, got, tc.want)
			}
		})
	}
}
