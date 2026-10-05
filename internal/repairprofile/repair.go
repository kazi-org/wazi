// Package repairprofile implements the deterministic, syntax-only Markdown
// profile used by the planned Wazi repair command.
package repairprofile

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

const maxSourceBytes = 4 << 20

var (
	ErrInvalidUTF8       = errors.New("repair source is not valid UTF-8")
	ErrNUL               = errors.New("repair source contains NUL")
	ErrTooLarge          = errors.New("repair source exceeds 4 MiB")
	taskPattern          = regexp.MustCompile(`^([ \t]*)([-*+])[ \t]+\[([ \t]*[xX~-]?[ \t]*)\][ \t]+(.+)$`)
	idPattern            = regexp.MustCompile(`(?i)^(?:[A-Z][A-Z0-9]*(?:[.-][A-Z0-9]+)+|[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})\b`)
	canonicalUUIDPattern = regexp.MustCompile(`(?i)^(?:T-)?[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	fieldPattern         = regexp.MustCompile(`(?i)(?:^|[ \t])((?:canonical[ \t]+id|task[ \t]+id)|[A-Za-z][A-Za-z0-9]*(?:[-_][A-Za-z0-9]+)*)[ \t]*:`)
	bracketList          = regexp.MustCompile(`^[ \t]*[-*+][ \t]+\[[^]]*\]`)
	continuationField    = regexp.MustCompile(`^\s*[A-Za-z][A-Za-z0-9_-]*\s*:`)
)

// Diagnostic identifies a source location and a repair-profile finding.
type Diagnostic struct {
	Line     int    `json:"line"`
	Field    string `json:"field"`
	Message  string `json:"message"`
	Blocking bool   `json:"blocking"`
}

// Result contains the candidate bytes and diagnostics. Candidate is always a
// complete source projection and is safe to inspect; this package never writes.
type Result struct {
	Candidate   []byte
	Diagnostics []Diagnostic
}

// Transform applies only unambiguous checkbox syntax normalization and reports
// semantic gaps without guessing values.
func Transform(source []byte) (Result, error) {
	if len(source) > maxSourceBytes {
		return Result{}, ErrTooLarge
	}
	if !utf8.Valid(source) {
		return Result{}, ErrInvalidUTF8
	}
	if bytes.IndexByte(source, 0) >= 0 {
		return Result{}, ErrNUL
	}

	lines := splitLines(source)
	result := Result{Candidate: make([]byte, 0, len(source))}
	var tasks []task
	inFence, inComment := false, false
	fenceChar, fenceLen := byte(0), 0
	for i := range lines {
		line := lines[i].body
		trim := bytes.TrimLeft(line, " \t")
		if fenceChar == 0 {
			if c, n := openingFence(trim); n > 0 {
				inFence, fenceChar, fenceLen = true, c, n
			}
		} else if isClosingFence(trim, fenceChar, fenceLen) {
			inFence, fenceChar, fenceLen = false, 0, 0
		}
		if !inFence {
			if inComment {
				if bytes.Contains(line, []byte("-->")) {
					inComment = false
				}
			} else if at := bytes.Index(line, []byte("<!--")); at >= 0 {
				if !bytes.Contains(line[at+4:], []byte("-->")) {
					inComment = true
				}
			}
		}
		if !inFence && !inComment {
			if match := taskPattern.FindSubmatch(line); match != nil {
				checkbox := match[3]
				canonical := string(checkbox)
				status := bytes.IndexAny(checkbox, "xX~-")
				if status < 0 {
					canonical = " "
				} else if checkbox[status] == 'x' || checkbox[status] == 'X' {
					canonical = "x"
				}
				if match[2][0] != '-' { // a task checkbox makes this list marker unequivocal
					line = append(append([]byte(nil), line[:len(match[1])]...), append([]byte("-"), line[len(match[1])+1:]...)...)
				}
				// Locate the original bracket interior after a possible marker rewrite.
				open := bytes.Index(line, []byte("["))
				close := bytes.IndexByte(line[open+1:], ']') + open + 1
				if status < 0 || checkbox[status] == 'x' || checkbox[status] == 'X' {
					line = append(append(append([]byte(nil), line[:open+1]...), canonical...), line[close:]...)
				}
				tasks = append(tasks, task{line: i + 1, text: string(line)})
			} else if looksLikeTask(line) {
				result.Diagnostics = append(result.Diagnostics, Diagnostic{i + 1, "checkbox", "malformed task checkbox; clarify the task marker and checkbox", true})
			}
		}
		lines[i].body = line
	}
	inspectTasks(tasks, lines, &result.Diagnostics)
	for _, line := range lines {
		result.Candidate = append(result.Candidate, line.body...)
		result.Candidate = append(result.Candidate, line.ending...)
	}
	return result, nil
}

type sourceLine struct{ body, ending []byte }
type task struct {
	line int
	text string
}

func splitLines(src []byte) []sourceLine {
	var out []sourceLine
	for len(src) > 0 {
		i := bytes.IndexByte(src, '\n')
		if i < 0 {
			out = append(out, sourceLine{body: src})
			break
		}
		end := []byte{'\n'}
		body := src[:i]
		if len(body) > 0 && body[len(body)-1] == '\r' {
			body = body[:len(body)-1]
			end = []byte{'\r', '\n'}
		}
		out = append(out, sourceLine{body: body, ending: end})
		src = src[i+1:]
	}
	if len(src) == 0 && (len(out) == 0 || len(out[len(out)-1].ending) != 0) { /* trailing newline needs no synthetic line */
	}
	return out
}

func openingFence(line []byte) (byte, int) {
	if len(line) < 3 || (line[0] != '`' && line[0] != '~') {
		return 0, 0
	}
	i := 0
	for i < len(line) && line[i] == line[0] {
		i++
	}
	if i < 3 {
		return 0, 0
	}
	return line[0], i
}
func isClosingFence(line []byte, ch byte, n int) bool {
	i := 0
	for i < len(line) && line[i] == ch {
		i++
	}
	return i >= n && len(bytes.TrimSpace(line[i:])) == 0
}
func looksLikeTask(line []byte) bool {
	return bracketList.Match(line)
}

