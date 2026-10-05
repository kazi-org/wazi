# E11 — `wazi repair`: safe plan source repair

Acceptance: `wazi repair` can produce a reviewable repair candidate for one selected local Markdown plan and write it only after an explicit apply request; repair preserves authored intent and fails closed on semantic ambiguity; optional EXPLABS backend requests occur only under explicit AI opt-in, with no automatic retry after an uncertain request. This plan covers future CLI implementation. Delivery of this planning artifact is a separate review/merge/landed operation.

fidelity: executable

**Change Summary — 2026-10-04:** Created an executable future-delivery plan for the `wazi repair` CLI, including schema/provider decision gates, one-file preview/apply safety, private AI proposal handling and complete delivery stages. No implementation is authorized by this planning artifact.

## Context

Wazi discovers and reads `plan.md`, `docs/plan.md`, and direct Markdown files under `docs/plans/`. Its browser-safe parser in `src/plan-parser.mjs` preserves task source blocks and authored checkbox status in a display projection; it does not write files or enforce a complete authoring schema. The Go host in `cmd/wazi` currently starts the observatory from flags and the separate `cmd/wazi-contract` validates portable JSON bundles. Neither implements a `wazi repair` command.

The request to “conform with the authorized plan schema” has a material unresolved target. This plan proposes a conservative default: repair only ordinary Wazi-supported plan Markdown using the current shared plan-authoring metadata guidance and parser behavior. This is not a newly versioned or formally approved schema. Keep frozen portable contract0.0.1 and its `contracts/plan/v0` bytes unchanged. `cmd/wazi-contract` validates the frozen portable JSON bundle. Shared `$plan` guidance describes a read-only Markdown-to-JSON normalizer, but no such normalizer source is present in this repository at the planning base; verify its actual availability before involving it. Neither that adapter concept nor portable JSON makes the Markdown source a writable JSON master.

Before implementation, the owner must confirm the target dialect. If a formal schema or portable JSON target was intended, stop at T11.0 and record that scope separately. Proposed safe default: accept a single selected Markdown source file, preserve all authored semantics and unknown text, change only unambiguous syntax, and report missing or conflicting meaning without inventing it.

Open decisions to resolve at T11.0:

- **D11-SCHEMA:** Which authority defines “authorized plan schema”? Proposed default is current Wazi Markdown parser behavior plus shared plan-authoring metadata rules, with no newly invented version and no modification of portable contract0.0.1.
- **D11-EXPLABS:** What exact provider API/authentication/model/retention contract do the requested variables configure? No OpenAI-compatible request/response or structured-output capability is assumed. Proposed default is deterministic-only until authoritative provider documentation is supplied; AI stays disabled.

Configuration discovery: the coordinator verified presence of the three requested variable names in the owner-provided primary-checkout `.env`, without reading or reporting values. This confirms configuration presence only, not credential validity, endpoint authentication, API compatibility, model capability or provider terms. Future `wazi repair` must load dotenv only for that repair invocation, default its explicit `--env-file` path to the primary Wazi project working directory's `.env`, never the selected plan's `.env`, and let process environment values override file entries. The host/desktop launch path must not load dotenv; the file stays uncommitted and out of bundles/artifacts.

Objectives:

- Add the exact top-level command `wazi repair`, without changing the existing host launch or Mac supervisor flag invocation.
- Build the command as the `wazi` executable from `cmd/wazi`; document a local build/use path, but do not add an installer, publish a release or change the Mac app's private bundled-host invocation.
- Make preview the default. Apply only one reviewed candidate to one original plan after an explicit `--apply` request and source-digest compare-and-swap.
- Keep deterministic repairs useful without a provider. Offer optional AI proposals only with explicit `--ai` and confirmed provider configuration.
- Preserve task/evidence IDs, checkboxes, task text, dependencies, owner/acceptance/stage, provenance and unknown metadata byte-for-byte unless an owner-approved, semantics-preserving transformation is specified at T11.0.
- Keep plan-source writes local and visibly recoverable with an exact original backup. The normal browser and desktop observatory remain read-only.

