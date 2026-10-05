# E12 — Local command and application discovery

fidelity: executable

Acceptance: bare `wazi` returns useful CLI help from any working directory without loading frontend assets; explicit repair and legacy desktop host invocations retain their behavior. Application discovery is qualified against the owner's actual launcher, with its icon and installation intact. No global indexing/security settings or existing app copies are changed without appropriate scope.

- [x] T12.P0 Reconcile installation, command and delivery baseline Owner: coordinator kind: agent stage: preflight acc: [main2bd85e7 clean; current local PATH command and installed Applications bundle identified; icon, importer and index state inspected; isolated SSD ownership allocated]
- [ ] T12.0 Identify and qualify the owner's CMD+K launcher Owner: owner + coordinator kind: human stage: preflight blocked: Awaiting launcher identity; Spotlight metadata is absent and filesystem index reports read-only acc: [actual launcher identified; installed Wazi with logo is visibly discoverable, or exact system restriction and safe next action are recorded]
- [x] T12.1 Make no-argument CLI successful and useful Owner: cli worker kind: agent lane: agent stage: implement deps: [T12.P0] acc: [bare wazi prints repair usage without asset/data/provider access; all explicit host arguments and repair dispatch remain unchanged]
- [ ] T12.2 Verify no-argument, repair and desktop subprocess paths Owner: coordinator kind: agent stage: verify deps: [T12.1] acc: [local changed Go tests/race/vet and unrelated-cwd command checks pass at exact revision; no source-plan writes or provider calls; hosted billing-only failures distinguished]
- [ ] T12.3 Independently review exact CLI correction Owner: independent reviewer kind: agent stage: review deps: [T12.2] acc: [base/head and full scoped correction reviewed independently; any accepted findings receive explicit fixes and re-review]
- [ ] T12.4 Rebase merge reviewed correction Owner: coordinator kind: agent stage: merge deps: [T12.3] acc: [exact reviewed head guarded; current qualified local evidence accepted when hosted CI unavailable solely for billing; protections preserved]
- [ ] T12.5 Verify landed command and local PATH update Owner: coordinator kind: agent stage: verify-landed deps: [T12.4] acc: [landed tree equals reviewed tree; exact-main binary and no-argument/repair help work outside checkout; only owned current PATH link updated atomically; records accurate]

The launcher investigation is independent of CLI correction. Existing `/Applications/Wazi.app` is preserved and its signature is valid. Spotlight's application importer recognizes its bundle and returns attributes in test mode; that does not imply persisted search indexing. No launcher identity is assumed from the shortcut. Global index rebuilds, disabling security protections, resetting LaunchServices or deleting old bundles are outside the current default fix.

Ownership: CLI worker owns cmd/wazi/main.go and new default_cli_test.go in its isolated SSD worktree. Coordinator owns delivery records and installation investigation. Independent review follows changed-behavior verification. Cloud scaling is unnecessary for this single-file implementation lane; native subscription Luna capacity suffices and no cloud spend is reserved.

Implementation handoff: the scoped CLI worker committed `bc2cc1b`; coordinator integrated it as `7fd87b7`. Formatting and diff checks passed. Heavy verification remains held while shared one-minute load exceeds 10. LaunchServices independently lists the installed `/Applications/Wazi.app` with `org.kazi.wazi`, native executable and `CFBundleIconFile = Wazi`; persistent Spotlight metadata remains unqualified.
