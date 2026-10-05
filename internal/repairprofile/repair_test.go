package repairprofile

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestTransformNormalizationPreservesSuffixAndNewlines(t *testing.T) {
	source := []byte("# Plan\r\n* [  X \t] T1.0 Title  authored suffix Owner: Ada stage: implement acc: [done; keep bytes]\r\n+ [\t] T1.1 Next Owner: Bo stage: verify acc: [ok]\r\n")
	want := []byte("# Plan\r\n- [x] T1.0 Title  authored suffix Owner: Ada stage: implement acc: [done; keep bytes]\r\n- [ ] T1.1 Next Owner: Bo stage: verify acc: [ok]\r\n")
	got, err := Transform(source)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got.Candidate, want) {
		t.Fatalf("candidate mismatch\n got: %q\nwant: %q", got.Candidate, want)
	}
	if len(got.Diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", got.Diagnostics)
	}
	second, err := Transform(got.Candidate)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(second.Candidate, got.Candidate) {
		t.Fatal("transform is not idempotent")
	}
}

func TestTransformLeavesFencesAndCommentsAlone(t *testing.T) {
	source := []byte("```md\n* [ X ] T99 quoted Owner: A stage: implement acc: [x]\n```\n<!--\n+ [ X ] T98 comment Owner: A stage: implement acc: [x]\n-->\n")
	got, err := Transform(source)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got.Candidate, source) {
		t.Fatalf("fenced/comment bytes changed: %q", got.Candidate)
	}
	if len(got.Diagnostics) != 0 {
		t.Fatalf("fenced/comment content diagnosed: %#v", got.Diagnostics)
	}
}

func TestTransformPreservesAuthoredBlockedAndActiveMarkers(t *testing.T) {
	source := []byte("- [ - ] T2.0 Blocked Owner: A stage: verify acc: [blocked]\n+ [ ~ ] T2.1 Active Owner: B stage: implement acc: [active]\n")
	got, err := Transform(source)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte("- [ - ] T2.0 Blocked Owner: A stage: verify acc: [blocked]\n- [ ~ ] T2.1 Active Owner: B stage: implement acc: [active]\n")
	if !bytes.Equal(got.Candidate, want) {
		t.Fatalf("authored status marker changed\n got: %q\nwant: %q", got.Candidate, want)
	}
	if len(got.Diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", got.Diagnostics)
	}
}

func TestTransformTreatsAcceptanceAsOpaqueAndBlocksEmptyValue(t *testing.T) {
	source := []byte("- [ ] T3.0 Opaque Owner: A stage: verify acc: [owner: retained; stage words; owner=foo]\n- [ ] T3.1 Empty Owner: B stage: verify acc: []\n")
	got, err := Transform(source)
	if err != nil {
		t.Fatal(err)
	}
	var emptyBlocking, otherBlocking int
	for _, diagnostic := range got.Diagnostics {
		if diagnostic.Field == "acc" && diagnostic.Line == 2 && diagnostic.Blocking {
			emptyBlocking++
		} else if diagnostic.Blocking {
			otherBlocking++
		}
	}
	if emptyBlocking != 1 || otherBlocking != 0 {
		t.Fatalf("opaque acceptance was parsed as metadata or empty value was missed: %#v", got.Diagnostics)
	}
}

func TestTransformDoesNotInterpretOwnerValueWordsAsMalformedFields(t *testing.T) {
	for _, owner := range []string{"owner", "stage worker"} {
		t.Run(owner, func(t *testing.T) {
			source := []byte("* [X] T1.0 Do work Owner: " + owner + " kind: agent stage: implement acc: [observable result]\n")
			got, err := Transform(source)
			if err != nil {
				t.Fatal(err)
			}
			if len(got.Diagnostics) != 0 {
				t.Fatalf("ordinary owner value was misdiagnosed: %#v", got.Diagnostics)
			}
			if !bytes.Contains(got.Candidate, []byte("Owner: "+owner+" kind: agent")) {
				t.Fatalf("owner value changed: %q", got.Candidate)
			}
		})
	}
}

func TestTransformCanonicalAliasesAndUUIDCollisions(t *testing.T) {
	const uuid = "123e4567-e89b-12d3-a456-426614174000"
	source := []byte("- [ ] T1.0 First Owner: A stage: verify acc: [one] canonical_id: " + uuid + "\n- [ ] T1.1 Second Owner: B stage: verify acc: [two] uuid: 123E4567-E89B-12D3-A456-426614174000\n")
	got, err := Transform(source)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got.Candidate, source) {
		t.Fatalf("identity source bytes changed: %q", got.Candidate)
	}
	var duplicates int
	for _, diagnostic := range got.Diagnostics {
		if diagnostic.Field == "canonical-id" && diagnostic.Blocking {
			duplicates++
		}
	}
	if duplicates != 2 {
		t.Fatalf("expected both canonical UUID owners to be diagnosed: %#v", got.Diagnostics)
	}
}

