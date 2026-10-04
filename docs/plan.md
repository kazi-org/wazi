# Wazi — plan observatory prototype

## E1 — Explore project plans in space

- [x] T1.0 Qualify reference, plan schema and local environment  Owner: coordinator  kind: agent stage: preflight  acc: [reference inspected; external SSD writable; local Three.js source available]
- [x] T1.1 Build read-only local plan discovery and parser  Owner: plan-reader  kind: agent stage: implement  blocked-by: [T1.0]  acc: [skill-format tasks, statuses and dependencies parse; source plans remain unchanged]
- [x] T1.2 Build interactive 3D observatory and task inspector  Owner: coordinator  kind: agent stage: implement  blocked-by: [T1.0]  acc: [project selection, orbit, zoom, task selection, filters and Markdown import work]
- [x] T1.3 Verify parser, build, browser interactions and design  Owner: coordinator  kind: agent stage: verify  blocked-by: [T1.1, T1.2]  acc: [targeted tests pass; browser inspected; design QA passed]
- [x] T1.4 Independently review integrated candidate  Owner: reviewer  kind: agent stage: review  blocked-by: [T1.3]  acc: [exact candidate SHA reviewed; blocking findings resolved]
- [x] T1.5 Integrate reviewed candidate into local main  Owner: coordinator  kind: agent stage: merge  blocked-by: [T1.4]  acc: [reviewed candidate on local main; no remote publication]
- [x] T1.6 Verify local landed tree and leave preview open  Owner: coordinator  kind: agent stage: verify-landed  blocked-by: [T1.5]  acc: [main matches candidate; local preview serves accepted interface]

Acceptance evidence: ten parser tests and production build passed; browser/design evidence in `design-qa.md`; independent exact-head review in `docs/review.md`; local main integrated by fast-forward, source equality verified, and preview restarted from the original workspace. No remote publication.

## E2 — Reverse-engineer implementation documentation and coordinate sessions

- [x] T2.0 Inspect current implementation, records and peer channel  Owner: docs-coordinator  kind: agent stage: preflight  acc: [baseline captured; source and existing docs read; peer coordination message appended]
- [x] T2.1 Document architecture and actual data flow  Owner: architecture-writer  kind: agent stage: implement  blocked-by: [T2.0]  delivers: [docs/architecture.md, docs/data-flow.md]  acc: [module boundaries and runtime contracts match source]
- [x] T2.2 Document as-built RFC, design notes and local operation  Owner: docs-coordinator  kind: agent stage: implement  blocked-by: [T2.0]  delivers: [docs/rfc/0001-local-plan-observatory.md, docs/operations.md, docs index]  acc: [implemented behavior, proposals and historical evidence clearly distinguished]
- [x] T2.3 Verify and independently review the documentation  Owner: docs-reviewer  kind: agent stage: review  blocked-by: [T2.1, T2.2]  acc: [relative links resolve; claims checked against baseline source; blocking factual findings resolved]
- [x] T2.4 Integrate documentation and post peer handoff  Owner: docs-coordinator  kind: agent stage: merge  blocked-by: [T2.3]  acc: [docs integrated without runtime changes; channel handoff appended and reply state reported]
- [x] T2.5 Verify landed documentation and channel state  Owner: docs-coordinator  kind: agent stage: verify-landed  blocked-by: [T2.4]  acc: [landed files and runtime equality confirmed; peer messages reread]

## E3 — Implement the settled observatory (future outline)

fidelity: outline

User-selected product scope is recorded in RFC 0002: Go core with the preserved Three.js interface, local plan discovery and selected-project code analysis, declared/confirmed file-test bindings, automatic local suggestions, explicit deeper GPT Luna requests with persistent reuse, and rich shared-brain project-scoped read-only Serenity context from the first version, also included in requested AI inputs under a lifecycle-aware persistence gate. All material user choices are resolved in E4; exact-digest concurrence and independent review passed; local record landed and verified; runtime implementation remains unstarted. No implementation lane is assigned; interface qualification is future work, not evidence of availability.

- [ ] T3.0 PLAN: expand the settled design when implementation is requested  Owner: TBD  kind: plan  deps: [T4.5]  delivers: [executable Go/suggestion/cache/context delivery plan]  acc: [implementation explicitly requested; RFC/ADR gates translated into bounded tasks; Go/parser compatibility and Serenity owner contracts qualified; verification/review/merge gates added before coding]

E2 acceptance evidence: documentation review and source baseline in docs/review.md; all local links resolve; docs-only fast-forward integration; runtime/config/package/test files unchanged from 8c5486d; peer handoff received and acknowledgement/proposal returned through ajent.social. Follow-up interface feedback is pending, not required to describe the current implementation.

## E4 — Settle the design with the peer, record it, then stop

