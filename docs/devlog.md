# Devlog

## 2026-10-04: E3 Go candidate, independent audit and blocked delivery

**Type:** finding
**Tags:** observatory, Go, traceability, private-results, scope, delivery

**Problem:** E3 requires selected-project code/test traceability, a Go host, rich scoped shared-brain context and lifecycle-qualified durable model answers. Existing main was clean at `f1cd3e7`; the richer owner API was not implemented.
**Root cause:** Read-only owner source audit at Serenity `e2b5dd17` found fact-only provider-free reads and brain-wide DIRECTION sections; neither qualifies the five-section project seam or all-type lineage. Chrome blocked the candidate loopback page with `ERR_BLOCKED_BY_CLIENT`, and the in-app browser was unavailable.
**Fix:** Expanded T3.0, used three isolated GPT-6-Luna implementation lanes, and integrated the Go host, source parser digest seam, bounded suggestions/previews, versioned sidecars, fail-closed context contract, metadata-only private request receipts and explicit fixed-model adapter. Preliminary exact-head review at `ca71d56` reported R01–R09; source fixes and regressions are tracked separately. Failed integration CI runs exposed task ID and scope-policy mismatches; they were fixed before accepted checks.
**Verification:** Candidate `bd965e6` CI `37192461087` passes both jobs including Go race/vet, the frozen 60-fixture suite and frontend tests/build. CI-built Mac host at `c43ba3c` (backend unchanged in `bd965e6`) passed neutral SSD API smoke for root separation, selected-only snapshots, metadata, confirm/manual/CAS, source/reverse links, confinement, unavailable context and provider-disabled behavior. No analysis receipts were created by disabled model attempts. Browser behavior, real owner connectivity and live provider execution remain unverified. Local multi-package commands were held while load or the shared lease excluded this lane; no foreign lease was released.
**Impact:** Draft PR #3 is reviewable source progress, not complete E3. Full review, GitHub rebase merge and landed verification remain blocked. Frozen portable contract and existing source parser semantics are unchanged; source plans/code were read-only except owned neutral fixture sidecars. No other repository edits, paid provider call, release or deployment occurred.

## 2026-10-03 — Plan observatory prototype

Chose a dark spatial observatory from the supplied diagram's columns and dependency topology. User subsequently confirmed continuing with the chosen direction. Plan parsing and discovery were implemented in an isolated external-SSD worker lane; frontend and integration in a separate coordinator worktree. The original reference is preserved and the local Three.js checkout is untouched.

Verification: ten focused Node tests pass. Single-package Vite production build passes (2.26 seconds), with a bundle-size advisory. The build lease was held by another lane; the single-package command falls outside the multi-package lease requirement. The coordinator mistakenly started the build before inspecting a refreshed uptime result above 10; it completed before interruption. No further heavy commands were run at that load. Browser interactions and visual comparison are recorded in design-qa.md. Live repository discovery found 73 projects; Serenity hit the configured 120 split-file cap, reported by a visible warning. Imported browser data is ephemeral and source plans remain read only.

Chrome extension file chooser automation could not set local files. A Markdown paste path was added and verified. Both paths share the same pure parser. No remote publication or deployment was performed. Screenshots and temporary fixtures stay in the external task worktree, outside Git.

Independent review passed at exact candidate `f60775f733bc300c75ae3a9a613aeffb433e2a90` against bootstrap base `be3992b`. Local main fast-forwarded to that candidate and source/package/config equality was verified. The running development server now uses the original workspace checkout; its ignored node_modules symlink keeps dependencies and Vite caches on the external SSD. Landed browser reload rendered all 17 sample tasks with no new runtime errors. Original reference and Three.js checkout remain unchanged.

## 2026-10-03 — Reverse-engineered documentation and cross-session coordination

User requested architecture/RFC/design documentation in docs/ and communication through ajent.social. Captured baseline `8c5486d`, inspected actual scanner/parser/API/UI/scene code and the existing verification records, and reserved a docs-only lane in the project channel. Two Codex processes were observed with this project's working directory; process presence alone does not identify which is the peer or prove a message was read. No wake signals or notifier were introduced.

Documentation is being authored in external-SSD worktrees with explicit file ownership. Runtime changes, builds, hosted delivery and private plan snapshots are outside this lane. Existing design/plan/devlog content is preserved with additive updates. Important implementation caveats now documented include separate scan/import status derivation, per-plan warnings not surfaced by the UI, request-ordering gaps, browser normalization of source IDs, static-vs-development API behavior and bounded discovery. These are findings, not accepted runtime fixes.

Peer exchange: received `SERENITY-WAZI-PLAN-CODE-20261004-01`, then posted `WAZI-TRACEABILITY-ACK-20261004-01`. The reply proposed explicit plan-to-file/test bindings, a small Go CLI/library producing versioned JSON, preservation of separate plan/source/check/judgment authority, and optional Serenity contextual reads later. Recorded that direction in proposed RFC 0002 and an outline planning item, without treating the channel handoff as an implementation assignment or a verified Serenity API. Peer interface feedback remains pending at this boundary.

