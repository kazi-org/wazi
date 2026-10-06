package repairai

import (
	"bytes"
	"errors"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/kazi-org/wazi/internal/repairprofile"
)

const syntaxPolicyVersion = "wazi-repair-ai-syntax-v1"

var (
	secretPattern    = regexp.MustCompile(`(?i)(-----BEGIN [A-Z ]*PRIVATE KEY-----|\bBearer[ \t]+[A-Za-z0-9._~+/=-]{12,}|(?:^|[^A-Za-z0-9])(?:[A-Z0-9]+[_-])*(?:API[_-]?KEY|ACCESS[_-]?TOKEN|TOKEN|SECRET|PASSWORD|AUTHORIZATION)[ \t]*[:=][ \t]*["']?(?:Bearer[ \t]+)?[A-Za-z0-9._~+/=-]{12,}|\b(?:sk-[A-Za-z0-9_-]{12,}|ghp_[A-Za-z0-9]{20,}|github_pat_[A-Za-z0-9_]{20,}|AKIA[A-Z0-9]{16})\b)`)
	spaceAfterBox    = regexp.MustCompile(`^([ \t]*[-*+][ \t]+\[[ xX~-]\])([A-Z][A-Z0-9]*(?:[.-][A-Z0-9]+)+|[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})(?:[ \t]+.*)?$`)
	spaceAfterMarker = regexp.MustCompile(`^([ \t]*)([-*+])\[([ xX~-])\]([ \t]*.+)$`)
	joinedCheckbox   = regexp.MustCompile(`^[ \t]*[-*+]\[`)
)

// Validate accepts only byte-preserving checkbox syntax normalization and a
// candidate that passes the pinned deterministic profile without diagnostics.
func Validate(source, candidate []byte) error {
	expected, err := normalizeAI(source)
	if err != nil {
		return err
	}
	if !bytes.Equal(expected, candidate) {
		return errors.New("proposal changes content outside permitted checkbox syntax")
	}
	if err := validateProtectedBytes(source, candidate); err != nil {
		return err
	}
	r, err := repairprofile.Transform(candidate)
	if err != nil {
		return errors.New("proposal failed repair profile validation")
	}
	for _, d := range r.Diagnostics {
		if d.Blocking {
			return errors.New("proposal has blocking repair profile diagnostics")
		}
	}
	if !bytes.Equal(r.Candidate, candidate) {
		return errors.New("proposal is not canonical under repair profile")
	}
	return nil
}

func containsCredentialLikeText(source []byte) bool { return secretPattern.Match(source) }

// CheckSource rejects source bytes that cannot safely enter the AI lane or be
// identified in a private proposal record.
func CheckSource(source []byte) error {
	if len(source) > 4<<20 {
		return errors.New("repair source exceeds 4 MiB")
	}
	if !utf8.Valid(source) || bytes.IndexByte(source, 0) >= 0 {
		return errors.New("repair source must be valid UTF-8 without NUL")
	}
	if containsCredentialLikeText(source) {
		return errors.New("AI proposal refused because selected content resembles a credential")
	}
	return nil
}

func normalizeAI(src []byte) ([]byte, error) {
	if err := CheckSource(src); err != nil {
		return nil, err
	}
	lines, protected := protectedAILines(src)
	out := make([]byte, 0, len(src))
	for i, line := range lines {
		b := line.body
		if !protected[i] {
			s := string(b)
			if m := spaceAfterBox.FindStringSubmatch(s); m != nil {
				s = m[1] + " " + s[len(m[1]):]
			}
			if m := spaceAfterMarker.FindStringSubmatch(s); m != nil {
				status := strings.TrimSpace(m[3])
				if status == "" {
					status = " "
				}
				payload := m[4]
				if payload[0] != ' ' && payload[0] != '\t' {
					payload = " " + payload
				}
				s = m[1] + m[2] + " " + "[" + status + "]" + payload
			}
			if joinedCheckbox.MatchString(s) {
				return nil, errors.New("ambiguous checkbox spacing")
			}
			b = []byte(s)
		}
		out = append(out, b...)
		out = append(out, line.ending...)
	}
	r, err := repairprofile.Transform(out)
	if err != nil {
		return nil, errors.New("source rejected by deterministic profile")
	}
	if err := compareProtectedLines(out, r.Candidate, protected); err != nil {
		return nil, errors.New("pinned repair profile would alter protected comment or fence bytes")
	}
	// Existing transformations are allowed by policy; malformed task-looking
	// syntax remains rejected through blocking checkbox diagnostics below.
	for _, d := range r.Diagnostics {
		if d.Blocking && d.Field == "checkbox" {
			return nil, errors.New("ambiguous checkbox syntax")
		}
	}
	return r.Candidate, nil
}

