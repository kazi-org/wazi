package deep

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type BackupStatus struct {
	Applicable bool   `json:"applicable"`
	Excluded   bool   `json:"excluded"`
	Qualified  bool   `json:"qualified"`
	Reason     string `json:"reason"`
}

// EnsureBackupExclusion excludes only the explicitly supplied private Wazi
// cache directory from Time Machine and verifies the resulting state. Other
// platforms are reported unsupported; callers must keep memory-derived output
// ephemeral there unless they establish an equivalent qualified policy.
func EnsureBackupExclusion(ctx context.Context, privateDir string) (BackupStatus, error) {
	if runtime.GOOS != "darwin" {
		return BackupStatus{Reason: "Time Machine exclusion is unavailable on this platform"}, nil
	}
	if !filepath.IsAbs(privateDir) {
		return BackupStatus{}, errors.New("private cache directory must be absolute")
	}
	info, err := os.Lstat(privateDir)
	if err != nil {
		return BackupStatus{}, fmt.Errorf("inspect private cache directory: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return BackupStatus{}, errors.New("backup exclusion target must be a real private directory")
	}
	if info.Mode().Perm()&0077 != 0 {
		return BackupStatus{}, errors.New("backup exclusion target must be owner-only")
	}
	tool, err := exec.LookPath("tmutil")
	if err != nil {
		return BackupStatus{Applicable: true, Reason: "tmutil is unavailable"}, fmt.Errorf("locate Time Machine utility: %w", err)
	}
	check := func() (bool, error) {
		out, e := exec.CommandContext(ctx, tool, "isexcluded", privateDir).CombinedOutput()
		if e != nil {
			return false, fmt.Errorf("verify Time Machine exclusion: %w", e)
		}
		text := string(out)
		if strings.Contains(text, "[Excluded]") {
			return true, nil
		}
		if strings.Contains(text, "[Included]") {
			return false, nil
		}
		return false, errors.New("tmutil returned an unrecognized exclusion state")
	}
	excluded, err := check()
	if err != nil {
		return BackupStatus{Applicable: true, Reason: "Time Machine exclusion could not be verified"}, err
	}
	if !excluded {
		if out, e := exec.CommandContext(ctx, tool, "addexclusion", privateDir).CombinedOutput(); e != nil {
			return BackupStatus{Applicable: true, Reason: "Time Machine exclusion could not be set"}, fmt.Errorf("set Time Machine exclusion: %w: %s", e, strings.TrimSpace(string(out)))
		}
		excluded, err = check()
		if err != nil {
			return BackupStatus{Applicable: true, Reason: "Time Machine exclusion could not be verified"}, err
		}
	}
	if !excluded {
		return BackupStatus{Applicable: true, Reason: "Time Machine still reports the private cache as included"}, errors.New("Time Machine backup exclusion verification failed")
	}
	return BackupStatus{Applicable: true, Excluded: true, Qualified: true, Reason: "private Wazi cache directory is excluded and verified by tmutil"}, nil
}