Documentation verification: 81 local Markdown links resolved, whitespace check clean and runtime files unchanged from `8c5486d`. Independent documentation review passed at candidate `34136b5`, confirming factual accuracy and the proposal/implementation authority boundary. No builds or tests were rerun for this docs-only change.

Documentation integrated into local main by fast-forward and 81 local links rechecked. Runtime/config/package/test files remain identical to baseline `8c5486d`; no deployment or remote publication. The channel was reread at integration and contained the peer handoff plus our acknowledgement/proposal and candidate handoff; no further peer interface reply had arrived. The docs lane is complete; future traceability remains a proposed outline.

## 2026-10-03 — Design settlement watch and clarification

User asked both sessions to watch the append-only channel until agreement, resolve intent through questions, record the settled design and stop. Direct choices established local plan discovery, code analysis only for selected projects, automatic candidate links requiring confirmation, first-version read-only shared-brain project-scoped Serenity context, a Go preference, and click-triggered OpenRouter `openai/gpt-6-luna` with persisted exact reuse. These supersede the earlier narrow first-slice recommendation.

Peer replies `SERENITY-WAZI-DESIGN-REPLY-20261004-01` through `03` agreed the broader version and Go local-host/CLI shape, and identified provider calls hidden behind some read APIs, inadequate project isolation in global Brief, and lifecycle risks of memory-derived cached answers. Final questions cover Go+browser confirmation, fact-only versus richer authored context, and whether AI may receive Serenity context. No default is inferred while answers are pending.

Prepared RFC/ADR drafts in an external-SSD docs-only worktree. Independent preliminary contract review found no blocking contradiction and requested explicit private cache location/permissions/retention/backup policy; those defaults were added. No source plans, runtime code or dependency checkout changed, and no models/credentials/services were invoked. The root channel remains ignored; the remote repository was verified public, so no channel or private result data is published.

Explicit Go/browser answer received in both conversations: `WAZI-DESIGN-USER-GO-20261004-01` and `SERENITY-WAZI-USER-ANSWER-20261004-01`. Independent draft review passed at `220200c` after cache privacy, discovery confinement and paid-request capacity-failure findings were resolved. Peer draft review of `1ce79b9` concurred with agreed contracts, requesting consistent Go-answer recording and an explicit warning that Brief TaskHint does not project-scope authored ledger context; both are reflected in the next draft. The two remaining content/input choices are still pending.

Direct rich-context answer received as `WAZI-DESIGN-USER-CONTEXT-20261004-01`: facts, decisions, constraints, intents and open questions with honest unavailable sections. RFC/design/ADR now require the scoped authored-context owner seam as a v1 integration/acceptance dependency, without inventing an available API or weakening privacy. Only the AI input scope question remains pending.

Final AI input answer received directly as `WAZI-DESIGN-USER-AI-20261004-01` and corroborated by peer `SERENITY-WAZI-USER-ANSWERS-20261004-02`: selected plan/code plus the displayed project-scoped Serenity context. All material choices are resolved. Frozen digest v1 adds required full-context lineage/eligibility/expiry validation before durable answer reuse, derived-body invalidation/removal and truthful ephemeral-only/persistence-unavailable behavior until qualified. No paid request or backend implementation occurs.

User additionally requested introduction/watch/participation in the shared plan-skill channel until discussion completion. Introduced the Wazi design lane there, without claiming schema/adapter implementation. This adds discussion-only scope after the Wazi design record; recommendations and approved/runtime claims remain distinct.

Design settled at semantic candidate `53f830d9ae31beb9651a8876e8f7d285eca7eb5a`. Peer final refinement concurrence and independent final-delta PASS agree on the same head. Recorded accepted design status in RFC/ADR, resolved all cache/source-confinement/offline findings, and kept runtime/owner-contract/provider qualification explicitly unpassed. Local documentation integration is the next step; runtime remains untouched.

Accepted design/status record `9d652075650c4e04c0f7432b0a262be79aa532ec` fast-forwarded into local main. Landed verification found a clean tree, 86 resolved local links and docs-only differences from `6b63dfd`; runtime/config/package/tests and original sources are unchanged. E4 design settlement is complete. No remote publication, backend implementation, provider request or actual brain connection. Continued only the separately requested shared plan-contract discussion/watch, with stewardship still open to the named user decision.

## 2026-10-03 — Shared plan-skill discussion completed

Introduced `wazi-observatory-design`, responded to portable full-SDLC identity/source/gate/evidence questions and confirmed the retained shared synthesis. A lightweight Node parser probe against baseline `6b63dfd` reproduced `stage: review lifecycle: enrolled provider: foundry` being absorbed as one stage value; `rereview`/`deploy` remain unsupported presentation stages routed to Build by `src/demo.mjs`. This is a documented compatibility/conformance case, not an implemented fix or a full acceptance test. Source checkbox status remains an authored assertion rather than trusted delivery evidence.

