# Local Mac development app

Phase 1 is an arm64 Swift/AppKit shell with a nonpersistent WKWebView. Its bundle contains the Go host, pinned Node runtime, parser closure and built React/Three.js assets. The deployment target is macOS 14.0; the actual runtime acceptance matrix and exact source receipts live in the E8 plan and devlog. A target setting alone does not qualify older macOS versions or Intel.

## Build and run

The `Native Mac desktop checks` workflow creates a three-day `wazi-mac-arm64-development` artifact. It is ad hoc signed for local development, not a notarized release. Download with `gh run download <qualified-run> -n wazi-mac-arm64-development -D <new-SSD-directory>` and expand the ZIP into a new directory. Preserve an existing app rather than overwriting it. Run `python3 packaging/desktop/verify_bundle.py <Wazi.app>` before launch. There is no production deployment or release publication in this workflow.

For a local build, preserve the configured Three.js checkout, use SSD outputs/caches and claim the shared build lease after checking host load. Build the frontend, Go host and Swift shell as shown in `.github/workflows/mac-desktop.yml`; pass their paths to `packaging/desktop/package_mac.py`. The output directory must not exist. The packager verifies the pinned Node archive hash, copies runtime licenses, creates resource hashes and verifies the completed signature. Do not bypass Gatekeeper warnings or remove quarantine to compensate for an unqualified distribution build.

Open the bundle normally for the default local profile. For neutral acceptance, launch its `Contents/MacOS/Wazi` executable from an unrelated directory with `PATH=/usr/bin:/bin`, `WAZI_DESKTOP_ROOT` pointing to synthetic projects and `WAZI_DESKTOP_DATA` pointing to an owner-private SSD directory. These trusted launcher overrides are not web bridge operations. No source checkout or developer Node is required at runtime.

## Data, authority and recovery

The default store is `~/Library/Application Support/Wazi`, separate from the CLI store. No existing answers or links are migrated automatically. The instance lock is scoped to the selected data profile; a duplicate launch activates the running app instead of starting another host. The packaged host accepts only its loopback origin and requires the existing session header for mutations. Plans and code remain read-only; confirmed links use versioned `.wazi` sidecars.

Close the window or use Command-Q to quit. The app closes its owned host input pipe, waits for graceful shutdown and terminates only its own child if needed. Unexpected app death also closes that pipe. Minimize uses the normal Dock; fullscreen and standard editing actions are available from the menu and keyboard.

Host or web-content failure clears the old web view and presents a native unavailable state. Restart requires an explicit click. No uncertain AI request is automatically repeated. The desktop launcher does not enable OpenRouter and removes inherited provider credentials from the child environment. Serenity context remains honestly unavailable until its project-scoped adapter is separately qualified.

Diagnostics are bounded startup/lifecycle categories; do not include plan text, answer bodies or credentials in bug reports. Preserve the previous bundle for rollback, quit the current app, then open the previous qualified bundle with the same data profile. Rollback does not delete private data or sidecars. Signed distribution, notarization, Intel qualification, older-runtime acceptance and Windows/Linux Wails delivery are future work.

## Accessibility follow-up

Owner-approved phase1 local acceptance includes AX labels/semantics and keyboard use. Spoken VoiceOver output remains unqualified; explicitly test announced window controls, project/task navigation, code preview, importer and recovery actions in a future accessibility qualification session. This follow-up is outside the accepted local prototype merge gate and remains required before claiming spoken screen-reader qualification.
