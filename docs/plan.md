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
