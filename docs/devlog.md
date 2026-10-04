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