Non-goals: editing `contracts/plan/v0`; generating missing task IDs, acceptance, owner, dependencies, stage or project facts; changing checkbox/status or unsupported stage tokens; bulk repair of repositories; calling AI from discovery, host startup, preview, validation or apply; reusing E3 analysis answers; automatic retries; Serenity context; plan synchronization, dispatch/admission, release or deployment.

Constraints and measures:

- No secret values were inspected during planning. Future implementation may read explicitly selected credentials in memory to authenticate a repair request, but must never display, log, commit, package or export them.
- Preserve source authority. A repair candidate is only a proposal until the user explicitly applies that exact candidate to the same unchanged file.
- Never infer execution completion or qualification from successful syntax repair.
- Keep frozen portable contract digest `sha256:7582512f122d2f2a9c4461facc7541c9887053f137260d6ebe9c6dea611d039d` unchanged.
- Success is measured by no-write preview, exact semantic-preservation and idempotency tests, safe atomic apply/recovery tests, compatibility tests for the host/Mac launch, and fake-provider tests proving opt-in and redaction. No live provider test is a delivery requirement.

## Discovery Summary

Source inspection at requested base `3bd02f791ff6c00f0fe6f22f0cf2dd5404d8df1e` found:

| Concern | Current source and behavior | Consequence |
| --- | --- | --- |
| Markdown discovery/parser | `scripts/plans.mjs`, `src/plan-parser.mjs`, `tests/plans.test.mjs`. Discovery recognizes the three path forms above. Parser reads checkbox rows, generated `line-N` IDs when no leading ID exists, headings/epics, `Owner`, `stage`, `status`, dependencies and acceptance aliases. It preserves `sourceBlock`, metadata and authored checkbox separately, but is permissive and does not diagnose every duplicate or unsupported construct. | Reuse/compare against these existing rules; do not claim parser success proves schema validity. Pin before/after parser projections and preserve unrecognized source. |
| Current Go host | `cmd/wazi/main.go` runs the loopback observatory using flag-based startup; the packaged Mac supervisor passes explicit host flags. It does not have subcommands. | Dispatch `wazi repair` before existing host flag parsing; verify the desktop invocation remains identical. |
| Portable contract | `contracts/plan/v0/{README.md,SEMANTICS.md,contract.schema.json,plan-definition.schema.json,...}` and 60 neutral fixtures define experimental portable JSON0.0.1. `contracts/plan/v0/manifest.json` pins the frozen digest. `cmd/wazi-contract` has `version`, `validate`, and `fixtures` commands. | This is a separate JSON interchange/validator contract. Do not modify it or label ordinary Markdown repair as 0.0.1 conformance. |
| Portable normalizer guidance | Shared `$plan` skill documentation describes a read-only Markdown-to-JSON adapter, but `git ls-files` and repository path inspection find no `scripts/normalize_plan.py` implementation here. | Do not claim that the adapter is available in Wazi. If portable normalization is requested, qualify the actual adapter separately; `wazi repair` must not masquerade as a source rewrite mode for it. |
| Provider | `internal/deep` implements optional OpenRouter for the E3 `Dig deeper` flow; this repair request specifies `EXPLABS_API_KEY`, `EXPLABS_BASE_URL`, and `EXPLABS_MODEL`. The EXPLABS backend's auth, request/response contract, model capability and data terms are not qualified by source inspection. | Keep provider identity/configuration separate. No OpenAI-compatible protocol, structured output or tool support is assumed. Require owner-supplied authoritative provider docs and fake HTTP contract tests before implementation of the adapter. |
| Secrets | The owner-provided primary-checkout `.env` contains the three requested variable names (coordinator checked presence only; values were not read). `.gitignore` excludes `.env.local` and `.env.*.local`, but not plain `.env`. | Repair-only dotenv loading defaults to the primary Wazi project working directory with explicit `--env-file` override, never selected-project dotenv. Add `.env` ignore/package exclusions and fake-value tests. Presence is not provider qualification. |
| Persistence/cache | RFC0002 governs E3 AI result lineage, exact-input reuse, private capacity/expiry/invalidation, request receipts, in-flight deduplication and uncertain dispatch handling. Repair proposals have different semantics and must use their own operation/schema/source-qualified namespace. | No reuse of `Dig deeper` answers. Persist no plan bodies in logs; private proposal candidates expire and invalidate on source/provider/model/prompt/profile changes. Never automatically resend an uncertain request. |