- [x] T4.0 Reconcile baseline, channel and direct user scope  Owner: design-coordinator  kind: agent stage: preflight  acc: [baseline preserved; peer replies read; direct product choices distinguished from recommendations]
- [x] T4.1 Resolve material choices and record the versioned design  Owner: design-coordinator  kind: agent stage: implement  blocked-by: [T4.0]  delivers: [RFC 0002, ADR 0001, design notes]  acc: [final Go/UI, Serenity content and AI input choices resolved; identity/provider/cache/eligibility contracts explicit; no runtime implementation]
- [x] T4.2 Verify documentation and source preservation  Owner: design-coordinator  kind: agent stage: verify  blocked-by: [T4.1]  acc: [local Markdown links and whitespace clean; runtime/source plans unchanged]
- [x] T4.3 Obtain peer concurrence and independent contract review  Owner: design-reviewer  kind: agent stage: review  blocked-by: [T4.2]  acc: [peer agrees frozen digest/revision; independent review names candidate and resolves material findings]
- [x] T4.4 Integrate the settled design into local main  Owner: design-coordinator  kind: agent stage: merge  blocked-by: [T4.3]  acc: [reviewed documents integrated without losing concurrent work; no remote publication]
- [x] T4.5 Verify landed record, post closure and stop runtime work  Owner: design-coordinator  kind: agent stage: verify-landed  blocked-by: [T4.4]  acc: [landed RFC/ADR/design/plan agree; channel settlement recorded; implementation remains unstarted; later shared plan-skill discussion continues separately until complete]

E4 design/review evidence: all direct user choices resolved; peer `SERENITY-WAZI-FINAL-CONCURRENCE-v1-REFINE-20261004` and independent reviewer `/root/review_observatory` concur at exact semantic candidate `53f830d9ae31beb9651a8876e8f7d285eca7eb5a`. Cache privacy/discovery/capacity/offline findings resolved; no runtime qualification claimed. Shared plan-contract discussion is separate and does not assign schema implementation.

E4 local landed evidence: accepted record `9d652075650c4e04c0f7432b0a262be79aa532ec` fast-forwarded into local main; clean tree, 86 local links resolved and all changed paths confined to docs. Runtime/config/package/test files remain identical to pre-settlement `6b63dfd`. No remote publication or provider/brain/backend invocation. Runtime work stops; shared plan-skill discussion remains a separate authorized watch until its closing receipt.

## E5 — Participate in the shared portable-plan discussion

- [x] T5.0 Introduce, contribute, review the synthesis and watch through explicit closure  Owner: design-coordinator  kind: agent stage: verify  delivers: [source-verified Wazi requirements and closed-synthesis reference]  acc: [participation recorded; retained text digest verified; DISCUSSION-CLOSED-01 observed; recommendation/ratification/implementation boundaries preserved]

Evidence: shared-skills discussion record `docs/tasks/portable-plan-contract-discussion-20261004.md`, SHA256 `6e17965153a8e8967372c52c64f935872737543e7227282facb6a86e62e2b69c`, local commit `2564160bd7b8a51ea8801d70f0ad57aeab6ec1a5`. Wazi shared-schema stewardship remains a user-owned OPEN decision, not an implementation assignment. Runtime work and both requested watches have stopped; no further schema or consumer work is started.

## E6 — Collapsible side panels for a larger 3D canvas

- [x] T6.0 Qualify current panel layout and isolate the UI lane  Owner: panel-coordinator  kind: agent stage: preflight  acc: [baseline captured; SSD writable; App/CSS/scene resizing inspected; peer channel read]
- [x] T6.1 Add independent panel controls and icon-only minimal navigation  Owner: panel-worker  kind: agent stage: implement  blocked-by: [T6.0]  acc: [left icons retain labels/tooltips; right collapse preserves selection; canvas reclaims width; narrow layout remains usable]
- [x] T6.2 Verify build and desktop/narrow browser interactions  Owner: panel-coordinator  kind: agent stage: verify  blocked-by: [T6.1]  acc: [panel states and canvas resize checked; controls keyboard accessible; no overflow/new browser errors]
- [x] T6.3 Independently review the exact panel candidate  Owner: panel-reviewer  kind: agent stage: review  blocked-by: [T6.2]  acc: [base/head and findings recorded; blocking issues resolved]
- [x] T6.4 Integrate reviewed UI into local main  Owner: panel-coordinator  kind: agent stage: merge  blocked-by: [T6.3]  acc: [reviewed changes land without losing concurrent work; no remote publication]
- [x] T6.5 Verify landed preview and records  Owner: panel-coordinator  kind: agent stage: verify-landed  blocked-by: [T6.4]  acc: [local main matches candidate; served UI shows collapsible panels; outcome recorded]

## E7 — PC-WAZI experimental portable plan contract

