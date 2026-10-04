# Mac desktop contract v1

Frozen for E8 at baseline4a6d7792, 2026-10-04. Coordinator owns this contract; workers propose amendments rather than edit it. Implementation authority is the user's `$ship docs/plans/E8-native-mac-desktop.md`, local application delivery only.

## Qualified scope and environment

Phase1 artifact is macOS arm64, deployment target14.0; current runtime acceptance machine is macOS26.6.2 arm64. Do not claim macOS14 runtime or Intel qualification from a deployment target. Intel/public distribution are not promised by this local slice; architecture expansion needs its own qualified matrix. Swift6.4/CommandLineTools are installed; current main CI37231755064 passed. Shared Mac load exceeds10, so heavy local builds are held. Hosted macOS CI produces app artifacts; native acceptance runs the resulting bundle locally without local compiler work. Go baseline and shared parser fixtures remain mandatory.

Pinned Node22.23.3 darwin-arm64 official archive SHA256 `23b25245dcfb9af7262f8ff142e9e2e0af025368117329e7a7458a51e5922f53` from https://nodejs.org/dist/v22.23.3/SHASUMS256.txt. Bundle executable and full LICENSE, preserve third-party notices; verify hash before extraction. No Homebrew/node developer runtime is required. Parser closure: scripts/host-bridge.mjs, scripts/plans.mjs and src/plan-parser.mjs in their existing relative layout. No parser/schema changes planned.

Native architecture fallback: no repository SwiftUI application or general AppKit skill binding exists; consulted Apple public API documentation and installed native tooling directly, with existing Apple craft guidance for accessibility. Current Codex profiles baseline/delivery/go/swift qualify discovery only. User/repository worker instruction selects isolated GPT-6-Luna lanes; Kazi install is not a qualified native worker/runtime binding. No global plugins changed.

## Bundle and data

`Wazi.app/Contents/MacOS/Wazi` is the Swift executable. `Contents/Resources/host/wazi` is the Go binary; `Resources/runtime/node` the Node binary; `Resources/parser/{scripts,src}` the parser closure; `Resources/web` the built frontend. Info.plist uses bundle identifier `org.kazi.wazi`, deployment target14.0 and local-networking ATS scope only if required. AppKit uses public APIs, no sandbox entitlement or broad ATS bypass; no updater, LaunchAgent or persistent background daemon. Local ad-hoc signing is development-only, never Developer ID/notarization/release qualification.

Default data is owner-private `~/Library/Application Support/Wazi`; use explicit path resolution, not shell commands. Existing CLI store remains untouched and is not automatically migrated/copied; document this separate-profile compatibility decision. No duplicated memory-derived cache transfer or invented lineage. Root defaults to user's Code directory. For neutral acceptance, trusted launcher environment overrides `WAZI_DESKTOP_ROOT` and `WAZI_DESKTOP_DATA` may select fixture/root data paths; not web message inputs. App ignores inherited OpenRouter activation and starts host without enable-openrouter. UI remains honest when Serenity unavailable.

## Host CLI handshake and lifetime

Add flags `-node` and `-bridge` (absolute resource paths when desktop), preserving legacy CLI defaults. Add `-desktop-ready`, `-desktop-nonce`, `-parent-watch`. App passes root/data/assets/node/bridge, port0 and those desktop flags directly as Process.arguments, never shell interpolation. No credentials in args or URL. Only desktop mode emits one bounded JSON stdout line:

`{"protocol":"wazi-desktop/1","version":"0.1.0","origin":"http://127.0.0.1:<port>","pid":123,"nonce":"<fresh-launch-nonce>"}`

Emit after bind and required resource validation; readiness byte limit4096 and shell timeout15s. `version` identifies handshake contract, not production app readiness. Parent validates protocol/version, pid against owned Process.processIdentifier, exact random nonce, and origin scheme/http, literal127.0.0.1, port1-65535, no userinfo/path/query/fragment. stdout contains no token/source/context. Diagnostics go to bounded stderr without private bodies. A backend failure or malformed/late readiness does not navigate.