Use cases:

| ID | Priority | Outcome | Current wiring |
| --- | --- | --- | --- |
| UC-RPR-001 | P1 | Inspect a deterministic repair diff for one Markdown file without changing it. | Planned; parser exists, repair command absent. |
| UC-RPR-002 | P1 | Apply the exact reviewed candidate only to the unchanged selected file, with an exact recoverable backup. | Planned; current observatory/desktop remain read-only. |
| UC-RPR-003 | P2 | Request an optional AI formatting proposal for a selected file when deterministic repair cannot resolve a syntax-only issue. | Planned; provider contract unqualified. |
| UC-RPR-004 | P1 | See precise diagnostics and retain the source unchanged when repair would require semantic guesses, cross-file edits or unsupported schema assumptions. | Planned; existing parser is permissive; portable normalizer already fails closed for unsupported mappings. |

## Scope and Deliverables

In scope: one `wazi repair` invocation; one explicitly selected Markdown source; bounded parse and candidate generation; default no-write preview; explicit candidate apply; backup, source/sidecar locks or equivalent concurrency control, CAS and recovery; optional opt-in EXPLABS backend proposal path after protocol qualification; private configuration/candidate/request storage; CLI, unit, filesystem-safety, fake-provider, integration and compatibility tests; documentation and delivery records.

Out of scope: recursive auto-repair, editing plans from the browser/desktop UI, writing portable JSON into Markdown, any same-version portable-contract change, AI-authoring task meaning or completion, automatic credential discovery from project directories, cloud sync, provider release claims, live paid requests in tests, live Serenity access, publishing/deployment.

| Deliverable | Description | Owner | Acceptance |
| --- | --- | --- | --- |
| D11.1 | Frozen source-dialect and transformation policy | Owner + coordinator | T11.0 records the owner's choice; unresolved behavior remains blocked. |
| D11.2 | Deterministic `wazi repair` preview/apply flow | CLI implementer | One-file scope, exact semantic preservation, digest CAS, private exact backup, fail-closed diagnostics. |
| D11.3 | Optional EXPLABS adapter and private repair-proposal receipts | CLI/provider implementer | Exact env names; documented endpoint contract; default disabled; fake HTTP tests; uncertainty never triggers automatic retry. |
| D11.4 | Integrated test and operating evidence | Verification owner | CLI/host/Mac/portable regressions, secret-safety checks, exact-head review and landed receipt. |

## Checkable Work Breakdown

- [ ] T11.0 Freeze target dialect, transformations, CLI exposure and write policy  Owner: product owner + coordinator  Est: 1h  kind: human  stage: preflight  delivers: [accepted D11.1 and implementation contract]  acc: [owner explicitly chooses ordinary Markdown repair or another separately scoped target; allowed byte/semantic transformations and local `wazi` executable exposure are recorded; repair-only dotenv source/precedence and EXPLABS auth/request/response/model capability/data handling are recorded or AI is explicitly deferred; unsupported or unknowns are blocked; no secret value is recorded]

All following tasks depend on T11.0. If the owner chooses the proposed default, the repair target is the current Wazi Markdown reader plus shared plan-authoring metadata rules, with ordinary Markdown as the only writable authority. The frozen portable JSON contract remains unchanged and read-only. Implementation stops if the selected target requires inventing semantics or a provider contract remains unknown.

- [ ] T11.1 Implement deterministic Markdown diagnostics and semantics-preserving candidate generation  Owner: CLI implementer  Est: 1d  kind: agent stage: implement  blocked-by: [T11.0]  verifies: [UC-RPR-001, UC-RPR-004]  delivers: [internal repair core]  acc: [candidate reparses under pinned existing parser behavior; every authored task ID, checkbox marker, task title, dependencies, stage, owner, acceptance and unknown metadata value is preserved; duplicate/missing IDs, absent semantic fields, unknown fields, split authority and unsupported repairs produce diagnostics without fabricated replacement]
- [ ] S11.1.1 Test deterministic repair matrix, idempotency and parser projection preservation  Owner: CLI implementer  Est: 3h  kind: agent stage: verify  blocked-by: [T11.1]  verifies: [UC-RPR-001, UC-RPR-004]  acc: [fixtures cover valid/malformed metadata, duplicate/missing IDs, unknown annotations, unsupported stages, comments/fences, multiline acceptance, status, split plans and no-safe-fix cases; second repair is byte-identical; original bytes never change]

