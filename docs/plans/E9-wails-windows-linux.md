# E9 -- Later phase: Wails on Windows and Linux

Acceptance: A reviewed executable Windows/Linux plan reuses the Go observatory and shared interface after phase-1 native delivery evidence exists.
fidelity: outline

The user selected Wails for Windows and Linux, with implementation deferred beyond Mac-only phase1. This is one future planning obligation, not permission to dispatch a port. Expand using the Mac bundle/lifecycle lessons: pin supported Wails version, Windows WebView2 and Linux WebKitGTK/distro dependencies, architecture matrix, Node/parser packaging, origin/session transport, native window controls, private data/backup behavior, platform fixtures and independent review/merge/landed gates. Do not assume Wails asset-server origins satisfy today's loopback guards or that macOS-only flock/backup mechanisms transfer unchanged. Release/update policy needs separate scope and authority.

- [ ] T9.0 PLAN expand Wails Windows/Linux delivery after Mac landed acceptance Owner: coordinator Est: 3h kind: plan stage: author delivers: [reviewed executable Windows/Linux delivery plan] deps: [T8.14] acc: [phase1 evidence and current Wails support researched; platform/use-case/dependency ownership and security/runtime gaps frozen; verification, independent review, merge and landed tasks decomposed without runtime dispatch; fidelity becomes executable only after planning artifact delivery]
