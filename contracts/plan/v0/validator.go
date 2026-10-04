// Package v0 validates portable plan contract 0.0.1 bundles.
//
// Validation establishes structural and claim coherence only. It does not
// authenticate receipts, prove grants, or execute a canonical service.
package v0

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"sort"
	"strings"
	"sync"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

const (
	maxInputBytes = 4 << 20
	maxJSONDepth  = 256
	maxJSONNodes  = 200_000
)

// Contract files are embedded so validation remains offline at runtime.
//
//go:embed contract.schema.json plan-definition.schema.json execution-snapshot.schema.json evidence-evaluations.schema.json manifest.json SEMANTICS.md fixtures
var contractFiles embed.FS

// Finding is a stable, machine-readable validation diagnostic.
type Finding struct {
	Code    string `json:"code"`
	Path    string `json:"path,omitempty"`
	Message string `json:"message"`
}

// Result reports contract and claim validation. AuthorityAuthenticated is
// always false: consumers must authenticate evidence independently.
type Result struct {
	Valid                  bool      `json:"valid"`
	ContractVersion        string    `json:"contractVersion"`
	ContractDigest         string    `json:"contractDigest"`
	AuthorityAuthenticated bool      `json:"authorityAuthenticated"`
	Findings               []Finding `json:"findings"`
}

type manifest struct {
	ContractVersion string         `json:"contractVersion"`
	Status          string         `json:"status"`
	DigestAlgorithm string         `json:"digestAlgorithm"`
	ContractDigest  string         `json:"contractDigest"`
	Files           []manifestFile `json:"files"`
}

type manifestFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type contractState struct {
	version string
	digest  string
	schema  *jsonschema.Schema
	err     error
}

var (
	stateOnce sync.Once
	state     contractState
)

func contract() contractState {
	stateOnce.Do(func() { state = loadContract() })
	return state
}

// ContractInfo returns the version and digest of the embedded, integrity-checked
// contract. An error means the binary's embedded contract failed integrity or
// schema initialization checks.
func ContractInfo() (version, digest string, err error) {
	c := contract()
	return c.version, c.digest, c.err
}

// Validate validates one bundle against the embedded contract and semantic
// rules. Oversized input, duplicate object keys, unknown versions and unavailable
// contract initialization all fail closed as structured findings.
func Validate(data []byte) Result {
	c := contract()
	r := Result{ContractVersion: c.version, ContractDigest: c.digest, AuthorityAuthenticated: false, Findings: []Finding{}}
	if c.err != nil {
		r.Findings = append(r.Findings, Finding{Code: "contract_integrity", Message: c.err.Error()})
		return finish(r)
	}
	if len(data) > maxInputBytes {
		r.Findings = append(r.Findings, Finding{Code: "input_too_large", Message: fmt.Sprintf("input exceeds the %d byte limit", maxInputBytes)})
		return finish(r)
	}
	if err := rejectDuplicateKeys(data); err != nil {
		r.Findings = append(r.Findings, Finding{Code: "invalid_json", Message: err.Error()})
		return finish(r)
	}
	var document any
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	if err := dec.Decode(&document); err != nil {
		r.Findings = append(r.Findings, Finding{Code: "invalid_json", Message: err.Error()})
		return finish(r)
	}
	if err := ensureEOF(dec); err != nil {
		r.Findings = append(r.Findings, Finding{Code: "invalid_json", Message: err.Error()})
		return finish(r)
	}
	if version := str(get(document, "contractVersion")); version != c.version {
		r.Findings = append(r.Findings, Finding{Code: "unknown_contract_version", Path: "/contractVersion", Message: fmt.Sprintf("contractVersion must be %q", c.version)})
		return finish(r)
	}
	if err := c.schema.Validate(document); err != nil {
		r.Findings = append(r.Findings, Finding{Code: "schema_invalid", Message: err.Error()})
		return finish(r)
	}
	semanticValidate(document, &r)
	return finish(r)
}

// Fixture describes a neutral, embedded conformance fixture.
type Fixture struct {
	ID    string `json:"id"`
	Path  string `json:"path"`
	Valid bool   `json:"valid"`
	Rule  string `json:"rule"`
}

type fixtureCatalog struct {
	ContractVersion string    `json:"contractVersion"`
	Fixtures        []Fixture `json:"fixtures"`
}