Observed `DISCUSSION-CLOSED-01` and `DELIVERY-METADATA-01`, verified the retained record's SHA256 `6e17965153a8e8967372c52c64f935872737543e7227282facb6a86e62e2b69c`, and verified local shared-skills main contains auto-sync record commit `2564160bd7b8a51ea8801d70f0ad57aeab6ec1a5`. No remote publication was checked or performed by this lane. All participants endorsed recommendation-level boundaries; Wazi schema stewardship remains explicitly OPEN with the user, and consumer/runtime/schema work requires separate scope. Acknowledged the closing receipt and ended the watch; persisted only the Wazi design/handoff reference here. No source/runtime changes, schema files, paid requests or service/deployment actions.

## 2026-10-03 — Collapsible side panels

Implemented the direct UI request in an isolated external-SSD lane. Changes are confined to panel UI, scene resizing and delivery records: independent project/inspector expansion, icon-only compact navigation, preserved selection, desktop width reclamation and existing narrow drawers/overlays. Stabilized task arrays across panel toggles and adjusted ResizeObserver to initialize the home view once instead of resetting orbit/zoom on every resize. No parser, source plan, API or dependency checkout changes.

Production `npm run build` passed at one-minute load 9.28 with generated output and temporary files on the SSD. This is a single-package frontend build; the optional shared lease attempt returned LOST and its unrelated owner was left untouched. The existing large-bundle advisory remains. Real Chrome checks at 1278 x 1080 and 390 x 844 covered four desktop panel combinations, keyboard activation, retained selected task, task-click reopening, mobile drawer open/close, mobile inspector close/reopen, hidden drawer accessibility, matching canvas/host widths and no horizontal overflow. Compact desktop scene width is 1170 px; both expanded yields 771 px. Browser error log was empty. Physical-device touch remains outside this check. Independent exact-head review and local integration are pending.

The final camera audit found the resize callback itself also called `home()`; that call was removed, preserving initial/explicit reset behavior. Settled central-column projection/scale values remained identical before and after a navigation-panel resize following a real canvas orbit gesture. Review feedback added modal drawer focus entry/wrapping, inert scene/details, focus restoration on Escape/project selection, correct focus on drawer-to-Help/Import transitions, and a visible inspector reopen control receiving focus after collapse. These interactions were checked in Chrome. Final successor build passed; exact-head review is pending.

Independent read-only review PASS at exact implementation head `77a9d502f39c969e03fb8cfcd0485c4fe2912ae9`, base `7021c382704877d1511f3ead1ec74b7d470c1e2a`: no actionable blocker. Review confirmed camera retention/reset behavior, panel sizing, responsive controls, drawer focus/inert behavior, modal transition focus and inspector close focus. The reviewer did not run builds or services. Final build passed at one-minute load 5.27; additional Chrome verification at 900 x 800 found an 834 px scene, working inline inspector controls and no overflow/errors. All 86 local documentation links resolve and whitespace checks pass. Root main remains clean at the base; local fast-forward integration is next.

Reviewed implementation and status record fast-forwarded into local main at `f3e65665358011d718e12474164aeab0d9b1b5ba`. Landed App/CSS/Space files match exact reviewed head `77a9d50`; main Chrome preview independently verified default 66 px icon navigation, 1170 px scene, both panel expansion/collapse and preserved selected task. Error log remained empty. The viewport override was reset and the main preview left open; the temporary candidate preview is closed. E6 is complete. No remote publication or deployment.

## 2026-10-04 — PC-WAZI portable contract delivery

Direct user ship request follows the bounded implementation dispatch, superseding
the earlier discussion-only stewardship restriction for this package. Source
inventory found the existing Markdown parser, Vite loopback discovery and
React/Three.js display; no Go module or existing GitHub base/checks exist. Main
was clean at bdf61d0. Separate SSD validator and display lanes preserve all old
worktrees. Installed Kazi cannot route the active Codex harness/model, so the
qualified current-session GPT-6-Luna fallback is used without installing tools
or activating another provider. Heavy build/test execution is held above load10.

Candidate 0d23e9a published to shared channel; CT001 duplicate manifest catalog
entry and CT002 conflicting logical evaluations were accepted and fixed in
127c393. Independent bounded contract audit passed that exact head. Consumer
feedback made the single-file Markdown bound, split/source diagnostics, native
metadata/array acceptance, heterogeneous subject and context/policy distinctions
explicit. Frozen 0.0.1 specification at16b66e5 has60 neutral cases and digest
7582512f122d2f2a9c4461facc7541c9887053f137260d6ebe9c6dea611d039d.
Adapters may pin those immutable bytes; Go conformance is still pending and no
runtime or receipt authentication is inferred. Initial remote publication must
avoid disclosing historical local-only machine paths or agent channels.

