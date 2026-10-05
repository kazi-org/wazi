package repairprofile

import (
	"bytes"
	"errors"
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