- [ ] T11.2 Add `wazi repair` CLI dispatch with no-write preview as the default  Owner: CLI implementer  Est: 4h  kind: agent stage: implement  blocked-by: [T11.0, S11.1.1]  verifies: [UC-RPR-001, UC-RPR-004]  delivers: [wazi repair help/preview command]  acc: [exact command is `wazi repair`; explicit one-file input, bounded diff/diagnostics and candidate digest are shown; no file is modified; existing host flags and packaged Mac startup still launch the host]
- [ ] S11.2.1 Add real-process CLI tests for preview, path confinement, bad input, and legacy host startup  Owner: CLI implementer  Est: 3h  kind: agent stage: verify  blocked-by: [T11.2]  verifies: [UC-RPR-001, UC-RPR-004]  acc: [tests invoke built CLI from unrelated cwd/minimal PATH; preview writes zero source bytes; traversal, symlink, nonregular, oversized and unsupported inputs fail closed; old Mac supervisor arguments retain the same readiness/API behavior]

- [ ] T11.3 Implement explicit apply with source CAS, exact backup and atomic replacement  Owner: CLI implementer  Est: 1d  kind: agent stage: implement  blocked-by: [T11.0, S11.2.1]  verifies: [UC-RPR-002]  delivers: [single-file explicit apply transaction]  acc: [apply requires an exact private candidate ID/digest and explicit `--apply`; rechecks source digest under cross-process lock; creates and fsyncs an owner-only exact-original backup before atomic same-filesystem replacement; refuses stale candidate, conflict, backup failure and out-of-root/symlink target without changing source]
- [ ] S11.3.1 Test apply recovery, CAS races, permissions and untouched neighboring plans  Owner: CLI implementer  Est: 4h  kind: agent stage: verify  blocked-by: [T11.3]  verifies: [UC-RPR-002, UC-RPR-004]  acc: [real filesystem tests prove source rewrite equals reviewed candidate, backup hashes to exact original bytes, rollback restores exact bytes, permissions are handled per T11.0, competing edits are retained and rejected, and adjacent plans remain unchanged]

- [ ] T11.4 Qualify EXPLABS backend endpoint/model contract from owner-provided authoritative documentation  Owner: product owner + provider qualifier  Est: 2h  kind: human stage: preflight  blocked-by: [T11.0]  blocked: Awaiting owner-supplied EXPLABS API/authentication/request/response/retention/rate-limit and model-capability documentation; do not infer OpenAI compatibility or make a live request  delivers: [qualified provider contract or explicit AI-disabled outcome]  acc: [provider identity, HTTPS/auth/header behavior, accepted model name, request/response/token/timeout limits, retention/privacy terms, redirects and retry semantics have cited owner-provided sources; absence keeps AI disabled]
- [ ] T11.5 Add optional EXPLABS proposal adapter using exactly `EXPLABS_API_KEY`, `EXPLABS_BASE_URL`, and `EXPLABS_MODEL`  Owner: provider implementer  Est: 1d  kind: agent stage: implement  blocked-by: [T11.0, T11.4]  verifies: [UC-RPR-003, UC-RPR-004]  delivers: [explicitly enabled provider proposal]  acc: [no provider request occurs without explicit `--ai`; request is confined to selected plan bytes and approved repair instructions; never load selected-project `.env`; a local secret scan that detects credential-looking content blocks AI dispatch without automatic redaction; candidate remains a proposal and is never applied by AI; key, prompts, plan bodies and answer bodies never appear in errors/logs; endpoint is HTTPS, credential-bearing URL components and unsafe redirects are rejected]
- [ ] S11.5.1 Add fake HTTP server tests for provider request/response, limits, redaction and no-call defaults  Owner: provider implementer  Est: 4h  kind: agent stage: verify  blocked-by: [T11.5]  verifies: [UC-RPR-003, UC-RPR-004]  acc: [tests cover exact env-name mapping without revealing values, malformed/oversized/truncated/rejected responses, timeout, redirects, auth failure, candidate validation, deterministic preview/apply issuing zero HTTP requests, and `--ai` issuing only the qualified single request]

