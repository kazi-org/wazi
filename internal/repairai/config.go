package repairai

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"unicode/utf8"
)

const productionBaseURL = "https://api.experientiallabs.ai/v1"
const maxEnvFile = 64 << 10

// Config contains the credentials and routing inputs for an explicit repair request.
type Config struct{ APIKey, BaseURL, Model string }

var relevantEnv = []string{"EXPLABS_API_KEY", "EXPLABS_BASE_URL", "EXPLABS_MODEL"}
var modelCredentialPattern = regexp.MustCompile(`(?i)\bsk-[a-z0-9_-]{12,}\b`)

// LoadConfig reads the process environment first and fills only absent values
// from an explicitly named file, or from .env when cwd is the Wazi repository.
func LoadConfig(envFile string) (Config, error) {
	values := map[string]string{}
	missing := false
	defaultEnv := false
	for _, name := range relevantEnv {
		if v, ok := os.LookupEnv(name); ok {
			values[name] = v
		} else {
			missing = true
		}
	}
	if missing {
		if envFile == "" {
			wd, err := os.Getwd()
			if err != nil {
				return Config{}, errors.New("could not determine configuration directory")
			}
			if b, e := os.ReadFile(filepath.Join(wd, "go.mod")); e == nil && strings.Contains(string(b), "module github.com/kazi-org/wazi\n") {
				envFile = filepath.Join(wd, ".env")
				defaultEnv = true
			}
		}
		if envFile != "" {
			fileValues, err := readEnvFile(envFile)
			if err != nil && !(defaultEnv && errors.Is(err, fs.ErrNotExist)) {
				return Config{}, errors.New("could not read configuration file safely")
			}
			for _, name := range relevantEnv {
				if _, ok := values[name]; !ok {
					if v, exists := fileValues[name]; exists {
						values[name] = v
					}
				}
			}
		}
	}
	cfg := Config{APIKey: values["EXPLABS_API_KEY"], BaseURL: values["EXPLABS_BASE_URL"], Model: values["EXPLABS_MODEL"]}
	if cfg.BaseURL == "" {
		cfg.BaseURL = productionBaseURL
	}
	if cfg.APIKey == "" || len(cfg.APIKey) > 4096 || strings.TrimSpace(cfg.APIKey) != cfg.APIKey || hasControl(cfg.APIKey) {
		return Config{}, errors.New("EXPLABS_API_KEY is required and must be a bounded single value")
	}
	if !validModel(cfg.Model, cfg.APIKey) {
		return Config{}, errors.New("EXPLABS_MODEL is required and must be a bounded single value")
	}
	if err := validateBaseURL(cfg.BaseURL); err != nil {
		return Config{}, errors.New("EXPLABS_BASE_URL must use the approved Experiential endpoint")
	}
	return cfg, nil
}

func hasControl(s string) bool {
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			return true
		}
	}
	return false
}

func validModel(model, apiKey string) bool {
	if model == "" || len(model) > 512 || model == apiKey || strings.TrimSpace(model) != model || hasControl(model) || secretPattern.MatchString(model) || modelCredentialPattern.MatchString(model) {
		return false
	}
	for _, r := range model {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("._:/-", r)) {
			return false
		}
	}
	return true
}

func validateBaseURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if u.Scheme != "https" || !strings.EqualFold(u.Hostname(), "api.experientiallabs.ai") || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" {
		return errors.New("invalid endpoint")
	}
	if !strings.EqualFold(u.Host, "api.experientiallabs.ai") && !strings.EqualFold(u.Host, "api.experientiallabs.ai:443") {
		return errors.New("invalid endpoint port")
	}
	if u.Path != "/v1" && u.Path != "/v1/" {
		return errors.New("invalid endpoint path")
	}
	return nil
}

func readEnvFile(path string) (map[string]string, error) {
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), "env")
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	owner, ok := st.Sys().(*syscall.Stat_t)
	if !ok || !st.Mode().IsRegular() || st.Size() > maxEnvFile || st.Mode().Perm()&0077 != 0 || owner.Uid != uint32(os.Getuid()) {
		return nil, errors.New("unsafe env file")
	}
	b, err := io.ReadAll(io.LimitReader(f, maxEnvFile+1))
	if err != nil {
		return nil, err
	}
	if len(b) > maxEnvFile || !utf8.Valid(b) {
		return nil, errors.New("invalid env file")
	}
	result := map[string]string{}
	s := bufio.NewScanner(strings.NewReader(string(b)))
	s.Buffer(make([]byte, 1024), maxEnvFile)
	for lineNo := 1; s.Scan(); lineNo++ {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}
		key, val, ok := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		if !ok {
			if strings.HasPrefix(line, "EXPLABS_") {
				return nil, fmt.Errorf("malformed relevant variable at line %d", lineNo)
			}
			continue
		}
		relevant := false
		for _, k := range relevantEnv {
			if key == k {
				relevant = true
			}
		}
		if !relevant {
			continue
		}
		if _, duplicate := result[key]; duplicate {
			return nil, fmt.Errorf("duplicate relevant variable at line %d", lineNo)
		}
		val = strings.TrimSpace(val)
		if len(val) > 0 && (val[0] == '\'' || val[0] == '"') {
			q := val[0]
			if len(val) < 2 || val[len(val)-1] != q {
				return nil, fmt.Errorf("malformed relevant variable at line %d", lineNo)
			}
			val = val[1 : len(val)-1]
		} else if i := strings.Index(val, " #"); i >= 0 {
			val = strings.TrimSpace(val[:i])
		}
		if strings.ContainsAny(val, "\r\n") || strings.ContainsRune(val, 0) {
			return nil, fmt.Errorf("malformed relevant variable at line %d", lineNo)
		}
		result[key] = val
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