func inspectTasks(tasks []task, lines []sourceLine, diagnostics *[]Diagnostic) {
	ids := map[string][]int{}
	canonicalUUIDs := map[string][]int{}
	canonicalByLine := map[int]string{}
	knownStages := map[string]bool{"preflight": true, "author": true, "implement": true, "verify": true, "review": true, "fix": true, "rereview": true, "merge": true, "verify-landed": true}
	for _, t := range tasks {
		match := taskPattern.FindStringSubmatch(t.text)
		if match == nil {
			continue
		}
		tail := match[4]
		title := strings.TrimLeft(tail, " \t")
		idMatch := idPattern.FindString(title)
		if idMatch == "" {
			*diagnostics = append(*diagnostics, Diagnostic{t.line, "id", "missing task ID; add an owner-authored ID", true})
		} else {
			localID := normalizeID(idMatch)
			ids[localID] = append(ids[localID], t.line)
		}
		block := tail
		for lineNo := t.line + 1; lineNo <= len(lines); lineNo++ {
			continuation := string(lines[lineNo-1].body)
			if taskPattern.MatchString(continuation) || looksLikeTask(lines[lineNo-1].body) || strings.HasPrefix(strings.TrimSpace(continuation), "#") {
				break
			}
			if strings.TrimSpace(continuation) == "" {
				continue
			}
			if !metadataContinuation(continuation) {
				break
			}
			block += "\n" + continuation
		}
		fields := parseFields(block)
		aliases := canonicalAliasValues(fields)
		if len(aliases) > 1 {
			*diagnostics = append(*diagnostics, Diagnostic{t.line, "canonical-id", "repeated or conflicting canonical ID aliases; keep one owner-selected value", true})
		} else if len(aliases) == 1 {
			canonical := strings.TrimSpace(aliases[0])
			if canonical == "" {
				*diagnostics = append(*diagnostics, Diagnostic{t.line, "canonical-id", "empty canonical ID is ambiguous; supply one owner-selected value", true})
			} else if canonicalUUIDPattern.MatchString(canonical) {
				canonicalByLine[t.line] = canonical
				key := canonicalUUIDKey(canonical)
				canonicalUUIDs[key] = appendUniqueLine(canonicalUUIDs[key], t.line)
			}
		} else if canonicalUUIDPattern.MatchString(idMatch) {
			// A bare UUID task ID is also its canonical UUID identity in the pinned reader.
			canonicalByLine[t.line] = idMatch
			key := canonicalUUIDKey(idMatch)
			canonicalUUIDs[key] = appendUniqueLine(canonicalUUIDs[key], t.line)
		}
		for _, name := range []string{"owner", "stage", "acc"} {
			values := fields[name]
			if name == "acc" {
				values = append(append([]string(nil), values...), fields["acceptance"]...)
			}
			if len(values) == 0 {
				*diagnostics = append(*diagnostics, Diagnostic{t.line, name, missingFieldMessage(name), true})
				continue
			}
			if len(values) > 1 {
				*diagnostics = append(*diagnostics, Diagnostic{t.line, name, "repeated " + name + " metadata; keep one unambiguous value", true})
				continue
			}
			value := strings.TrimSpace(values[0])
			if value == "" {
				*diagnostics = append(*diagnostics, Diagnostic{t.line, name, "empty " + name + " metadata; supply an owner-authored value", true})
				continue
			}
			if name == "acc" && (value[0] != '[' || !balancedBrackets(value)) {
				*diagnostics = append(*diagnostics, Diagnostic{t.line, name, "acceptance must be one complete bracketed annotation", true})
			} else if name == "acc" && emptyBracketedValue(value) {
				*diagnostics = append(*diagnostics, Diagnostic{t.line, name, "acceptance annotation is empty; supply an owner-authored value", true})
			}
			if name == "stage" && !knownStages[strings.ToLower(value)] {
				*diagnostics = append(*diagnostics, Diagnostic{t.line, name, fmt.Sprintf("unsupported stage %q is preserved unchanged", value), false})
			}
		}
	}
	for _, t := range tasks {
		match := taskPattern.FindStringSubmatch(t.text)
		if match == nil {
			continue
		}
		id := normalizeID(idPattern.FindString(strings.TrimLeft(match[4], " \t")))
		if id != "" && len(ids[id]) > 1 {
			*diagnostics = append(*diagnostics, Diagnostic{t.line, "id", fmt.Sprintf("duplicate task ID %q; resolve identity explicitly", id), true})
		}
	}
	for _, t := range tasks {
		canonical := canonicalByLine[t.line]
		if canonicalUUIDPattern.MatchString(canonical) {
			key := canonicalUUIDKey(canonical)
			if len(canonicalUUIDs[key]) > 1 {
				*diagnostics = append(*diagnostics, Diagnostic{t.line, "canonical-id", fmt.Sprintf("duplicate canonical UUID %q; resolve identity explicitly", canonical), true})
			}
		}
	}
}