func protectedAILines(src []byte) ([]aiLine, []bool) {
	lines := splitAI(src)
	protected := make([]bool, len(lines))
	inFence, inComment := false, false
	var fence byte
	fenceN := 0
	for i, line := range lines {
		trim := bytes.TrimLeft(line.body, " \t")
		fenceLine := inFence
		if !inFence {
			if ch, n := aiOpeningFence(trim); n > 0 {
				inFence, fence, fenceN = true, ch, n
				fenceLine = true
			}
		} else if aiClosingFence(trim, fence, fenceN) {
			inFence, fence, fenceN = false, 0, 0
			fenceLine = true
		}
		protected[i] = fenceLine
		if !fenceLine {
			if inComment {
				protected[i] = true
				if bytes.Contains(line.body, []byte("-->")) {
					inComment = false
				}
			} else if at := bytes.Index(line.body, []byte("<!--")); at >= 0 {
				protected[i] = true
				if !bytes.Contains(line.body[at+4:], []byte("-->")) {
					inComment = true
				}
			}
		}
	}
	return lines, protected
}

func aiOpeningFence(line []byte) (byte, int) {
	if len(line) < 3 || line[0] != '`' && line[0] != '~' {
		return 0, 0
	}
	n := 0
	for n < len(line) && line[n] == line[0] {
		n++
	}
	if n < 3 {
		return 0, 0
	}
	return line[0], n
}

func aiClosingFence(line []byte, ch byte, width int) bool {
	n := 0
	for n < len(line) && line[n] == ch {
		n++
	}
	return n >= width && len(bytes.TrimSpace(line[n:])) == 0
}

func validateProtectedBytes(source, candidate []byte) error {
	sourceLines, protected := protectedAILines(source)
	candidateLines := splitAI(candidate)
	if len(sourceLines) != len(candidateLines) {
		return errors.New("proposal changes protected line structure")
	}
	for i, isProtected := range protected {
		if isProtected && (!bytes.Equal(sourceLines[i].body, candidateLines[i].body) || !bytes.Equal(sourceLines[i].ending, candidateLines[i].ending)) {
			return errors.New("proposal changes comment or fenced code bytes")
		}
	}
	return nil
}

func compareProtectedLines(before, after []byte, protected []bool) error {
	a, b := splitAI(before), splitAI(after)
	if len(a) != len(b) || len(a) != len(protected) {
		return errors.New("protected line structure changed")
	}
	for i, isProtected := range protected {
		if isProtected && (!bytes.Equal(a[i].body, b[i].body) || !bytes.Equal(a[i].ending, b[i].ending)) {
			return errors.New("protected bytes changed")
		}
	}
	return nil
}

type aiLine struct{ body, ending []byte }

func splitAI(src []byte) []aiLine {
	var out []aiLine
	for len(src) > 0 {
		i := bytes.IndexByte(src, '\n')
		if i < 0 {
			out = append(out, aiLine{body: src})
			break
		}
		end := []byte{'\n'}
		body := src[:i]
		if len(body) > 0 && body[len(body)-1] == '\r' {
			body = body[:len(body)-1]
			end = []byte{'\r', '\n'}
		}
		out = append(out, aiLine{body, end})
		src = src[i+1:]
	}
	return out
}
