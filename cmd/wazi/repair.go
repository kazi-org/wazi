package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/kazi-org/wazi/internal/repairprofile"
	"github.com/kazi-org/wazi/internal/repairstore"
)

func runRepair(args []string, out, errOut io.Writer) int {
	flags := flag.NewFlagSet("wazi repair", flag.ContinueOnError)
	flags.SetOutput(errOut)
	data := flags.String("data", "", "owner-private candidate directory outside the selected repository")
	save := flags.Bool("save-candidate", false, "explicitly save a private immutable candidate after preview")
	apply := flags.String("apply-candidate", "", "apply the exact saved candidate ID with source check and backup")
	ai := flags.Bool("ai", false, "optional AI proposals (disabled: provider contract unqualified)")
	interactive := flags.Bool("interactive", false, "owner-value proposals (unavailable; edit reported fields and preview again)")
	env := flags.String("env-file", "", "reserved for qualified explicit AI; never loaded by deterministic repair")
	flags.Usage = func() {
		fmt.Fprintln(errOut, "Usage: wazi repair [--save-candidate] [--data DIR] FILE.md\n       wazi repair --apply-candidate ID [--data DIR]\nPreview writes nothing. Place flags before the selected file. AI is disabled.")
		flags.PrintDefaults()
	}
	if e := flags.Parse(args); e != nil {
		if e == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if *ai {
		fmt.Fprintln(errOut, "AI repair unavailable: EXPLABS API/authentication/privacy contract is unqualified; no environment file loaded or request sent")
		return 2
	}
	if *env != "" || *interactive {
		fmt.Fprintln(errOut, "this deterministic version does not load dotenv or accept interactive metadata proposals; edit reported source fields explicitly and preview again")
		return 2
	}
	storeDir := *data
	if storeDir == "" {
		home, e := os.UserHomeDir()
		if e != nil {
			fmt.Fprintln(errOut, "private data location unavailable")
			return 1
		}
		storeDir = filepath.Join(home, ".local", "share", "wazi", "repair")
	}
	if *apply != "" {
		if *save || flags.NArg() != 0 {
			flags.Usage()
			return 2
		}
		backup, e := repairstore.Apply(storeDir, *apply)
		if e != nil {
			if errors.Is(e, repairstore.ErrApplyUncertain) {
				fmt.Fprintln(errOut, "Apply durability uncertain: inspect source and compare candidate/original digests before taking another action.")
				fmt.Fprintln(errOut, "Original backup:", backup)
			} else {
				fmt.Fprintln(errOut, "apply refused:", e)
			}
			return 1
		}
		fmt.Fprintln(out, "Applied exact candidate. Original backup:", backup)
		return 0
	}
	if flags.NArg() != 1 {
		flags.Usage()
		return 2
	}
	source, e := repairstore.ReadSource(flags.Arg(0))
	if e != nil {
		fmt.Fprintln(errOut, "source refused:", e)
		return 1
	}
	result, e := repairprofile.Transform(source.Bytes)
	if e != nil {
		fmt.Fprintln(errOut, "repair refused:", e)
		return 1
	}
	fmt.Fprintln(out, "Profile: wazi-markdown-syntax-20261004 (ordinary Markdown; not portable JSON conformance)")
	fmt.Fprintln(out, "Source SHA256:", repairDigest(source.Bytes))
	fmt.Fprintln(out, "Candidate SHA256:", repairDigest(result.Candidate))
	printRepairDiff(out, source.Bytes, result.Candidate)
	blocking := false
	for _, d := range result.Diagnostics {
		fmt.Fprintf(out, "line %d [%s]: %s\n", d.Line, d.Field, d.Message)
		blocking = blocking || d.Blocking
	}
	if blocking {
		fmt.Fprintln(errOut, "Unresolved authored semantics: edit the reported fields, then preview again. Candidate not saved.")
		return 1
	}
	if *save {
		m, e := repairstore.Save(storeDir, source, result.Candidate, "wazi-markdown-syntax-20261004")
		if e != nil {
			fmt.Fprintln(errOut, "candidate save refused:", e)
			return 1
		}
		if e = json.NewEncoder(out).Encode(m); e != nil {
			return 1
		}
		fmt.Fprintln(out, "Apply explicitly with: wazi repair --data", storeDir, "--apply-candidate", m.ID)
	} else {
		fmt.Fprintln(out, "Preview only; no files written. Use --save-candidate to retain this proposal.")
	}
	return 0
}
func repairDigest(b []byte) string { sum := sha256.Sum256(b); return hex.EncodeToString(sum[:]) }
func printRepairDiff(w io.Writer, original, candidate []byte) {
	if string(original) == string(candidate) {
		fmt.Fprintln(w, "No safe syntax changes needed.")
		return
	}
	a, b := strings.SplitAfter(string(original), "\n"), strings.SplitAfter(string(candidate), "\n")
	prefix := 0
	for prefix < len(a) && prefix < len(b) && a[prefix] == b[prefix] {
		prefix++
	}
	suffix := 0
	for suffix < len(a)-prefix && suffix < len(b)-prefix && a[len(a)-1-suffix] == b[len(b)-1-suffix] {
		suffix++
	}
	fmt.Fprintf(w, "--- source\n+++ candidate\n@@ -%d,%d +%d,%d @@\n", prefix+1, len(a)-prefix-suffix, prefix+1, len(b)-prefix-suffix)
	for _, line := range a[prefix : len(a)-suffix] {
		fmt.Fprint(w, "-", line)
		if !strings.HasSuffix(line, "\n") {
			fmt.Fprint(w, "\n\\ No newline at end of file\n")
		}
	}
	for _, line := range b[prefix : len(b)-suffix] {
		fmt.Fprint(w, "+", line)
		if !strings.HasSuffix(line, "\n") {
			fmt.Fprint(w, "\n\\ No newline at end of file\n")
		}
	}
}