Go implementation integrated from a422484/c7cd678: embedded specification integrity,
network-denying schema loader, bounded duplicate-key-safe JSON parsing, structural
and semantic checks, CLI and 60-case conformance tests. `go mod tidy` resolved the
required indirect x/text dependency with all caches on SSD; this was dependency
resolution only, not a held build/test. Display integrated from2f7f34c: Node15/15
passed with SSD temporary files, browser retained qualified IDs/raw source/opaque
metadata and showed pending/0% authored progress despite reported complete,
verified and satisfied claims. Unknown contract0.0.2 rejected; rereview shown
as Other. Browser findings D1 end-column clipping during inspector resize and
D2 collapsed visual acceptance newlines have tracked fix/verification rows;
source fixes are in progress. Heavy Mac checks remain held; GitHub Linux CI
provides scoped Go/Node/build verification and a disposable Mac verifier artifact.

Initial public main is audited clean root snapshotadc4e8d; normative0.0.1 source
is also publicly pinned atf04497a with unchanged manifestdigest7582512f. No local
development history or agent channel is published. This necessary initial-base
step is distinct from the upcoming reviewed implementation rebase merge.

Integrated verification: CI run37183983768 caught G1 unused Go local while the
observatory checks passed. Fix7ecf984 removed that local without changing frozen
specification bytes. CI run37184251198 passed at publice7ed7b8: Go API/CLI tests,
60 fixture outcomes, Node15 tests and production build. Its Darwin arm64 verifier
SHA256caa915df2bc1f08f495ec7b452bf7bbe9e67edd2e1c2f350c71ac043c5688100
ran offline with unreachable proxies: version/fixtures/valid pass, invalid review
and wrong digest fail with exit2, authorityAuthenticated remains false. D1/D2
fixes04d8c08/ba7da1e verified in Chrome: desktop panels resize with final lane
visible, Reset/Map return coherent fit,390px no overflow, acceptance pre-wrap,
opaque records preserved, no console errors. Public PR1 is draft pending exact
integrated independent review; no branch protections/rulesets exist, but both
scoped checks are required by this delivery. Heavy Mac builds stayed held.

Independent pre-merge probe G2: a review for a different head/base could lend
independence to a current review without proof. The existing CI binary accepted
that mixed bundle because proof selection did not apply current binding checks.
The coordinator initial audit-only probe altered a check instead of the review
and was invalid as a counterexample; independent review confirmed audit-only
review proof is already rejected. Tracked separate fix/verify/re-review tasks; frozen0.0.1 bytes remain
unchanged and merge waits for corrected conformance and independent review.

G2 fix7b41e78 delegates proof selection to the same qualifying-evidence binding
as the satisfied gate and scans for a qualifying proof. Added mixed/order and
wrong task/attempt/revision/policy/head/base tests without frozen fixture edits.
CI37184831031 passed public8693d3d. Offline artifact SHA256
f26d7731a0b4f9662dedb7d007ee375236c7dc8f7f3cff9393cf9d0aab59695e
rejects the actual wrong-subject review probe (exit2/independence_missing), where
the prior binary returned valid. All60 frozen fixtures and legitimate current
attested review still pass; wrong digest fails closed. Final re-review/merge wait.

Independent full review (review_observatory; baseadc4e8d/headb45c705) returned
BLOCKED P1/G2 review proof binding and P2/G3 browser duplicate JSON keys. Go rejects
duplicate keys, but JSON.parse silently selected the last authored title; display
import must reject duplicates before conversion. G3 fix/verify assigned to the
UI lane and joined to exact corrected re-review. No other scoped blocker found.

G2 independent correction review closed the wrong-subject finding at0244526,
confirming60cases, audit-only rejection, subject mismatch and valid-proof ordering.
G3 browser fix79a4acc scans decoded keys (including escaped and surrogate-equivalent
names) before JSON.parse; bounds match Go4MiB/depth256/200000values. Node16 tests
pass. Chrome duplicate-title import now rejects without replacing the current
selection; normal valid import still works and no new console error appears.
Final integrated CI and independent joint re-review remain required.

CI37185145653 passes public357a806 with both G2/G3 corrections (Go/60fixtures,
Node16 and production build). Joint independent exact-head re-review is next.

PC-WAZI complete. Independent review_observatory final PASS at public57dd957 vs
baseadc4e8d covers T7.1/T7.3/T7.4; G2/G3 closed. PR1 merged by guarded GitHub
REBASE on2026-10-04T07:22:35Z at47b9d91bca0d30ac44337a6e5aa710efa89bfcfd.
Fetched target has exactly the reviewed tree and preserves base ancestry.
CI37185210114 (candidate) and37185594744 (landed) both pass contract/observatory
checks: Go tests,60 fixture outcomes,Node16 and production build. Landed Darwin
verifier SHA256ea086c07024c3cb384e3327fbc037659878d12805660650e4d61753e97ee5f70
ran offline: version/fixtures/valid pass; wrong digest and borrowed wrong-subject
proof reject with exit2. Chrome main valid import and duplicate rejection retain
selection without errors. Earlier full browser QA applies to identical code.
Frozen0.0.1 public specification f04497a/digest7582512f remains unchanged.
PC-WAZI-LANDED-01 handoff records durable Go source pin and disposable artifact
retrieval, commands and trust limits. Original local history was preserved on
archive/local-main-pre-public-20261004; local main tracks the public landing and
its preview is open. E7 is complete; broader E3 observatory runtime remains future
work. No production/provider/deployment or consumer authentication claim.