- [ ] T11.6 Implement private repair-candidate/receipt storage, expiry, invalidation, capacity and deduplication  Owner: provider implementer  Est: 1d  kind: agent stage: implement  blocked-by: [T11.0, T11.5]  verifies: [UC-RPR-003]  delivers: [operation-qualified private repair state]  acc: [owner-only storage is separate from target repos and browser; cache identity includes operation, dialect/profile, qualified plan/task identity when stable, path identity, exact source digest, prompt/schema version, provider/model, owner constraints and all sent inputs; cache never aliases Dig deeper; concurrent exact requests deduplicate; completed proposals expire and invalidate on any input/source change; capacity is reserved before dispatch; non-content receipt distinguishes completed, failed and unknown outcomes; uncertain requests are never automatically resent; no automatic body backup/export/sync, and actual platform backup exclusion remains qualified or disclosed]
- [ ] S11.6.1 Test repair cache lineage, separate namespace, expiry/invalidation, deduplication and uncertain retry refusal  Owner: provider implementer  Est: 4h  kind: agent stage: verify  blocked-by: [T11.6]  verifies: [UC-RPR-003]  acc: [same exact repair input reuses only an eligible repair proposal; a Dig deeper record never matches; source/profile/model/prompt changes invalidate; parallel duplicate calls collapse; timeout-after-dispatch displays unknown and a later invocation performs no implicit resend]

- [ ] T11.7 Integrate config ownership, `.env` exclusions, CLI documentation and portable/desktop compatibility  Owner: coordinator  Est: 4h  kind: agent stage: implement  blocked-by: [T11.0, S11.3.1, S11.5.1, S11.6.1]  verifies: [UC-RPR-001, UC-RPR-002, UC-RPR-003]  delivers: [integrated command and user handoff]  acc: [only repair invocation loads dotenv from the primary Wazi project cwd by default with explicit `--env-file` override; never selected-plan `.env`; process env overrides file; plain `.env` is ignored and excluded from Mac bundle/CI artifacts; host launch flags, Go contract CLI, portable schema and digest remain unchanged]
- [ ] S11.7.1 Verify configuration privacy and source/build compatibility  Owner: verification owner  Est: 3h  kind: agent stage: verify  blocked-by: [T11.7]  verifies: [UC-RPR-002, UC-RPR-003]  acc: [fake key/base/model sentinels are absent from stdout/stderr/repo diff/bundle; config permission and precedence tests pass; `wazi repair --help`, desktop launch and `wazi-contract version/fixtures` pass; portable adapter availability is not falsely claimed; frozen contract digest is unchanged]

- [ ] T11.8 Run integrated verification and record changed behavior  Owner: verification owner  Est: 1d  kind: agent stage: verify  blocked-by: [T11.7, S11.7.1]  verifies: [UC-RPR-001, UC-RPR-002, UC-RPR-003, UC-RPR-004]  acc: [required Go tests/race where affected, Node parser regressions, format/lint, CLI subprocess and filesystem tests pass at exact candidate SHA; no live provider call; no authored source files change during preview; private candidates/backups stay outside tracked artifacts]
- [ ] T11.9 Independently review exact implementation candidate  Owner: independent reviewer  Est: 4h  kind: agent stage: review  blocked-by: [T11.8]  verifies: [UC-RPR-001, UC-RPR-002, UC-RPR-003, UC-RPR-004]  delivers: [exact-head review receipt]  acc: [reviewer is independent of implementation; base/head and covered tasks are recorded; source-write scope, schema boundary, identity/status preservation, secret redaction, retry/cache behavior and tests are reviewed; accepted findings have tracked fixes, affected verification and fresh re-review]
- [ ] T11.10 Merge reviewed repair candidate to GitHub main by rebase  Owner: coordinator  Est: 1h  kind: agent stage: merge  blocked-by: [T11.9]  verifies: [UC-RPR-001, UC-RPR-002, UC-RPR-003, UC-RPR-004]  acc: [exact reviewed PR head passes required checks and rebase merge guard; landed SHA and merge receipt are recorded; no provider call, release or deployment occurs]
- [ ] T11.11 Verify behavior and recovery on landed SHA  Owner: coordinator  Est: 4h  kind: agent stage: verify-landed  blocked-by: [T11.10]  verifies: [UC-RPR-001, UC-RPR-002, UC-RPR-003, UC-RPR-004]  acc: [fetched main contains reviewed tree; landed CLI preview remains read-only, explicit apply creates exact backup and preserves semantics, error/conflict paths fail closed, packaged Mac host still starts, and provider remains disabled unless explicitly opted in]

