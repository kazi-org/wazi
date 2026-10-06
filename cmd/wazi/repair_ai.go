package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kazi-org/wazi/internal/deep"
	"github.com/kazi-org/wazi/internal/repairai"
	"github.com/kazi-org/wazi/internal/repairprofile"
	"github.com/kazi-org/wazi/internal/repairrequests"
	"github.com/kazi-org/wazi/internal/repairstore"
)

var proposeRepairAI = repairai.Propose
var excludeRepairCache = deep.EnsureBackupExclusion

func runAIRepair(source repairstore.Source, storeDir, envFile string, out, errOut io.Writer) (repairprofile.Result, error) {
	if err := repairai.CheckSource(source.Bytes); err != nil {
		return repairprofile.Result{}, err
	}
	if err := checkRepairConfigScope(source.Path, envFile); err != nil {
		return repairprofile.Result{}, err
	}
	cfg, err := repairai.LoadConfig(envFile)
	if err != nil {
		return repairprofile.Result{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 65*time.Second)
	defer cancel()
	root, err := repairstore.Prepare(storeDir, source.Path)
	if err != nil {
		return repairprofile.Result{}, err
	}
	requests, err := repairstore.Prepare(filepath.Join(root, "ai-requests"), source.Path)
	if err != nil {
		return repairprofile.Result{}, err
	}
	status, backupErr := excludeRepairCache(ctx, requests)
	if backupErr != nil || !status.Excluded {
		fmt.Fprintln(errOut, "Private AI cache backup exclusion could not be verified. Other backups/sync may retain content; no universal exclusion is guaranteed.")
	} else {
		fmt.Fprintln(out, "Private AI request cache excluded from Time Machine; other backups/sync are not covered.")
	}
	fmt.Fprintln(out, "Explicit AI repair: EXPLABS", cfg.Model, "at api.experientiallabs.ai; selected plan and fixed syntax instructions only.")
	fmt.Fprintln(out, "Source SHA256:", repairDigest(source.Bytes))
	fmt.Fprintln(out, "Provider charges and account-dependent content retention may apply. One attempt; no automatic retry or apply.")
	candidate, err := repairrequests.Run(ctx, requests, repairai.Identity(cfg, source.Path, source.Bytes), func(ctx context.Context) ([]byte, error) {
		proposal, e := proposeRepairAI(ctx, cfg, source.Bytes)
		if e != nil {
			if errors.Is(e, repairai.ErrUncertain) {
				return nil, e
			}
			return nil, repairrequests.Failed(e)
		}
		if e = repairai.Validate(source.Bytes, proposal); e != nil {
			return nil, repairrequests.Failed(e)
		}
		return proposal, nil
	})
	if err != nil {
		return repairprofile.Result{}, err
	}
	if err = repairai.Validate(source.Bytes, candidate); err != nil {
		return repairprofile.Result{}, err
	}
	current, err := repairstore.ReadSource(source.Path)
	if err != nil || !bytes.Equal(current.Bytes, source.Bytes) || current.Mode != source.Mode {
		return repairprofile.Result{}, fmt.Errorf("selected source changed; proposal not saved or applied")
	}
	result, err := repairprofile.Transform(candidate)
	if err != nil {
		return repairprofile.Result{}, err
	}
	result.Candidate = candidate
	return result, nil
}

// Explicit configuration belongs to the caller, never to a selected foreign project.
func checkRepairConfigScope(source, envFile string) error {
	if envFile == "" {
		return nil
	}
	config, err := filepath.EvalSymlinks(envFile)
	if err != nil {
		return fmt.Errorf("could not qualify configuration location; no request sent")
	}
	config, err = filepath.Abs(config)
	if err != nil {
		return fmt.Errorf("could not qualify configuration location; no request sent")
	}
	source, err = filepath.EvalSymlinks(source)
	if err != nil {
		return fmt.Errorf("could not qualify source location; no request sent")
	}
	source, err = filepath.Abs(source)
	if err != nil {
		return fmt.Errorf("could not qualify source location; no request sent")
	}
	root := filepath.Dir(source)
	for d := root; ; d = filepath.Dir(d) {
		if st, e := os.Lstat(filepath.Join(d, ".git")); e == nil && (st.IsDir() || st.Mode().IsRegular()) {
			root = d
			break
		}
		if filepath.Dir(d) == d {
			break
		}
	}
	rel, err := filepath.Rel(root, config)
	inside := err == nil && (rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)))
	if !inside {
		return nil
	}
	// The explicitly selected Wazi owner configuration remains authorized, including
	// when the owner repairs Wazi's own plans. Do not infer a dotenv from the source.
	if filepath.Base(config) == ".env" {
		b, e := os.ReadFile(filepath.Join(filepath.Dir(config), "go.mod"))
		if e == nil {
			for _, line := range strings.Split(string(b), "\n") {
				if strings.TrimSpace(line) == "module github.com/kazi-org/wazi" {
					return nil
				}
			}
		}
	}
	return fmt.Errorf("configuration inside the selected project is not authorized; use owner configuration outside it; no request sent")
}