Scope: direct ship request authorizes the PC-WAZI dispatch: headless experimental v0 schemas, Go structural/semantic validator, neutral fixtures, immutable revision/digest and offline consumption, authoritative Markdown/native-source mapping, and Wazi display compatibility. E3 broader observatory/Serenity/OpenRouter runtime remains an outline. Ordinary unenrolled delivery; no new scheduler or native service authority. Public artifacts must contain only neutral examples and portable paths. Historical E4/E5 stewardship-open wording records the earlier discussion; the later dispatch and direct ship request assign this bounded stewardship.

Bindings/preflight: local shell/Git/gh, Go 1.26.1, Node/Vite and real Chrome available; Kazi CLI available for bounded engineering execution; relevant ship/plan/apply, Go and team skills loaded through local packages. Optional capability profile catalog is unavailable (catalog/skills.json missing); use qualified current-session CLI tools without global plugin changes. No code graph exists, so manual parser/scanner/UI/API discovery applies. SSD mounted/writable with ample space; caches/artifacts stay there. Main clean at bdf61d0; existing worktrees preserved. GitHub repository is public and empty, with no default branch or CI/protections observable yet: initial-base publication is a delivery prerequisite, to be resolved on a concrete reviewed candidate. Heavy build/test execution waits for one-minute load <=10 and an owned shared build lease for multi-package Go commands.

Use cases: UC-PC1 consume/validate a pinned bundle offline; UC-PC2 preserve one source authority, task identity, metadata and acceptance across normalization; UC-PC3 reject incoherent dependency/evidence/compound mappings without claiming receipt authentication; UC-PC4 explore authored status and unsupported stages honestly in Wazi.

- [x] T7.0 Qualify dispatch, sources, ownership and delivery prerequisites  Owner: contract-coordinator kind: agent stage: preflight acc: [dispatch/source inventory recorded; isolated SSD lanes allocated; tools/access/load and empty remote evaluated]
- [x] T7.0.1 Independently audit clean initial source baseline Owner: independent-reviewer kind: agent stage: review blocked-by: [T7.0] acc: [root snapshot privacy/source equality audited; local history and channel excluded]
- [x] T7.0.2 Initialize empty GitHub base from audited snapshot Owner: contract-coordinator kind: agent stage: merge blocked-by: [T7.0.1] acc: [only reviewed root snapshot published; default main fetched; no existing remote work overwritten]
- [x] T7.1 Specify candidate schemas, semantics and neutral conformance cases  Owner: contract-coordinator kind: agent stage: implement blocked-by: [T7.0] verifies: [UC-PC1, UC-PC2, UC-PC3] acc: [separate definition/snapshot/evidence shapes; strict extension rules; stages and typed predicates; canonical compound ownership; revision-bound evidence; source mapping and trust limits explicit]
- [x] T7.1.1 Resolve candidate audit CT001/CT002 and consumer safety cases Owner: contract-coordinator kind: agent stage: implement blocked-by: [T7.1] acc: [manifest unique path set; logical evaluation key unique; portable-reference safety and nonsuccess fixtures added]
- [x] T7.1.2 Verify refined schemas and neutral fixtures Owner: contract-coordinator kind: agent stage: verify blocked-by: [T7.1.1] acc: [schema compiles offline; valid fixtures pass structure; manifest digest and coverage recomputed]
- [x] T7.2 Resolve consumer counterexamples and freeze experimental revision  Owner: contract-coordinator kind: agent stage: verify blocked-by: [T7.1.2] acc: [candidate posted to shared channel; concrete consumer findings disposition recorded; exact immutable contract digest published before adapters start]
- [x] T7.3 Implement Go library/CLI and conformance tests  Owner: validator-worker kind: agent stage: implement blocked-by: [T7.2] verifies: [UC-PC1, UC-PC3] acc: [offline structural and semantic validation; version/digest pin checked; qualified identity/reference/cycle/source/attempt/subject/policy/independence/late-audit/compound cases covered; no authority inferred]
- [x] T7.4 Implement read-only display compatibility and metadata preservation  Owner: display-worker kind: agent stage: implement blocked-by: [T7.2] verifies: [UC-PC2, UC-PC4] acc: [metadata and opaque acceptance retained; unsupported stages visible; authored checkbox separated from qualified completion; pinned JSON display untrusted until independently qualified]
- [x] T7.3.1 Verify Go API/CLI on isolated CI runner Owner: contract-coordinator kind: agent stage: verify blocked-by: [T7.3] acc: [Go tests and60fixture conformance pass exacthead; Macverifier compiled andrunoffline; digestmismatch failsclosed]
- [x] T7.4.1 Fix browser findings D1/D2 Owner: display-worker kind: agent stage: implement blocked-by: [T7.4] acc: [end-lane remainsvisiblewithinspector; explicitreset/Map coherentwithretainedcamera; acceptancevisualnewlines preserved]
- [x] T7.4.2 Verify affected browser resize/reset/criteria paths Owner: contract-coordinator kind: agent stage: verify blocked-by: [T7.4.1] acc: [desktop/narrowimport andinspection work; nodesfitafterresize/reset; nooverflow/newerrors]
- [x] T7.3.2 Fix Go compile finding G1 Owner: validator-worker kind: agent stage: implement blocked-by: [T7.3] acc: [unused local removed; frozen spec unchanged]
- [x] T7.3.3 Verify corrected Go implementation Owner: contract-coordinator kind: agent stage: verify blocked-by: [T7.3.2] acc: [isolated CI rerun passes compilation and conformance]
- [x] T7.5 Verify integrated conformance, required checks and browser behavior  Owner: contract-coordinator kind: agent stage: verify blocked-by: [T7.3.1, T7.3.3, T7.4.2] acc: [Go/Node conformance fixtures and targeted tests pass; production build and browser status/stage/metadata cases pass; neutral public artifacts audited; required-check facts recorded]
- [~] T7.6 Independently review exact integrated candidate  Owner: independent-reviewer kind: agent stage: review blocked-by: [T7.5] acc: [all T7.1/T7.3/T7.4 covered; reviewer/base/head/findings recorded; blockers resolved with tracked fix/verification/re-review tasks]
- [x] T7.6.1 Fix current-evidence independence binding G2 Owner: validator-worker kind: agent stage: implement blocked-by: [T7.5] acc: [unrelated candidate review cannot lend independence; existing audit rejection retained to a current satisfied gate; qualified review carries its own proof]
- [x] T7.6.2 Verify independence fix and integrated regressions Owner: contract-coordinator kind: agent stage: verify blocked-by: [T7.6.1] acc: [targeted wrong-subject and mixed-proof regressions and60frozen cases pass CI; offline probe rejects missing current proof]
- [ ] T7.6.3 Independently re-review corrected exact candidate Owner: independent-reviewer kind: agent stage: rereview blocked-by: [T7.6, T7.6.2] acc: [G2 closed at exact corrected head; all remaining findings disposition recorded]
- [ ] T7.7 Publish initial base and merge reviewed PR by GitHub rebase  Owner: contract-coordinator kind: agent stage: merge blocked-by: [T7.6.3] acc: [empty-repository publication prerequisite resolved; exact reviewed head/checks pass; rebase merge receipt observed; no release/deploy]
- [ ] T7.8 Verify landed contract and post consumer handoff  Owner: contract-coordinator kind: agent stage: verify-landed blocked-by: [T7.7] acc: [landed SHA/contract digest and offline fixtures verified; PR/check/review facts posted; remaining runtime trust gaps explicit; no other repository edited]

