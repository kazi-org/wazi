package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/kazi-org/wazi/internal/app"
	brain "github.com/kazi-org/wazi/internal/context"
	"github.com/kazi-org/wazi/internal/deep"
	"github.com/kazi-org/wazi/internal/desktop"
	"github.com/kazi-org/wazi/internal/observatory"
)

var desktopDiagnostics bool

func main() {
	if len(os.Args) == 1 {
		printUsage(os.Stdout)
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "repair" {
		os.Exit(runRepair(os.Args[2:], os.Stdout, os.Stderr))
	}

	root := flag.String("root", "", "bounded plan discovery root")
	assets := flag.String("assets", "dist/client", "built frontend directory")
	data := flag.String("data", "", "private app data directory")
	node := flag.String("node", "node", "Node executable used for the fixed parser bridge")
	bridge := flag.String("bridge", "", "fixed host bridge script")
	port := flag.Int("port", 0, "loopback port (0 selects an available port)")
	snapshotID := flag.String("snapshot", "", "print selected project snapshot and exit")
	enableAI := flag.Bool("enable-openrouter", false, "enable explicit user-click requests with WAZI_OPENROUTER_KEY; may incur charges")
	contextMap := flag.String("context-map", "", "private JSON repository-to-brain scope configuration; no reader is qualified by this flag")
	desktopReady := flag.Bool("desktop-ready", false, "emit the private desktop readiness handshake after bind")
	desktopNonce := flag.String("desktop-nonce", "", "fresh launcher nonce for desktop readiness")
	parentWatch := flag.Bool("parent-watch", false, "shutdown when the launcher's stdin pipe closes")
	flag.Parse()
	desktopMode := *desktopReady
	desktopDiagnostics = desktopMode
	home := os.Getenv("HOME")
	if *root == "" {
		*root = filepath.Join(home, "Code")
	}
	if *data == "" {
		if desktopMode {
			*data = filepath.Join(home, "Library", "Application Support", "Wazi")
		} else {
			*data = filepath.Join(home, ".local", "share", "wazi")
		}
	}
	if *bridge == "" {
		*bridge = filepath.Join(mustwd(), "scripts", "host-bridge.mjs")
	}
	if (*desktopNonce != "" || *parentWatch) && !desktopMode {
		fatal("desktop nonce and parent watch require -desktop-ready")
	}
	if desktopMode {
		if *port != 0 {
			fatal("desktop mode requires an ephemeral port")
		}
		if !*parentWatch {
			fatal("desktop mode requires -parent-watch")
		}
		if err := desktop.ValidateNonce(*desktopNonce); err != nil {
			fatal(err.Error())
		}
		if *enableAI {
			fatal("desktop mode does not accept inherited OpenRouter activation")
		}
		if *snapshotID != "" {
			fatal("desktop mode does not support snapshot output")
		}
	}
	absRoot, e := filepath.Abs(*root)
	check(e)
	absData, e := filepath.Abs(*data)
	check(e)
	absAssets, e := filepath.Abs(*assets)
	check(e)
	nodePath := *node
	if desktopMode {
		for name, path := range map[string]string{"root": *root, "assets": *assets, "data": *data, "bridge": *bridge} {
			if !filepath.IsAbs(path) {
				fatal("desktop " + name + " path must be absolute")
			}
		}
		if !filepath.IsAbs(*node) {
			fatal("desktop node path must be absolute")
		}
		nodePath = filepath.Clean(*node)
	}
	absBridge, e := filepath.Abs(*bridge)
	check(e)
	if desktopMode {
		if e = (desktop.Resources{Root: absRoot, Assets: absAssets, Data: absData, Node: nodePath, Bridge: absBridge}).Validate(); e != nil {
			fatal(e.Error())
		}
	}
	svc := observatory.New(absRoot, absData, nodePath, absBridge)
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
	if desktopMode {
		check(desktop.WriteReady(os.Stdout, ln.Addr(), *desktopNonce, os.Getpid()))
	} else {
		fmt.Printf("Wazi local host: http://%s (session capability held in memory)\n", ln.Addr())
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if *parentWatch {
		go func() { _, _ = io.Copy(io.Discard, os.Stdin); cancel() }()
	}
	server := &http.Server{Handler: host.Handler(), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20, BaseContext: func(net.Listener) context.Context { return ctx }}
	serveErr := make(chan error, 1)
	go func() { serveErr <- server.Serve(ln) }()
	select {
	case <-ctx.Done():
		shutdownCtx, stop := context.WithTimeout(context.Background(), 3*time.Second)
		defer stop()
		if err := server.Shutdown(shutdownCtx); err != nil {
			_ = server.Close()
		}
		if err := <-serveErr; err != nil && err != http.ErrServerClosed {
			fatal(err.Error())
		}
	case err := <-serveErr:
		if err != nil && err != http.ErrServerClosed {
			fatal(err.Error())
		}
	}
}

func printUsage(out io.Writer) {
	fmt.Fprintln(out, "Wazi local plan observatory")
	fmt.Fprintln(out, "Usage:")
	fmt.Fprintln(out, "  wazi repair [--save-candidate] [--data DIR] FILE.md")
	fmt.Fprintln(out, "  wazi repair --apply-candidate ID [--data DIR]")
	fmt.Fprintln(out, "  wazi [host flags]                         start the developer host")
	fmt.Fprintln(out, "  wazi [host flags] --assets DIR            use an explicit frontend directory")
	fmt.Fprintln(out, "Host flags include --root, --assets, --data, --node, --bridge, and --port; defaults are for local development.")
	fmt.Fprintln(out, "Run 'wazi repair --help' for repair options. No browser, desktop app, or provider starts from this help screen.")
}
func mustwd() string { x, e := os.Getwd(); check(e); return x }
func check(e error) {
	if e != nil {
		fatal(e.Error())
	}
}
func fatal(s string) {
	if desktopDiagnostics {
		s = "desktop host startup failed"
	}
	if len(s) > 512 {
		s = s[:512]
	}
	fmt.Fprintln(os.Stderr, s)
	os.Exit(1)
}
