# ADR 0003 — Platform desktop shells around the Go observatory

Status: accepted product direction; implementation unstarted. Date: 2026-10-04.

## Decision

Use Swift, AppKit and WKWebView for the Mac application. Phase 1 is Mac-only. Reserve Wails for Windows and Linux in a later phase. Keep the existing Go observatory and React/Three.js interface as the shared product; do not create a second plan reader, storage authority or scheduler in Swift or Wails.

The Mac shell owns one frameless, resizable, keyboard-focusable window and one bundled Go child process. It loads the exact loopback origin announced by that child after a bounded readiness/version handshake. Preserve existing HTTP host/origin/session protections; do not replace them with unrestricted file URLs or a permissive native bridge. Package built web assets and the existing Node parser bridge plus a pinned, licensed Node runtime. End users must not need Node, Go, npm, a terminal, the source checkout or the Three.js dependency checkout. A Go parser rewrite is outside this phase and must not alter the frozen portable contract.

Frameless means no native title bar, toolbar or browser navigation chrome. The web interface fills the content view. Use a small dedicated drag region and accessible custom window controls; never turn the interactive 3D canvas into a window drag surface. Public AppKit APIs handle window movement, resizing, full screen and keyboard behavior. Closing the last window ends the desktop session and shuts down its owned host; no hidden service or launch-at-login feature is introduced. Reopening launches a fresh session while retaining private durable state.

Native messaging is limited to named window actions. Validate the main-frame origin and message shape; no arbitrary shell execution, filesystem reads, URLs or credential access. Navigation and new-window delegates keep the webview on its pinned local origin. Explicit external links may open a validated HTTP(S) destination in the system browser. WebKit website state is ephemeral; private answers and link sidecars remain Go-owned. On host or web-content failure, show a truthful recovery screen, clear transient sensitive UI and never automatically repeat an AI request. Host restart is an explicit action and creates a fresh origin/session.

Use app-private Application Support storage, with an explicit non-destructive compatibility/migration decision for the current CLI store before implementation. Maintain permissions, backup exclusion and owner-lineage rules; shipping a shell does not qualify memory persistence. No source plans, code or private snapshots are bundled. The existing unavailable Serenity panel remains honest: integration is a separate E3 requirement and not a prerequisite for wrapping the current observatory.

## Delivery scope

Phase 1 ends with a locally runnable `.app`, integrated behavior verification, independent exact-head review, source rebase merge and landed local-app verification. Freeze the macOS deployment target and supported architecture matrix during preflight; claim only combinations actually verified. A local ad-hoc development signature may be used for testing. Developer ID signing, notarization, App Store submission, public distribution, auto-updates, paid providers and production deployment are outside this authorization.

Windows/Linux retain the same Go ownership and frontend through Wails; exact Wails version, WebView2/WebKitGTK requirements, transport/origin mapping and packaging are deferred to the dependency-triggered phase-2 plan. Do not port the Mac shell to Wails.

## Rationale and qualification

This is the user's explicit platform decision. It provides native Mac window ownership while reusing the current observatory. Node packaging, host lifecycle, WebKit WebGL behavior and window focus/resize are real qualification work, not solved by choosing a wrapper. Existing Chromium headless acceptance does not prove WKWebView acceptance.

Public API references: [NSWindow drag](https://developer.apple.com/documentation/appkit/nswindow/performdrag(with:)), [Process lifecycle](https://developer.apple.com/documentation/foundation/process), [WKNavigationDelegate](https://developer.apple.com/documentation/webkit/wknavigationdelegate), [web-content termination](https://developer.apple.com/documentation/webkit/wknavigationdelegate/webviewwebcontentprocessdidterminate(_:)), [Wails frameless windows](https://v2.wails.io/docs/guides/frameless/).