func TestTransformRecognizesEveryPinnedCanonicalAlias(t *testing.T) {
	const uuid = "123e4567-e89b-12d3-a456-426614174000"
	for _, alias := range []string{"canonical_task_id", "canonicalId", "task-id", "task_id", "taskId", "task id"} {
		t.Run(alias, func(t *testing.T) {
			source := []byte("- [ ] T5.0 First Owner: A stage: verify acc: [one] canonical-id: " + uuid + "\n- [ ] T5.1 Second Owner: B stage: verify acc: [two] " + alias + ": " + uuid + "\n")
			got, err := Transform(source)
			if err != nil {
				t.Fatal(err)
			}
			var duplicateCount int
			for _, diagnostic := range got.Diagnostics {
				if diagnostic.Field == "canonical-id" && diagnostic.Blocking && strings.Contains(diagnostic.Message, "duplicate canonical UUID") {
					duplicateCount++
				}
			}
			if duplicateCount != 2 {
				t.Fatalf("alias %q was not treated as a canonical UUID: %#v", alias, got.Diagnostics)
			}
		})
	}
}

func TestTransformCanonicalAliasesRepeatOrConflict(t *testing.T) {
	const uuid = "123e4567-e89b-12d3-a456-426614174000"
	for _, tc := range []struct{ name, aliases string }{
		{"same", "canonical-id: " + uuid + " uuid: " + uuid},
		{"conflicting", "canonical_id: " + uuid + " task-id: T-123e4567-e89b-12d3-a456-426614174001"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte("- [ ] T2.0 Alias Owner: A stage: implement acc: [ok] " + tc.aliases + "\n")
			got, err := Transform(source)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got.Candidate, source) {
				t.Fatal("canonical alias bytes changed")
			}
			found := false
			for _, diagnostic := range got.Diagnostics {
				if diagnostic.Field == "canonical-id" && diagnostic.Blocking {
					found = true
				}
			}
			if !found {
				t.Fatalf("expected blocking repeated/conflicting alias diagnostic: %#v", got.Diagnostics)
			}
		})
	}
}

func TestTransformBareUUIDAndLegacyCanonicalValues(t *testing.T) {
	const uuid = "123e4567-e89b-12d3-a456-426614174000"
	source := []byte("- [ ] " + uuid + " First Owner: A stage: verify acc: [one]\n- [ ] T3.0 Second Owner: B stage: verify acc: [two] canonical-task-id: T-123E4567-E89B-12D3-A456-426614174000\n- [ ] T3.1 Legacy Owner: C stage: verify acc: [three] canonical id: legacy-id\n")
	got, err := Transform(source)
	if err != nil {
		t.Fatal(err)
	}
	var canonicalDuplicates int
	for _, diagnostic := range got.Diagnostics {
		if diagnostic.Field == "canonical-id" && diagnostic.Blocking {
			canonicalDuplicates++
		}
	}
	if canonicalDuplicates != 2 {
		t.Fatalf("bare UUID/canonical UUID collision was not reported: %#v", got.Diagnostics)
	}
}

func TestMissingMetadataDiagnosticsShowAcceptedShapes(t *testing.T) {
	got, err := Transform([]byte("- [ ] T4.0 Missing fields\n"))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"owner": "Owner: <owner>",
		"stage": "stage: <authored token>",
		"acc":   "acc: [owner-authored acceptance]",
	}
	for _, diagnostic := range got.Diagnostics {
		if shape := want[diagnostic.Field]; shape != "" && (!diagnostic.Blocking || !strings.Contains(diagnostic.Message, shape)) {
			t.Errorf("%s diagnostic lacks accepted shape %q: %#v", diagnostic.Field, shape, diagnostic)
		}
		delete(want, diagnostic.Field)
	}
	if len(want) != 0 {
		t.Fatalf("missing field diagnostics for %v: %#v", want, got.Diagnostics)
	}
}

func TestTransformDiagnosesIdentityAndSemanticAmbiguity(t *testing.T) {
	source := []byte("- [ ] T1.0 A Owner: Ada Owner: Bo stage: magical acc: [unfinished\n  continuation\n- [ ] T1.0 B stage: implement acc: [ok]\n- [ ] NoId Owner: C acc: [ok]\n- [x x] T1.2 Odd Owner: D stage: verify acc: [ok]\n")
	got, err := Transform(source)
	if err != nil {
		t.Fatal(err)
	}
	blocking, nonblocking := 0, 0
	seen := map[string]int{}
	for _, d := range got.Diagnostics {
		if d.Blocking {
			blocking++
		} else {
			nonblocking++
		}
		seen[d.Field]++
	}
	if blocking < 5 {
		t.Fatalf("expected missing/repeated/multiline/duplicate diagnostics, got %#v", got.Diagnostics)
	}
	if nonblocking != 1 || seen["stage"] == 0 {
		t.Fatalf("unsupported stage should be nonblocking: %#v", got.Diagnostics)
	}
	if !bytes.Contains(got.Candidate, []byte("stage: magical")) || !bytes.Contains(got.Candidate, []byte("[unfinished")) {
		t.Fatal("ambiguous semantics were rewritten")
	}
	if !bytes.Contains(got.Candidate, []byte("- [x x] T1.2")) {
		t.Fatal("malformed checkbox syntax was rewritten")
	}
}

func TestTransformRejectsInvalidSources(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input []byte
		want  error
	}{
		{"utf8", []byte{0xff}, ErrInvalidUTF8},
		{"nul", []byte("a\x00b"), ErrNUL},
		{"large", bytes.Repeat([]byte("a"), maxSourceBytes+1), ErrTooLarge},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Transform(tc.input)
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}
}
