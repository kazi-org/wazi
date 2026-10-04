package v0

import (
	"bytes"
	"encoding/json"
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

func TestIndependentApprovalDoesNotBorrowAuditOrMismatchedProof(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{name: "audit previous attempt", mutate: func(proof map[string]any) {
			proof["auditOnly"] = true
			proof["attemptId"] = "example:plan:A-old"
			proof["planRevision"] = "r0"
			proof["planDigest"] = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
		}},
		{name: "stale attempt", mutate: func(proof map[string]any) {
			proof["attemptId"] = "example:plan:A-old"
		}},
		{name: "stale revision", mutate: func(proof map[string]any) {
			proof["planRevision"] = "r0"
			proof["planDigest"] = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
		}},
		{name: "wrong task", mutate: func(proof map[string]any) {
			proof["taskId"] = "example:plan:T4"
			proof["attemptId"] = "example:plan:A4"
		}},
		{name: "wrong policy", mutate: func(proof map[string]any) {
			proof["policyRevision"] = "other-policy/1"
		}},
		{name: "wrong head and base", mutate: func(proof map[string]any) {
			proof["subject"].(map[string]any)["head"] = "different-head"
			proof["subject"].(map[string]any)["base"] = "different-base"
			proof["independence"] = map[string]any{"mode": "contributors"}
			proof["contributors"] = []any{"example:author"}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bundle := authorityBundle(t)
			proof, currentID, evaluation := splitCurrentIndependence(t, bundle)
			proof["id"] = "example:plan:EVproof"
			tc.mutate(proof)
			prependEvidence(bundle, proof)
			delete(currentProof(bundle, currentID), "independence")
			setEvaluationEvidence(evaluation, str(proof["id"]), currentID)

			result := Validate(marshalBundle(t, bundle))
			if result.Valid || !hasFinding(result, "independence_missing") {
				t.Fatalf("mismatched proof rescued current review: valid=%t findings=%+v", result.Valid, result.Findings)
			}
		})
	}
}

func TestIndependentApprovalSelectsAnyQualifyingCurrentProof(t *testing.T) {
	t.Run("audit proof first, valid current proof after", func(t *testing.T) {
		bundle := authorityBundle(t)
		proof, currentID, evaluation := splitCurrentIndependence(t, bundle)
		proof["id"] = "example:plan:EVaudit"
		proof["auditOnly"] = true
		proof["attemptId"] = "example:plan:A-old"
		proof["planRevision"] = "r0"
		proof["planDigest"] = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
		prependEvidence(bundle, proof)
		setEvaluationEvidence(evaluation, str(proof["id"]), currentID)

		result := Validate(marshalBundle(t, bundle))
		if !result.Valid {
			t.Fatalf("valid current proof after audit evidence failed: %+v", result.Findings)
		}
	})

	t.Run("earlier invalid current proof, later valid proof", func(t *testing.T) {
		bundle := authorityBundle(t)
		proof, currentID, evaluation := splitCurrentIndependence(t, bundle)
		proof["id"] = "example:plan:EVsingular"
		proof["independence"] = map[string]any{"mode": "singular"}
		prependEvidence(bundle, proof)
		setEvaluationEvidence(evaluation, str(proof["id"]), currentID)

		result := Validate(marshalBundle(t, bundle))
		if !result.Valid {
			t.Fatalf("valid current proof after invalid proof failed: %+v", result.Findings)
		}
	})
}

func authorityBundle(t *testing.T) map[string]any {
	t.Helper()
	data, err := FixtureData("fixtures/valid/authority-attested-independence.json")
	if err != nil {
		t.Fatal(err)
	}
	var bundle map[string]any
	if err := json.Unmarshal(data, &bundle); err != nil {
		t.Fatal(err)
	}
	return bundle
}

func splitCurrentIndependence(t *testing.T, bundle map[string]any) (map[string]any, string, map[string]any) {
	t.Helper()
	var evaluation map[string]any
	for _, value := range array(get(bundle, "evaluations")) {
		candidate := object(value)
		if str(get(candidate, "requirementId")) == "example:plan:R3" {
			evaluation = candidate
			break
		}
	}
	if evaluation == nil {
		t.Fatal("independent-approval evaluation missing from fixture")
	}
	currentID := "example:plan:EV3"
	var proof map[string]any
	for _, value := range array(get(bundle, "evidence")) {
		candidate := object(value)
		if str(get(candidate, "id")) == currentID {
			encoded, err := json.Marshal(candidate)
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(encoded, &proof); err != nil {
				t.Fatal(err)
			}
			break
		}
	}
	if proof == nil {
		t.Fatal("current review evidence missing from fixture")
	}
	return proof, currentID, evaluation
}

func currentProof(bundle map[string]any, id string) map[string]any {
	for _, value := range array(get(bundle, "evidence")) {
		if str(get(value, "id")) == id {
			return object(value)
		}
	}
	return nil
}

func prependEvidence(bundle map[string]any, evidence map[string]any) {
	all := array(get(bundle, "evidence"))
	bundle["evidence"] = append([]any{evidence}, all...)
}

func setEvaluationEvidence(evaluation map[string]any, proofID, currentID string) {
	evaluation["evidenceIds"] = []any{proofID, currentID}
}

func marshalBundle(t *testing.T, bundle map[string]any) []byte {
	t.Helper()
	data, err := json.Marshal(bundle)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func hasFinding(result Result, code string) bool {
	for _, finding := range result.Findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}