func appendUniqueLine(lines []int, line int) []int {
	for _, existing := range lines {
		if existing == line {
			return lines
		}
	}
	return append(lines, line)
}

func canonicalUUIDKey(value string) string {
	value = asciiCase(value, false)
	if strings.HasPrefix(value, "t-") {
		value = value[2:]
	}
	return value
}

func canonicalAliasValues(fields map[string][]string) []string {
	var values []string
	for _, alias := range []string{"canonical-id", "canonical_id", "canonical-task-id", "canonical_task_id", "canonicalid", "canonical id", "task-id", "task_id", "taskid", "task id", "uuid"} {
		values = append(values, fields[alias]...)
	}
	return values
}

func missingFieldMessage(name string) string {
	switch name {
	case "owner":
		return "missing owner metadata; add Owner: <owner> using an owner-authored value"
	case "stage":
		return "missing stage metadata; add stage: <authored token>"
	case "acc":
		return "missing acc metadata; add acc: [owner-authored acceptance]"
	default:
		return "missing required metadata; supply an owner-authored value"
	}
}

func normalizeID(id string) string {
	if strings.Contains(id, "-") && len(id) == 36 {
		return asciiCase(id, false)
	}
	return asciiCase(id, true)
}

func asciiCase(s string, upper bool) string {
	b := []byte(s)
	for i, c := range b {
		if upper && c >= 'a' && c <= 'z' {
			b[i] = c - ('a' - 'A')
		}
		if !upper && c >= 'A' && c <= 'Z' {
			b[i] = c + ('a' - 'A')
		}
	}
	return string(b)
}

func metadataContinuation(line string) bool {
	trimmed := strings.TrimLeft(line, " \t")
	if len(line)-len(trimmed) >= 2 {
		return true
	}
	if strings.HasPrefix(trimmed, "Acceptance ") || strings.HasPrefix(trimmed, "acceptance ") || strings.HasPrefix(trimmed, "Stage ") || strings.HasPrefix(trimmed, "stage ") || strings.HasPrefix(trimmed, "Status ") || strings.HasPrefix(trimmed, "status ") {
		return true
	}
	return continuationField.MatchString(line)
}

func parseFields(text string) map[string][]string {
	fields := map[string][]string{}
	all := fieldPattern.FindAllStringSubmatchIndex(text, -1)
	idx := make([][]int, 0, len(all))
	for _, m := range all {
		if bracketDepthAt(text, m[2]) == 0 {
			idx = append(idx, m)
		}
	}
	for n, m := range idx {
		name := normalizeFieldName(text[m[2]:m[3]])
		if name != "owner" && name != "stage" && name != "acc" && name != "acceptance" && !isCanonicalAlias(name) {
			continue
		}
		valueStart := m[1]
		valueEnd := len(text)
		if n+1 < len(idx) {
			valueEnd = idx[n+1][0]
		}
		value := strings.TrimSpace(text[valueStart:valueEnd])
		fields[name] = append(fields[name], value)
	}
	return fields
}

func normalizeFieldName(name string) string {
	return strings.Join(strings.Fields(strings.ToLower(name)), " ")
}

func isCanonicalAlias(name string) bool {
	switch name {
	case "canonical-id", "canonical_id", "canonical-task-id", "canonical_task_id", "canonicalid", "canonical id", "task-id", "task_id", "taskid", "task id", "uuid":
		return true
	default:
		return false
	}
}

func bracketDepthAt(s string, end int) int {
	depth := 0
	escaped := false
	for _, r := range s[:end] {
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		if r == '[' {
			depth++
		}
		if r == ']' && depth > 0 {
			depth--
		}
	}
	return depth
}

func balancedBrackets(s string) bool {
	depth := 0
	escaped := false
	for _, r := range s {
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		if r == '[' {
			depth++
		}
		if r == ']' {
			depth--
			if depth < 0 {
				return false
			}
		}
	}
	return depth == 0 && strings.HasSuffix(strings.TrimSpace(s), "]")
}

func emptyBracketedValue(s string) bool {
	s = strings.TrimSpace(s)
	return len(s) >= 2 && s[0] == '[' && s[len(s)-1] == ']' && strings.TrimSpace(s[1:len(s)-1]) == ""
}