## E3 corrected source review receipt

Independent reviewer `/root/review` passed exact `88bee528e840c9d2c52ff8bf64831d0dad2f63a1` with a clean detached review checkout. R01 was withdrawn: existing App normalization already aligned browser task IDs; host normalization is now explicit. R02-R09 are fixed; no new verified source blocker was found in the bounded review. CI [37192695341](https://github.com/kazi-org/wazi/actions/runs/37192695341) passed at that head. Full E3 acceptance remains blocked by missing qualified Serenity owner reader/lineage APIs and Chrome `ERR_BLOCKED_BY_CLIENT`. T3.8, merge and landed verification are unrun; PR #3 remains draft. Owned preview stopped; unmerged SSD worktrees are retained for resumption. No provider request, release or deployment occurred.

## Headless acceptance and authorized candidate merge

The user explicitly requested headless review and merge after being told rich Serenity owner integration was absent. This changes candidate delivery authority; it does not complete the missing owner capability or full E3 acceptance. Headless `agent-browser` verified CI-built source `3d62e48e5c3aa5c468a5b1642ee09aad455042a7` against neutral SSD fixtures, with provider disabled. CI run [37194261045](https://github.com/kazi-org/wazi/actions/runs/37194261045) passed. The workflow now retains built frontend assets for three days so browser and landed checks do not require a heavy local build under excessive shared load.

Observed PASS: discovery and project/task selection; explained proposals; confirm and persisted current sidecar; dismissal; read-only source preview; manual link and reverse navigation across split plans; refresh/selection isolation; all five unavailable context sections; explicit unavailable-context and disabled-provider request messages; icon-only collapsed navigation and restored panels; desktop 1440px and narrow 390px layouts with body width equal to viewport; unsupported `rereview` visible in Other; authored Marked done remains distinct from execution evidence. No uncaught browser errors were reported. Expected request refusals returned 409. No analysis receipts or AI answers were created; no live provider was enabled. Screenshots remain private external artifacts. Rich context, actual owner lineage, live model and memory-derived persistence acceptance remain unverified/incomplete.

Independent delta review passed exact `3d62e48`; final documentation receipt will receive exact-head review before guarded GitHub rebase merge. Candidate merge and landed receipts are recorded in PR #3 and the established local project channel; original full-E3 tasks remain open for the missing owner capability.

## Desktop planning artifact — 2026-10-04

Reconciled clean local/remote main e667bfc7, preserved E1-E7 history and frozen contract. Read actual CLI/resource and HTTP origin/session seams; no fresh code graph available. New E8 records15 dependency-linked Mac tasks and6 planned use cases; E9 contains exactly one deferred Wails planning task after T8.14. ADR0003 records user's platform decision, packaged Node choice and local-only delivery authority. Planning profiles baseline/delivery/go qualified installed tools only; no Swift/Go builds, live AI/Serenity calls or application implementation were run. Structural validation and independent planning review receipts follow through the planning PR and local project channel.

## E8 native implementation candidate -- 2026-10-04

Implemented the Mac arm64 AppKit/WKWebView bundle, owned Go supervisor protocol, explicit packaged parser runtime, restricted window bridge, ephemeral web data, native importer/recovery and desktop controls. Link confirmation is now reversible through the existing versioned sidecar CAS boundary. The runtime source is self-contained; the configured Three.js checkout and portable contract files are unchanged.

Hosted candidate c212add34c1d858e8baf3792c9bc0a58b13c0e6a passed native run [37237316157](https://github.com/kazi-org/wazi/actions/runs/37237316157) and observatory/contract run [37237316193](https://github.com/kazi-org/wazi/actions/runs/37237316193). Native checks include arm64 resource hashes, strict deep ad hoc signature verification, exact readiness/origin/action/drag-boundary helper checks and supervisor invalid nonce/protocol/origin, exited child, startup timeout and explicit EOF stop. Regression checks include21frontend/parser tests,60portable fixtures, Go tests/race/vet and parser-cancellation/revoke preservation cases. Hosted checks do not establish native pointer or VoiceOver acceptance.

Installed candidate artifacts in fresh external-SSD directories; local verification of resource hashes/signature and native self-tests passed. Actual WKWebView on arm64 macOS26.6.2 rendered the Three.js observatory. Earlier6e9df0e artifact verified minimal system PATH/unrelated cwd with bundled Node, neutral project discovery/selection, authored marked-done versus qualified-completion distinction, Other-stage visibility, local proposals/source preview/link confirmation and honestly unavailable Serenity sections. Its native importer opened NSOpenPanel. A positively identified app SIGTERM stopped the owned host through stdin EOF; both PIDs were absent afterward. Seven authored fixture files remained byte-for-byte unchanged, excluding authorized sidecars. No AI/provider/Serenity call was made.

Pure-borderless pointer automation failed to locate a native window; the public hidden-titlebar/full-size-content fallback c212add rendered without native title text or standard buttons. Subsequent CUA actions reported concurrent desktop changes. Installed link reversal, drag/orbit/resize/fullscreen, keyboard/minimize/quit, complete importer cancellation and crash/content recovery are still open native acceptance gates. These observations are not upgraded to pass or landed evidence. Deployment target14.0 does not qualify macOS14runtime or Intel. The local operating handoff is in `docs/desktop/mac-development.md`.

## E8 owner acceptance and recovery correction -- 2026-10-04

The owner reported that the installed native checklist works: dedicated drag versus canvas orbit, resize, minimize/restore, fullscreen, text/paste, importer cancellation, Cmd-Q and reopen. Installed sidecar reversal also passed. AX labels and keyboard behavior are qualified locally; the owner explicitly accepted spoken VoiceOver testing as a follow-up. Spoken output remains unverified.

Actual host-crash testing exposed E8-R03: native recovery lost its controller and weak button target. A retained recovery controller fixes the defect; native action regression checks cover replacement and teardown. Installed signed candidate3bc8e18 on arm64 macOS26.6.2 remained open after its positively identified owned host was killed, cleared old web content, and did not restart automatically. Clicking Restart created exactly one replacement owned host and restored neutral-project discovery. This is actual crash/restart evidence. The new WebKit termination regression injects the public delegate callback into an owned unshown view, and verifies stale-view clearing and actionable recovery; no ambiguous WebKit process is killed.

The owner requested the application icon match the logo. The native bundle now includes the existing Phosphor Planet duotone geometry in Wazi lavender over the sidebar background, with source SVG, MIT attribution and an ICNS resource selected by Info.plist. The offline bundle verifier requires the icon and checks its manifest hash. No provider calls, private screenshots or fixture snapshots enter the repository.

Corrected source467d5ce passed native [37254766343](https://github.com/kazi-org/wazi/actions/runs/37254766343) and regression [37254766353](https://github.com/kazi-org/wazi/actions/runs/37254766353). New callback tests assert Wazi ownership/detachment and actionable recovery; they deliberately avoid requiring synchronous destruction of SDK-retained objects. Independent full-source review atc66fbd4 and affected correction review at467d5ce found no blocking source defect. The rendered ICNS visually matches the existing planet mark. T8.13/T8.14 remain open until actual merge/landed checks.

## E8 landed local Mac delivery

PR5 rebase-landed67d4a5e after exacta373bef independent PASS and green current CI; reviewed/landed trees match. Main native37255227113 and regression37255227007 passed. Installed exact-main artifact manifest67d4a5e verifies resource/icon hashes, arm64 binaries and strict deep ad hoc signatures. Actual landed WKWebView qualified neutral discovery/selection/task navigation, explained proposals, unavailable context, persisted confirm/revoke returning0bindings and Cmd-Q stopping both owned app/host. An old alpha sidecar from a different private identity store was honestly rejected; it was preserved, with valid beta used for the golden path. Minimal PATH and unrelated cwd worked.

Icon candidate467d5ce also passed actual host-crash/no-auto-restart, explicit Restart and recovery Quit; its runtime inputs match landed main. Source plans/code, Three.js and frozen contract remain unchanged. Spoken VoiceOver is an owner-approved follow-up. No live providers, public release, deployment, Intel or older-runtime qualification. This documentation-only closeout preserves implementation provenance and records E8 27/27.

## Owner-selected aperture logo -- 2026-10-04

The selected PNG is preserved byte-for-byte as the shared header/icon source; SHA2561614059c486c0c7df6e8840a662b02c30f3ad331815c2dbbc816e5939b0ab0eb. ICNS conversion changes size/format only; the bundle now records generated-image provenance, keeping Phosphor notice for other UI icons. Candidatea77259c passed native37259831484 and regression37259831463. Actual installed WKWebView shows the aperture in collapsed navigation and expanded fullscreen header with readable wordmark and preserved3D space. First expanded snapshots retained narrow transition geometry until native resize; this is recorded as capture evidence, not a proven logo regression. Preliminary independent source reviewPASS; merge/landed gates remain. Shared local build lease was held by another project and preserved; hosted CI supplied build evidence.

## Aperture identity landed verification

PR7 guarded rebase landed360141ec71983f4db11068a8609357c4585d2084 after exact015ecac independent PASS and current CI. Reviewed/landed trees match. Main native37260736461 and regression37260736435 passed. Installed exact-main manifest, resource/icon hashes, signatures and arm64 checks pass; bundled header PNG is byte-identical to the selected source. Actual main-built WKWebView shows the aperture and existing sample3D/collapsible navigation. E10 is complete. No provider calls, dependency changes, private evidence commits, release or deployment.

## E11 repair planning -- 2026-10-04

The requested command is top-level `wazi repair`. The executable future plan has 23 unchecked tasks. Deterministic preview, explicit private candidate saving and source-digest guarded apply with backup can land independently of optional AI. Preflight pins the current Markdown authoring profile; missing semantics are diagnosed rather than invented. Independent planning findings RPR-P01/P02 were accepted and corrected, removing routine owner gating and mandatory provider dependencies from the deterministic path. Configuration discovery checked variable-name presence only. No CLI implementation, source-plan rewrite or provider call occurred.

## E11 deterministic repair verification

Top-level CLI is wired before unchanged host flags. The ordinary Markdown profile preserves authored meaning and rejects ambiguous identity/metadata. Explicit candidates stay outside source repositories or standalone source folders. Apply is source-locked, digest/inode checked, backed up privately, and atomically replaced; non-cooperating editor race limits are disclosed. New private directories parent-sync before use; post-rename sync uncertainty returns the exact backup for reconciliation.

Initial integrated and hosted contract checks exposed PROFILE-D01 false metadata detection; accepted correction and affected regression passed. STORE-D01/D02 durability/uncertainty and PROFILE-D02 canonical identity corrections are recorded in E11. Full Go race/vet passed at34b6a96; final affected race/vet at9207cf2 passed, alongside contract60fixtures, Node22tests and frontend build. CLI subprocess tests preserve Mac host handshake/parent-watch behavior. Actual neutral CLI dogfood qualifies no-write preview, save, stale refusal, exact apply/backup, CRLF and spaces, disabled AI and sentinel privacy. No changed UI interactions require browser acceptance; frontend runtime source is unchanged. Optional AI/interactive modes are visibly unavailable, with no dotenv read or provider request. CI, independent review, merge and landed gates remain pending.

Independent original review c035fb4/baseccbe1d6 found only STORE-R01 cross-store locking; FAIL retained. Accepted fix a4839bf uses a source-inode flock and stale waiter inode check. Forced real-process overlap for distinct and identical candidates across separate stores passes: one succeeds, one stale. Affected race/vet and CLI/host checks pass at28287a8. Fresh exact-head review/current CI gate merge.

## E11 deterministic repair landed

PR9 independently re-reviewed c64e047/baseccbe1d6 PASS after STORE-R01 fix, then guarded rebase landed48f217cabe9ec8915a7b77eb099e682c53b54bb9; trees match. Main contract/observatory37266279479 and native37266279469 pass. Local exact-main executable reports vcs.revision48f217c and vcs.modified=false. Actual neutral preview/save/stale/apply/exact-backup/disabled-AI, legacy desktop handshake and parent-watch shutdown pass. Fresh local PATH symlink exposes `wazi` through its SSD executable; no existing command was replaced and no installer/release was added. Exact-main native artifact11326840694 verifies manifest/resources/signatures/arm64. Actual packaged WKWebView renders the existing sample observatory; CUA Quit stops both positively owned app/host. No private screenshots committed. E11 deterministic22/32 complete, optional AI10tasks blocked at unqualified T11.4. Interactive metadata proposal mode is unavailable; authored gaps require explicit source edits. Frozen contract and Three checkout preserved. No provider call or production deployment.

## E12 installation and CLI report

Owner reports CMD+K cannot find the installed app and bare CLI reports missing frontend assets. Installed bundle has Wazi name/APPL type/icon and strict signature; importer test recognizes application type and returns34attributes. Persisted Spotlight metadata remains absent; system/data index reports read-only. Launcher identity requested, no global settings/reset performed. Bare CLI falls through to development host startup; independent no-argument help correction assigned, preserving explicit host/Mac arguments and repair behavior. Repair default is preview; only explicit source-bound candidate apply changes the original with exact backup.

E12 CLI verification passed at `fa1b172380d66fbc5da02a184dce89530b59ae8b`: changed-package race tests (including existing desktop/repair subprocess tests), vet, build, unrelated-cwd no-argument and repair-help checks. Shared load initially held the gate; verification began below the limit with the lease and SSD caches. Independent review identified stale pre-verification status in the plan; that record was corrected without changing tested source.

E12 CLI delivery: PR #11 rebase-merged to `503969029b93df0ea5d087f3a270c87eded6be51`, equal tree to independently reviewed `fa05e5b`. All three hosted checks passed. The clean exact-landed binary replaced only the verified owned local PATH symlink atomically and passed no-argument/repair-help checks from `/`. The installed app/logo is preserved. Search qualification remains open pending actual launcher identification; no global indexing or registration reset was performed.

## E13 AI repair and search delivery — 2026-10-06

Baseline c8afe78 was clean. Separate SSD Luna lanes implemented provider and receipts while coordinator integrated CLI and documented authoritative provider sources. No live AI requests, credentials sent to workers or provider changes. The owner-provided dotenv permissions were tightened to 0600 for safe operation-scoped loading. Mac bundle/importer/registration are valid; per-app refresh/import did not yield search metadata. Owner administrator mdutil status confirms read-only Data index, with enable-index action pending explicit whole-volume scope approval.

Initial PR13 hosted regression37475260995 failed adapter expectation/combined-spacing cases and an absolute temporary-directory test helper; frontend and native checks passed. These actual test failures block delivery until corrected and verified. New source validation was also tightened to preserve HTML comment closing lines.

Corrected candidate b6621e5 has tree-equal clean CI checkout b9c88320. Hosted37476046507 passes full Go race/vet/contract60 and frontend checks; local Node22 passes. Its Darwin host and verifier were exercised on SSD with no real provider: noargs/help, no-write preview, explicit save/stale/apply/exact backup, credential-source refusal, unsafe endpoint rejection and all60 fixtures pass. Local Go builds/tests remain unavailable due shared load above10; native packaging check pending.

2026-10-06 E13 review loop: independent96e8000 review failed with four accepted findings, integrated fixes at dcf1a89. Added real filesystem receipt-failure/no-resend regression, canonical selected-source dotenv boundary and caller-checkout exception, status-preserving whitespace tests and zero-dispatch malformed-source preflight. Formatting/diff checks pass; new hosted checks and exact-head review pending. Local Go held by load above10. No live AI request.

2026-10-06 E13 final sourcebf71d42: independent review PASS no findings; hosted37479695890 full Go race/vet, 60 frozen fixtures and frontend PASS; native37479696073 PASS. Syntheticb5157bb tree equality verified. CI-built Darwin CLI help and inadmissible-source no-request/no-cache checks PASS, verifier60/60. No live calls. Local full Go remains unrun under shared load; worker focused request test passed. Search remains blocked by administrator-confirmed read-only data index.

2026-10-06 AI repair landed: PR13 rebase main633c148 equals reviewedfc65270. Pre-merge checks37480081504/37480081495 and main37480417986/37480417760 PASS. Exact-main clean Darwin CLI installed through the owned PATH symlink; help and ambiguous/missing-meaning no-request/no-cache checks and frozen60 PASS. App signature/logo unchanged and valid. Shared lease held by another project; local full build/tests not run. Search remains open under T13.1a/E12; no global index changes, live provider call, release or deployment.

2026-10-06 Mac search acceptance: owner enabled indexing; owner and agent readback report Indexing enabled. mdls returns Wazi/org.kazi.wazi; bundle-ID mdfind returns installed app. Owner confirmed successful app discovery in the requested search. E12 7/7 and E13 10/10; prior read-only blocker is historical. Logo/signature remain verified; no new screenshot, index erase/rebuild or app resource change. Mac-specific documentation-only closeout uses the native-project exemption; no local build/test suite or new coding worker.

2026-10-06 E14 source69afa2b: DGX Node22 npm test passes22/22 and Vite production build passes with exact Three1ea31f source; Mac dependency checkout untouched. Real Chrome CUA verification on neutral sample: wheel zoom, orbit, canvas right-pan, Map, selected-task focus and reset preserve each nonempty header/top-card center alignment within0.001px and positive clearance. Narrow390x844 has body width390 and rendered heading font9px at far/reset distance; temporary viewport override reset. Initial inspector parseFloat failure was corrected in the read-only inspection, not an app error; final browser error log empty. No-WebGL fallback remains source-reviewed rather than freshly forced in a browser. Existing dense-card crowding during narrow resize is unchanged; Reset restores fitted overview. Source review pending; no Mac build/test suite, provider call, release or deployment.

E14 review loop: independent review of c5abb9a identified accepted P2 LH-R01, per-label measurement/write interleaving can force repeated synchronous layouts every frame. Merge held; UI worker batches all geometry reads before card/header writes, then affected checks and exact-head re-review. No profiling result is claimed.

2026-10-06 E14 LH-R01 correction2231a5f: all frame geometry reads precede style writes. DGX22/22 frontend tests and production build PASS. Real Chrome wheel zoom, orbit, actual canvas right-pan, Map, focus and reset retain header/top-card centers within0.001px and positive clearance; narrow390x844 reset retains9px headings and exact center alignment. Viewport restored. No performance profiling claim; exact-head independent re-review pending.

2026-10-06 E14 landed/installed: PR16 guarded REBASE fb78218 equals reviewed6cb8278 tree. Both landed workflows37566505535/37566505537 PASS. Downloaded exact-main Mac bundle sourceRevisionfb78218; verified arm64 resource hashes/ad-hoc signature before/after atomic installation, prior app retained. Native CUA launch shows expected headers/logo/sample and local discovery. Native automated wheel/focus gestures did not change the camera even after raising the window; do not claim native zoom acceptance. Browser camera coverage passed; T14.3 retains this outstanding check. No Mac builds/test suites, provider calls, signing-key changes, release/deploy.