// Fixtures returns the catalog entries in their declared order.
func Fixtures() ([]Fixture, error) {
	if _, _, err := ContractInfo(); err != nil {
		return nil, err
	}
	b, err := contractFiles.ReadFile("fixtures/catalog.json")
	if err != nil {
		return nil, fmt.Errorf("read embedded fixture catalog: %w", err)
	}
	var catalog fixtureCatalog
	if err := json.Unmarshal(b, &catalog); err != nil {
		return nil, fmt.Errorf("decode embedded fixture catalog: %w", err)
	}
	return catalog.Fixtures, nil
}

// FixtureData reads one catalog-listed fixture from the embedded contract.
func FixtureData(name string) ([]byte, error) {
	fixtures, err := Fixtures()
	if err != nil {
		return nil, err
	}
	for _, fixture := range fixtures {
		if fixture.Path == name || fixture.ID == name {
			return contractFiles.ReadFile(fixture.Path)
		}
	}
	return nil, fmt.Errorf("fixture %q is not in the embedded catalog", name)
}

type rejectURLLoader struct{}

func (rejectURLLoader) Load(raw string) (any, error) {
	return nil, fmt.Errorf("external schema resource loading is disabled: %s", raw)
}

func loadContract() contractState {
	var result contractState
	b, err := contractFiles.ReadFile("manifest.json")
	if err != nil {
		result.err = fmt.Errorf("read embedded manifest: %w", err)
		return result
	}
	var m manifest
	if err := json.Unmarshal(b, &m); err != nil {
		result.err = fmt.Errorf("decode embedded manifest: %w", err)
		return result
	}
	result.version, result.digest = m.ContractVersion, m.ContractDigest
	if m.ContractVersion != "0.0.1" || m.Status != "frozen" {
		result.err = fmt.Errorf("embedded contract is not the frozen 0.0.1 revision")
		return result
	}
	if m.DigestAlgorithm != "sha256(sorted UTF-8 path + NUL + lowercase file SHA256 + LF)" {
		result.err = errors.New("unsupported embedded contract digest algorithm")
		return result
	}
	if err := verifyManifest(m); err != nil {
		result.err = err
		return result
	}
	compiler := jsonschema.NewCompiler()
	compiler.UseLoader(rejectURLLoader{})
	for _, name := range []string{"contract.schema.json", "plan-definition.schema.json", "execution-snapshot.schema.json", "evidence-evaluations.schema.json"} {
		raw, err := contractFiles.ReadFile(name)
		if err != nil {
			result.err = fmt.Errorf("read embedded schema %s: %w", name, err)
			return result
		}
		var resource any
		if err := json.Unmarshal(raw, &resource); err != nil {
			result.err = fmt.Errorf("decode embedded schema %s: %w", name, err)
			return result
		}
		var meta struct {
			ID string `json:"$id"`
		}
		if err := json.Unmarshal(raw, &meta); err != nil || meta.ID == "" {
			result.err = fmt.Errorf("schema %s has no valid $id", name)
			return result
		}
		if err := compiler.AddResource(meta.ID, resource); err != nil {
			result.err = fmt.Errorf("register embedded schema %s: %w", name, err)
			return result
		}
	}
	result.schema, result.err = compiler.Compile("urn:wazi:plan:0.0.1")
	if result.err != nil {
		result.err = fmt.Errorf("compile embedded contract schema: %w", result.err)
		return result
	}
	for _, id := range []string{"urn:wazi:plan:0.0.1:plan-definition", "urn:wazi:plan:0.0.1:execution-snapshot", "urn:wazi:plan:0.0.1:evidence-evaluations"} {
		if _, err := compiler.Compile(id); err != nil {
			result.err = fmt.Errorf("compile embedded schema %s: %w", id, err)
			return result
		}
	}
	return result
}