## Parallel Work

After T11.0, deterministic core/preview and provider contract research can proceed separately. Provider implementation waits for T11.4. Apply waits for preview candidate identity. Integration waits for deterministic/apply/provider seams. Verification and independent review do not overlap the implementation being reviewed. Runtime and CI builds use SSD caches/artifacts and the shared build lease only when one-minute load is at most10; no live provider calls are used for routine checks. Coordinator owns `cmd/wazi` dispatch, shared Go integration, `.gitignore`, packaging/workflow changes and project-root documentation. Any worker receives an isolated SSD worktree and explicit files only after T11.0 freezes package ownership.

## Timeline and Milestones

| Milestone | Tasks | Exit |
| --- | --- | --- |
| M1 — Contract frozen | T11.0, T11.4 | Markdown target and transformation policy chosen; provider contract cited or AI deliberately deferred. |
| M2 — Safe deterministic repair | T11.1–T11.3 and subtasks | Preview is read-only; exact candidate apply is recoverable and concurrency-safe. |
| M3 — Optional AI proposal path | T11.5–T11.6 and subtasks | Explicit AI proposal, private operation-scoped reuse and uncertain-request handling pass fake-server checks. |
| M4 — Integrated candidate accepted | T11.7–T11.9 | Privacy, compatibility, required checks and independent exact-head review pass. |
| M5 — Landed | T11.10–T11.11 | Reviewed tree lands and behavior is verified from fetched main. |

## Risk Register

| ID | Risk | Impact | Likelihood | Mitigation |
| --- | --- | --- | --- | --- |
| R11.1 | “Authorized plan schema” means portable JSON0.0.1 or another target instead of the ordinary Markdown dialect. | High | Medium | Owner decision at T11.0; safe default is Markdown-only; never mutate the frozen contract. |
| R11.2 | AI edits valid but author-specific wording/status/IDs while making syntax appear valid. | High | Medium | Compare complete authored semantic projection, reject invented semantic fields, display exact diff, require explicit apply. |
| R11.3 | EXPLABS endpoint/model protocol or data retention differs from assumptions. | High | High | Do not assume protocol; provider task stays blocked until owner docs qualify endpoint/model; fake HTTP tests precede any authorized live qualification. |
| R11.4 | Key leaks through project `.env`, redirects, logs, diff artifacts or bundle. | High | Medium | Use Wazi-owned config only; TLS/redirect policy; redact all errors; `.env` ignore/package tests and sentinel scan. |
| R11.5 | User edits the plan between preview and apply or a crash occurs during write. | High | Medium | Exact digest CAS under cross-process lock; backup must be durable before same-filesystem atomic rename; recovery regression tests. |
| R11.6 | Existing Mac host invocation or contract tooling is broken by CLI subcommand dispatch. | Medium | Medium | Dispatch exact `wazi repair` before legacy flags; real-process regression for Mac arguments; contract digest/fixture and normalizer checks. |
| R11.7 | AI proposal cache is confused with E3 analysis answers or a timed-out call repeats charges. | High | Medium | Separate operation namespace, qualify complete exact inputs, persist non-content dispatch receipt before request, deduplicate and never auto-retry unknown outcome. |

## Operating Procedure