T7.2 evidence: candidate source audit PASS at127c393; CT001/CT002 fixed; PC-PLAN/Serenity/Foundry constraints disposition recorded. Frozen0.0.1 digest `sha256:7582512f122d2f2a9c4461facc7541c9887053f137260d6ebe9c6dea611d039d` covers66 unique schemas/semantics/catalog/fixture paths; Go runtime qualification pending. Single-file Markdown bound and out-of-band missing stage/acceptance/split-source diagnostics explicit. Kazi CLI has no qualified Codex harness, so bounded in-session GPT-6-Luna implementation fallback is used; no alternate harness/model/provider activated.

Initial base publication: independent source/privacy audit PASS at `adc4e8d68d11692d4089c7680fb363482f824e3e`; that clean root snapshot alone initialized public main. Original local history/channel remain unpublished. Heavy local checks remain load/lease gated; isolated hosted Linux CI can run Go/Node checks and build a disposable Mac offline verifier without consuming the shared Mac build lane. This is CI verification, not release publication.

T7.5 evidence: isolated CI run37184251198 passed exact public head e7ed7b8 (Go tests, all60 conformance cases, Node15 tests and production build). Its disposable Darwin arm64 verifier passed version, fixtures, valid input, invalid independence and digest-mismatch commands offline with proxies unreachable; authentication stays false. Browser passed desktop resize/reset/Map fitting,390px no overflow, opaque acceptance newlines and metadata retained, unverified reports separated from authored0%. No browser console errors. Repository protection API returned404 and rulesets are empty; both scoped CI jobs are still delivery-required. Final independent review and rebase/landed checks remain pending.

G2 corrected verification: public8693d3d / CI37184831031 passes all Go regressions,60fixtures,Node15 and build. Offline Darwin verifier f26d7731a0b4f9662dedb7d007ee375236c7dc8f7f3cff9393cf9d0aab59695e rejects the independent wrong-head/base proof probe with independence_missing/exit2 (priorbinary exit0); valid own-proof and digest tests pass. Final exact-head independent re-review remains required.