func verifyManifest(m manifest) error {
	if len(m.Files) == 0 {
		return errors.New("embedded manifest has no files")
	}
	seen := make(map[string]struct{}, len(m.Files))
	entries := append([]manifestFile(nil), m.Files...)
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	var canonical strings.Builder
	for _, entry := range entries {
		if entry.Path == "" || path.Clean(entry.Path) != entry.Path || strings.HasPrefix(entry.Path, "/") || strings.HasPrefix(entry.Path, "../") || strings.Contains(entry.Path, "\\") {
			return fmt.Errorf("manifest contains unsafe path %q", entry.Path)
		}
		if _, duplicate := seen[entry.Path]; duplicate {
			return fmt.Errorf("manifest contains duplicate path %q", entry.Path)
		}
		seen[entry.Path] = struct{}{}
		raw, err := contractFiles.ReadFile(entry.Path)
		if err != nil {
			return fmt.Errorf("manifest file %q is unavailable: %w", entry.Path, err)
		}
		sum := sha256.Sum256(raw)
		actual := hex.EncodeToString(sum[:])
		if actual != entry.SHA256 {
			return fmt.Errorf("manifest digest mismatch for %q", entry.Path)
		}
		canonical.WriteString(entry.Path)
		canonical.WriteByte(0)
		canonical.WriteString(actual)
		canonical.WriteByte('\n')
	}
	embedded := make(map[string]struct{})
	if err := fs.WalkDir(contractFiles, ".", func(name string, item fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !item.IsDir() && name != "manifest.json" {
			embedded[name] = struct{}{}
		}
		return nil
	}); err != nil {
		return fmt.Errorf("walk embedded contract: %w", err)
	}
	if len(embedded) != len(seen) {
		return fmt.Errorf("manifest covers %d files but binary embeds %d authoritative files", len(seen), len(embedded))
	}
	for name := range embedded {
		if _, ok := seen[name]; !ok {
			return fmt.Errorf("embedded contract file %q is not covered by manifest", name)
		}
	}
	sum := sha256.Sum256([]byte(canonical.String()))
	actual := "sha256:" + hex.EncodeToString(sum[:])
	if actual != m.ContractDigest {
		return fmt.Errorf("contract digest mismatch: manifest says %s, files produce %s", m.ContractDigest, actual)
	}
	return nil
}

func rejectDuplicateKeys(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	nodes := 0
	if err := readJSONValue(dec, 0, &nodes); err != nil {
		return err
	}
	if err := ensureEOF(dec); err != nil {
		return err
	}
	return nil
}

func readJSONValue(dec *json.Decoder, depth int, nodes *int) error {
	if depth > maxJSONDepth {
		return fmt.Errorf("JSON nesting exceeds the %d level limit", maxJSONDepth)
	}
	*nodes++
	if *nodes > maxJSONNodes {
		return fmt.Errorf("JSON value exceeds the %d node limit", maxJSONNodes)
	}
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	switch t := tok.(type) {
	case json.Delim:
		switch t {
		case '{':
			keys := map[string]struct{}{}
			for dec.More() {
				keyToken, err := dec.Token()
				if err != nil {
					return err
				}
				key, ok := keyToken.(string)
				if !ok {
					return errors.New("JSON object key is not a string")
				}
				if _, exists := keys[key]; exists {
					return fmt.Errorf("duplicate JSON object key %q", key)
				}
				keys[key] = struct{}{}
				if err := readJSONValue(dec, depth+1, nodes); err != nil {
					return err
				}
			}
			end, err := dec.Token()
			if err != nil || end != json.Delim('}') {
				return errors.New("unterminated JSON object")
			}
		case '[':
			for dec.More() {
				if err := readJSONValue(dec, depth+1, nodes); err != nil {
					return err
				}
			}
			end, err := dec.Token()
			if err != nil || end != json.Delim(']') {
				return errors.New("unterminated JSON array")
			}
		default:
			return errors.New("unexpected JSON delimiter")
		}
	}
	return nil
}

func ensureEOF(dec *json.Decoder) error {
	var extra any
	if err := dec.Decode(&extra); err == nil {
		return errors.New("multiple JSON values are not allowed")
	} else if !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

func finish(r Result) Result {
	sort.Slice(r.Findings, func(i, j int) bool {
		if r.Findings[i].Path == r.Findings[j].Path {
			if r.Findings[i].Code == r.Findings[j].Code {
				return r.Findings[i].Message < r.Findings[j].Message
			}
			return r.Findings[i].Code < r.Findings[j].Code
		}
		return r.Findings[i].Path < r.Findings[j].Path
	})
	r.Valid = len(r.Findings) == 0
	return r
}

func get(value any, key string) any {
	m, _ := value.(map[string]any)
	return m[key]
}

func object(value any) map[string]any {
	m, _ := value.(map[string]any)
	return m
}

func array(value any) []any {
	a, _ := value.([]any)
	return a
}

func str(value any) string {
	s, _ := value.(string)
	return s
}

func stringsOf(value any) []string {
	out := []string{}
	for _, item := range array(value) {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func truth(value any) bool { return value == true }

func hasKey(value any, key string) bool {
	_, ok := object(value)[key]
	return ok
}