Default mode reads one selected Markdown file and outputs a bounded diff and diagnostics. It writes no plan, backup, sidecar or provider receipt. Build the user-facing executable as `wazi` from `cmd/wazi`; do not rename/change the private host launch protocol inside the Mac bundle. Load the owner-provided primary Wazi project's `.env` only within `wazi repair`, defaulting to the primary working directory and allowing an explicit `--env-file`; never inspect/load the selected plan repository's `.env`. Process environment values override dotenv entries. `--ai` is a separate explicit opt-in and may send only the selected plan bytes plus the frozen repair instruction; before dispatch, show the selected provider/model identity, endpoint host and exact input digest. Do not include environment secrets, `.env` content or unrelated workspace/context. If the local secret scan detects credential-looking text in the selected plan, refuse AI dispatch and keep deterministic preview available; do not silently redact source content before generating a candidate. An AI response is an untrusted proposal: run it through deterministic parse/semantic checks and reject invalid candidates. It must never synthesize IDs, status, acceptance, owner, dependencies, stage or portable evidence.

Apply is a distinct explicit operation against the exact candidate identity/digest. Resolve one canonical regular file and reject symlink escape, path traversal, non-Markdown input, oversized input, split/include changes or a source digest mismatch. Acquire a cross-process lock, re-read and hash the original, persist and sync the exact original backup in Wazi-owned private data, write a same-filesystem temporary candidate, preserve source permissions under the frozen policy, sync and atomically rename, then sync the parent directory. If any step before rename fails, leave the source unchanged. If rename completion is uncertain, report it explicitly and reconcile from the source and backup hashes; never repeat an AI request automatically.

Diagnostics distinguish syntax repair, missing semantic input, ambiguous identity, unsupported source shape and provider uncertainty. A successful parser pass is not portable contract conformance, execution authorization, or evidence of task completion. Keep source plans as the sole authored authority; Wazi browser/desktop stay read-only. No provider call during CI, routine tests, discovery, preview without `--ai`, apply, build, packaging or verification.

Definition of done: complete the dependency graph through implementation, changed-behavior verification, independent exact-head review, GitHub rebase merge and landed verification. Include every accepted finding's fix, affected check and independent re-review. Record skipped/blocked provider qualification honestly; a non-live fake-server test does not qualify actual endpoint operation. No release, deployment or paid call is included.

## Progress Log

**Change Summary — 2026-10-04:** Added this executable future-delivery plan after source/contract discovery. No CLI implementation, provider call, source-plan rewrite or portable-contract change has been performed.

- 2026-10-04 — T11.0 planned as a blocking owner decision. Source inspection found parser/authoring rules are not the frozen portable JSON contract; exact EXPLABS protocol remains unqualified. Safe defaults and downstream gates are explicit.

## Hand-off Notes

- This file plans implementation only. Review, merge and verify landing of this document separately; that does not execute E11.
- Start future implementation from the then-current fetched `main`, not the planning base. Reconcile E11 with `docs/plan.md`, desktop packaging and concurrent Go work before editing.
- Parent/coordinator owns shared CLI dispatch, root docs, `.gitignore`, bundle/CI and final integration. Workers must receive explicit isolated file ownership. Do not edit the configured Three.js dependency checkout.
- Read the Go, team, `$ship` and relevant plan skills before implementation. Use external-SSD worktrees/caches, check load and acquire/release the shared build lease for heavy Go builds.
- Use `docs/rfc/0002-plan-code-evidence-traceability.md` for private-storage/request-receipt lessons without importing E3 cache entries into repair. Use the current provider's owner-supplied docs; do not infer them from variable names or the existing OpenRouter adapter.
- No credentials, `.env` values, private plans, repair candidates, request bodies or answer bodies belong in Git, logs, docs, tests, CI artifacts or screenshots.

## Appendix: source references

- [Current Go host](../../cmd/wazi/main.go), [portable bundle CLI](../../cmd/wazi-contract/main.go)
- [Wazi Markdown scanner](../../scripts/plans.mjs), [shared Markdown parser](../../src/plan-parser.mjs), [parser tests](../../tests/plans.test.mjs)
- [Portable contract README](../../contracts/plan/v0/README.md), [normative semantics](../../contracts/plan/v0/SEMANTICS.md), [frozen manifest](../../contracts/plan/v0/manifest.json)
- [RFC0002 AI/cache rules](../rfc/0002-plan-code-evidence-traceability.md), [current architecture](../architecture.md), [current data flow](../data-flow.md)
- Shared authoring guidance: plan skill `AUTHORING-METADATA.md` (reference only; no schema version is invented here).