Parent holds an otherwise-unused stdin pipe open. In parent-watch mode EOF cancels host contexts and begins graceful shutdown, allowing app crash/forced quit to release owned parser children. SIGTERM/SIGINT also stop host with a bounded shutdown. Native normal quit closes stdin, waits asynchronously up to5s, then terminates only its owned child if still alive. Never pkill by name, attach to an arbitrary existing server, or kill another Wazi/agent process. New explicit restart gets fresh Process/nonce/origin/session. Duplicate launch uses macOS app-instance behavior plus a scoped native instance guard; exact lock policy is shell-owned and must be tested.

## Native view and bridge

One frameless resizable NSWindow with key/main focus and public resize handling. No native titlebar/toolbar or visible traffic lights. WKWebView fills content area and uses nonpersistent website storage. Keep the normal menu shortcuts (quit, close, copy/paste/select-all, minimize, full screen). Closing last window quits owned desktop session; no hidden host.

Before loading own origin, inject immutable `window.__WAZI_DESKTOP__` metadata `{platform:"macos",protocol:"wazi-desktop/1"}` in main frame only. JS message handler name `waziWindow`; body `{action:"close"|"minimize"|"fullscreen"|"drag"}` only, no additional fields. Require main frame and exact pinned-origin securityOrigin for every received action; reject all other forms. The action `drag` is permitted only for a current native left-mouse event over the reserved web drag region, not arbitrary canvas content; frontend sends on pointerdown of that region only. Use public NSWindow drag APIs and accessibility controls.

Coordinator/frontend owns `src/DesktopChrome.jsx`, `src/desktop.mjs`, assigned styles and App.jsx integration. Chrome appears only when qualified native marker and handler are present; normal browser has no new controls. Drag region must be a small separate hit area; never mark entire body or canvas draggable. Buttons carry close/minimize/fullscreen labels. macOS menu actions remain fallback controls.

Navigation delegates admit only exact own loopback origin; untrusted top/subframe navigation, file/data URLs and target-window loads are refused. Explicit user-activated valid HTTP(S) external links may open NSWorkspace, never automatic redirects or javascript/file schemes. Native bridge cannot run shell/IO/provider actions. File input uses WKUIDelegate native open panel for bounded Markdown/JSON selection; cancellation is ordinary. No arbitrary browser or filesystem automation exposed.

## Recovery and file ownership

Host startup/exit and web-content termination show a native truthful recovery panel. Stop/remove old webview, messages and transient sensitive DOM; retries are explicit buttons. Reload/restart never automatically replays a model operation or restores private context/answers. App close always stops owned children. Unknown provider receipts remain protected by existing Go behavior.

Exclusive lanes: host worker `cmd/wazi/` and new `internal/desktop/` only (no shared parser/portable edits); shell worker `apps/macos/` only; controls worker `src/DesktopChrome.jsx`, `src/desktop.mjs`, `src/desktop.css`, `tests/desktop.test.mjs` only. Coordinator alone integrates App.jsx/package test entry, packages under `packaging/desktop/`, workflow, docs, app resources/manifests and final evidence. All worktrees/caches/temp/generated app/browser evidence stay on SSD. Hosted CI builds avoid shared machine admission; workers must not run heavy local build/test above10 or without exact owned lease. Independent reviewer is separate from these authors.

## Coordinator amendment A — drag geometry and development isolation

Desktop web controls occupy a reserved34px header, not an overlay on navigation/canvas. The sole drag rectangle is x112..224,y4..30 in top-left WK content/CSS coordinates at page zoom1; three24px controls with7px gaps and12px side padding, followed by2px region margin precede it. Native validates mouse-down coordinates against that region; component forwards primary pointerdown only. Native bridge is main-frame/origin scoped as above. Child environment excludes NODE_OPTIONS, NODE_PATH, DYLD_* and provider credentials; trusted fixture root/data overrides remain explicit. Swift inventory corrected to6.4, installed Node22.23.2; bundled Node remains frozen22.23.3.
