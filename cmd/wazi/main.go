package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/kazi-org/wazi/internal/app"
	brain "github.com/kazi-org/wazi/internal/context"
	"github.com/kazi-org/wazi/internal/deep"
	"github.com/kazi-org/wazi/internal/observatory"
)

func main() {
	root := flag.String("root", filepath.Join(os.Getenv("HOME"), "Code"), "bounded plan discovery root")
	assets := flag.String("assets", "dist/client", "built frontend directory")
	data := flag.String("data", filepath.Join(os.Getenv("HOME"), ".local", "share", "wazi"), "private app data directory")
	port := flag.Int("port", 0, "loopback port (0 selects an available port)")
	snapshotID := flag.String("snapshot", "", "print selected project snapshot and exit")
	enableAI := flag.Bool("enable-openrouter", false, "enable explicit user-click requests with WAZI_OPENROUTER_KEY; may incur charges")
	contextMap := flag.String("context-map", "", "private JSON repository-to-brain scope configuration; no reader is qualified by this flag")
	flag.Parse()
	absRoot, e := filepath.Abs(*root)
	check(e)
	absData, e := filepath.Abs(*data)
	check(e)
	absAssets, e := filepath.Abs(*assets)
	check(e)
	svc := observatory.New(absRoot, absData, "node", filepath.Join(mustwd(), "scripts", "host-bridge.mjs"))
	if *snapshotID != "" {
		_, e = svc.Discover(context.Background())
		check(e)
		snap, e := svc.Snapshot(context.Background(), *snapshotID)
		check(e)
		check(json.NewEncoder(os.Stdout).Encode(snap))
		return
	}
	if st, e := os.Stat(absAssets); e != nil || !st.IsDir() {
		fatal("frontend assets missing; build frontend first")
	}
	var contextService *brain.Service
	if *contextMap != "" {
		raw, err := os.ReadFile(*contextMap)
		if err != nil {
			fatal("context configuration unavailable")
		}
		var config brain.HostConfig
		if err = json.Unmarshal(raw, &config); err != nil {
			fatal("context configuration invalid")
		}
		contextService, err = brain.NewService(config, nil)
		if err != nil {
			fatal("context scope configuration invalid")
		}
	}
	var engine deep.Engine
	if *enableAI {
		provider, err := deep.NewOpenRouter(os.Getenv("WAZI_OPENROUTER_KEY"))
		if err != nil {
			fatal("explicit OpenRouter configuration unavailable")
		}
		engine = provider
	}
	analysisDir := filepath.Join(absData, "answers")
	check(os.MkdirAll(analysisDir, 0700))
	backup, backupErr := deep.EnsureBackupExclusion(context.Background(), analysisDir)
	if backupErr != nil || !backup.Qualified {
		fmt.Fprintln(os.Stderr, "memory-derived durable storage remains unqualified")
	}
	// No supported owner Reader is present yet; configuration alone cannot qualify memory persistence.
	analysis, err := deep.New(deep.Config{Dir: analysisDir, MaxBytes: 64 << 20, Engine: engine, Validator: app.LineageAdapter{Service: contextService}, MemoryPersistenceQualified: false})
	if err != nil {
		fatal("private analysis storage unavailable")
	}
	routes := &app.Routes{Observatory: svc, Context: contextService, Deep: analysis}
	host, e := observatory.NewHost(svc, http.FileServer(http.Dir(absAssets)), routes.Handle)
	check(e)
	bind := fmt.Sprintf("127.0.0.1:%d", *port)
	ln, e := net.Listen("tcp", bind)
	check(e)
	fmt.Printf("Wazi local host: http://%s (session capability held in memory)\n", ln.Addr())
	server := &http.Server{Handler: host.Handler(), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20}
	check(server.Serve(ln))
}
func mustwd() string { x, e := os.Getwd(); check(e); return x }
func check(e error) {
	if e != nil {
		fatal(e.Error())
	}
}
func fatal(s string) { fmt.Fprintln(os.Stderr, s); os.Exit(1) }
