package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	planv0 "github.com/kazi-org/wazi/contracts/plan/v0"
)

type fixtureResult struct {
	ID       string           `json:"id"`
	Path     string           `json:"path"`
	Expected bool             `json:"expectedValid"`
	Valid    bool             `json:"valid"`
	Passed   bool             `json:"passed"`
	Rule     string           `json:"rule"`
	Findings []planv0.Finding `json:"findings"`
}

type fixturesReport struct {
	ContractVersion string          `json:"contractVersion"`
	ContractDigest  string          `json:"contractDigest"`
	Passed          bool            `json:"passed"`
	Total           int             `json:"total"`
	Valid           int             `json:"valid"`
	Invalid         int             `json:"invalid"`
	Results         []fixtureResult `json:"results"`
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func run(args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: wazi-contract <version|validate|fixtures> [options]")
	}
	version, digest, err := planv0.ContractInfo()
	if err != nil {
		return fmt.Errorf("initialize embedded contract: %w", err)
	}
	switch args[0] {
	case "version":
		if len(args) != 1 {
			return errors.New("usage: wazi-contract version")
		}
		return writeJSON(out, map[string]any{"contractVersion": version, "contractDigest": digest, "authorityAuthenticated": false})
	case "validate":
		return runValidate(args[1:], version, digest, out)
	case "fixtures":
		return runFixtures(args[1:], version, digest, out)
	default:
		return fmt.Errorf("unknown command %q; use version, validate, or fixtures", args[0])
	}
}

func runValidate(args []string, version, digest string, out io.Writer) error {
	flags := flag.NewFlagSet("validate", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	pinned := flags.String("contract-digest", "", "required exact SHA-256 contract digest")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("usage: wazi-contract validate --contract-digest sha256:<digest> <file>: %w", err)
	}
	if *pinned == "" || flags.NArg() != 1 {
		return errors.New("usage: wazi-contract validate --contract-digest sha256:<digest> <file>")
	}
	if *pinned != digest {
		result := planv0.Result{Valid: false, ContractVersion: version, ContractDigest: digest, AuthorityAuthenticated: false, Findings: []planv0.Finding{{Code: "contract_digest_mismatch", Message: fmt.Sprintf("requested contract digest %q does not match embedded digest %q", *pinned, digest)}}}
		if err := writeJSON(out, result); err != nil {
			return err
		}
		return errors.New("contract digest mismatch")
	}
	data, err := readBounded(flags.Arg(0))
	if err != nil {
		return err
	}
	result := planv0.Validate(data)
	if err := writeJSON(out, result); err != nil {
		return err
	}
	if !result.Valid {
		return errors.New("bundle validation failed")
	}
	return nil
}

func runFixtures(args []string, version, digest string, out io.Writer) error {
	flags := flag.NewFlagSet("fixtures", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	pinned := flags.String("contract-digest", "", "optional exact SHA-256 contract digest")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 {
		return errors.New("usage: wazi-contract fixtures [--contract-digest sha256:<digest>]")
	}
	if *pinned != "" && *pinned != digest {
		return errors.New("contract digest mismatch")
	}
	fixtures, err := planv0.Fixtures()
	if err != nil {
		return fmt.Errorf("load fixture catalog: %w", err)
	}
	report := fixturesReport{ContractVersion: version, ContractDigest: digest, Passed: true, Total: len(fixtures), Results: make([]fixtureResult, 0, len(fixtures))}
	for _, fixture := range fixtures {
		data, err := planv0.FixtureData(fixture.Path)
		if err != nil {
			return fmt.Errorf("read fixture %s: %w", fixture.ID, err)
		}
		result := planv0.Validate(data)
		entry := fixtureResult{ID: fixture.ID, Path: fixture.Path, Expected: fixture.Valid, Valid: result.Valid, Passed: result.Valid == fixture.Valid, Rule: fixture.Rule, Findings: result.Findings}
		report.Results = append(report.Results, entry)
		if fixture.Valid {
			report.Valid++
		} else {
			report.Invalid++
		}
		if !entry.Passed {
			report.Passed = false
		}
	}
	if err := writeJSON(out, report); err != nil {
		return err
	}
	if !report.Passed {
		return errors.New("fixture conformance failed")
	}
	return nil
}

func readBounded(name string) ([]byte, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, fmt.Errorf("open input %s: %w", name, err)
	}
	defer f.Close()
	const limit = 4 << 20
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, fmt.Errorf("read input %s: %w", name, err)
	}
	if len(data) > limit {
		return nil, fmt.Errorf("input exceeds the %d byte limit", limit)
	}
	return data, nil
}

func writeJSON(out io.Writer, value any) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	if err := enc.Encode(value); err != nil {
		return fmt.Errorf("encode JSON output: %w", err)
	}
	return nil
}
