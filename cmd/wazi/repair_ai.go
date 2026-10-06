package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
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
