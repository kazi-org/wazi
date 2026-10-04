package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	var output bytes.Buffer
	if err := run([]string{"version"}, &output); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["contractVersion"] != "0.0.1" || result["contractDigest"] != "sha256:7582512f122d2f2a9c4461facc7541c9887053f137260d6ebe9c6dea611d039d" || result["authorityAuthenticated"] != false {
		t.Fatalf("version result = %#v", result)
	}
}

func TestValidateRequiresExactDigest(t *testing.T) {
	var output bytes.Buffer
	err := run([]string{"validate", "--contract-digest", "sha256:wrong", "not-read.json"}, &output)
	if err == nil || !strings.Contains(err.Error(), "digest mismatch") {
		t.Fatalf("validate error = %v, want digest mismatch", err)
	}
	var result map[string]any
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["valid"] != false || result["contractVersion"] != "0.0.1" || result["authorityAuthenticated"] != false {
		t.Fatalf("mismatch result = %#v", result)
	}
}

func TestFixturesCommandRequiresNoExternalInputs(t *testing.T) {
	var output bytes.Buffer
	if err := run([]string{"fixtures", "--contract-digest", "sha256:7582512f122d2f2a9c4461facc7541c9887053f137260d6ebe9c6dea611d039d"}, &output); err != nil {
		t.Fatal(err)
	}
	var result struct {
		Passed  bool `json:"passed"`
		Total   int  `json:"total"`
		Valid   int  `json:"valid"`
		Invalid int  `json:"invalid"`
	}
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if !result.Passed || result.Total != 60 || result.Valid != 21 || result.Invalid != 39 {
		t.Fatalf("fixture result = %+v", result)
	}
}
