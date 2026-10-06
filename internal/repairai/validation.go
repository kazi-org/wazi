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
	spaceAfterMarker = regexp.MustCompile(`^([ \t]*)([-*+])\[([ xX~-])\]([ \t]+.+)$`)
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
	lines := splitAI(src)
	out := make([]byte, 0, len(src))
	inFence, inComment := false, false
	var fence byte
	fenceN := 0
	for _, line := range lines {
		b := line.body
		trim := bytes.TrimLeft(b, " \t")
		if fence == 0 {
			if len(trim) >= 3 && (trim[0] == '`' || trim[0] == '~') {
				n := 0
				for n < len(trim) && trim[n] == trim[0] {
					n++
				}
				if n >= 3 {
					fence, fenceN, inFence = trim[0], n, true
				}
			}
		} else {
			n := 0
			for n < len(trim) && trim[n] == fence {
				n++
			}
			if n >= fenceN && len(bytes.TrimSpace(trim[n:])) == 0 {
				fence, fenceN, inFence = 0, 0, false
			}
		}
		if !inFence {
			if inComment {
				if bytes.Contains(b, []byte("-->")) {
					inComment = false
				}
			} else if at := bytes.Index(b, []byte("<!--")); at >= 0 && !bytes.Contains(b[at+4:], []byte("-->")) {
				inComment = true
			}
		}
		if !inFence && !inComment {
			s := string(b)
			if m := spaceAfterBox.FindStringSubmatch(s); m != nil {
				s = m[1] + " " + s[len(m[1]):]
			}
			if m := spaceAfterMarker.FindStringSubmatch(s); m != nil {
				status := strings.TrimSpace(m[3])
				if status == "" {
					status = " "
				}
				s = m[1] + m[2] + " " + "[" + status + "]" + m[4]
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
	// Existing transformations are allowed by policy; malformed task-looking
	// syntax remains rejected through blocking checkbox diagnostics below.
	for _, d := range r.Diagnostics {
		if d.Blocking && d.Field == "checkbox" {
			return nil, errors.New("ambiguous checkbox syntax")
		}
	}
	return r.Candidate, nil
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
