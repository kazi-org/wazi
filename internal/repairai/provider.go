package repairai

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const repairPrompt = "Apply only these syntax repairs to the selected Markdown plan: change task list markers * or + to -; change checkbox X to x; canonicalize redundant whitespace inside a checkbox while preserving blank unchecked status as [ ] and preserving ~ and - statuses; insert one missing space between a task list marker and its checkbox; insert one missing space after ] only when the following text begins with a valid task ID. Leave comments and fenced code unchanged. Preserve every task payload, authored fact, line ending, and every other byte exactly. Do not add, remove, reorder, or reinterpret content. Return the complete Markdown source only, without fences or commentary."
const maxResponseBytes = 4 << 20

// ErrUncertain marks a request whose provider-side completion cannot be known.
// Callers must not retry automatically.
var ErrUncertain = errors.New("repair provider outcome is uncertain")

// Identity binds a proposal to the operation, syntax policy, prompt, model,
// approved endpoint, exact path, source digest, and no credential material.
func Identity(cfg Config, path string, source []byte) string {
	h := sha256.New()
	keyScope := sha256.Sum256(append([]byte("wazi-repairai-key-scope-v1\x00"), []byte(cfg.APIKey)...))
	for _, v := range []string{"plan-repair-proposal", syntaxPolicyVersion, repairPrompt, cfg.Model, cfg.BaseURL, path, "max_tokens=16384", "stream=false", "timeout=60s", "gateway.retry.max_attempts_per_route=1", "gateway.retry.max_total_attempts=1", "gateway.retry.backoff.type=none", "gateway.routing.allow_fallbacks=false"} {
		_, _ = io.WriteString(h, v)
		_, _ = h.Write([]byte{0})
	}
	_, _ = h.Write(keyScope[:])
	d := sha256.Sum256(source)
	_, _ = h.Write(d[:])
	return hex.EncodeToString(h.Sum(nil))
}

// Propose sends one explicit syntax-only proposal request to the fixed
// Experiential endpoint. It never retries and returns only sanitized errors.
func Propose(ctx context.Context, cfg Config, source []byte) ([]byte, error) {
	if ctx == nil {
		return nil, errors.New("repair provider request requires a context")
	}
	transport := &http.Transport{Proxy: nil}
	return propose(ctx, cfg, source, &http.Client{Transport: transport, Timeout: 60 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }})
}

func propose(ctx context.Context, cfg Config, source []byte, client *http.Client) ([]byte, error) {
	if _, err := Preflight(source); err != nil {
		return nil, errors.New("repair source failed AI preflight")
	}
	if cfg.APIKey == "" || len(cfg.APIKey) > 4096 || strings.TrimSpace(cfg.APIKey) != cfg.APIKey || hasControl(cfg.APIKey) || !validModel(cfg.Model, cfg.APIKey) || validateBaseURL(cfg.BaseURL) != nil {
		return nil, errors.New("repair provider configuration is invalid")
	}
	if err := CheckSource(source); err != nil {
		return nil, err
	}
	endpoint, _ := url.JoinPath(cfg.BaseURL, "chat/completions")
	body, err := json.Marshal(map[string]any{
		"model":      cfg.Model,
		"messages":   []map[string]string{{"role": "system", "content": repairPrompt}, {"role": "user", "content": string(source)}},
		"stream":     false,
		"max_tokens": 16384,
		"gateway":    map[string]any{"retry": map[string]any{"max_attempts_per_route": 1, "max_total_attempts": 1, "backoff": map[string]string{"type": "none"}}, "routing": map[string]any{"allow_fallbacks": false}},
	})
	if err != nil {
		return nil, errors.New("could not encode repair request")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, errors.New("could not prepare repair request")
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if err := ctx.Err(); err != nil {
		return nil, errors.New("repair provider request was canceled before dispatch")
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: request transport failed", ErrUncertain)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("%w: provider returned HTTP %d", ErrUncertain, resp.StatusCode)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("repair provider returned HTTP %d", resp.StatusCode)
	}
	if ignored := resp.Header.Get("x-experiential-ignored-parameters"); ignored != "" && ignored != "[]" {
		return nil, errors.New("repair provider reported ignored request parameters")
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("%w: response body read failed", ErrUncertain)
	}
	if len(raw) > maxResponseBytes {
		return nil, errors.New("repair provider response is too large")
	}
	text, err := responseText(raw)
	if err != nil {
		return nil, errors.New("repair provider response did not contain one complete text proposal")
	}
	candidate := []byte(text)
	if err := Validate(source, candidate); err != nil {
		return nil, errors.New("repair provider proposal failed syntax-only validation")
	}
	return candidate, nil
}

type completionEnvelope struct {
	Choices []struct {
		FinishReason string `json:"finish_reason"`
		Message      struct {
			Content   json.RawMessage `json:"content"`
			Refusal   string          `json:"refusal"`
			ToolCalls json.RawMessage `json:"tool_calls"`
		} `json:"message"`
	} `json:"choices"`
	Ignored json.RawMessage `json:"x-experiential-ignored-parameters"`
}

func responseText(raw []byte) (string, error) {
	if err := rejectDuplicateJSONKeys(raw); err != nil {
		return "", err
	}
	var v completionEnvelope
	d := json.NewDecoder(bytes.NewReader(raw))
	if err := d.Decode(&v); err != nil {
		return "", err
	}
	if len(v.Choices) != 1 || v.Choices[0].FinishReason != "stop" || len(v.Choices[0].Message.Content) == 0 || v.Choices[0].Message.Refusal != "" || len(v.Choices[0].Message.ToolCalls) > 0 && string(v.Choices[0].Message.ToolCalls) != "null" {
		return "", errors.New("invalid completion")
	}
	if len(v.Ignored) > 0 && string(v.Ignored) != "null" && string(v.Ignored) != "[]" {
		return "", errors.New("ignored parameters")
	}
	var content string
	if err := json.Unmarshal(v.Choices[0].Message.Content, &content); err != nil || strings.TrimSpace(content) == "" {
		return "", errors.New("content must be nonempty text")
	}
	return content, nil
}

// rejectDuplicateJSONKeys walks JSON tokens recursively because encoding/json
// otherwise silently accepts duplicate object keys.
func rejectDuplicateJSONKeys(raw []byte) error {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var walk func() error
	walk = func() error {
		t, err := d.Token()
		if err != nil {
			return err
		}
		switch x := t.(type) {
		case json.Delim:
			switch x {
			case '{':
				seen := map[string]bool{}
				for d.More() {
					k, err := d.Token()
					if err != nil {
						return err
					}
					key, ok := k.(string)
					if !ok || seen[key] {
						return errors.New("duplicate JSON key")
					}
					seen[key] = true
					if err := walk(); err != nil {
						return err
					}
				}
				end, err := d.Token()
				if err != nil || end != json.Delim('}') {
					return errors.New("invalid JSON object")
				}
			case '[':
				for d.More() {
					if err := walk(); err != nil {
						return err
					}
				}
				end, err := d.Token()
				if err != nil || end != json.Delim(']') {
					return errors.New("invalid JSON array")
				}
			default:
				return errors.New("unexpected JSON delimiter")
			}
		}
		return nil
	}
	if err := walk(); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return errors.New("trailing JSON data")
	}
	return nil
}
