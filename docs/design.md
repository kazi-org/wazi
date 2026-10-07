# Wazi prototype

A local, read-only plan observatory inspired by the supplied column-and-dependency reference. The intentional redesign uses a deep ink canvas, cyan and periwinkle accents, spatial task cards and inspectable dependency paths. Three.js comes from the user's local checkout; never edit that checkout.

## Ownership

Coordinator owns frontend, Vite configuration and integration. Plan reader worker owns scripts/plans.mjs and tests/plans.test.mjs in an isolated worktree. Independent reviewer reads the exact integrated candidate before local main integration. No deployment or remote publication is in scope.

## Plan reader contract

scripts/plans.mjs exports parsePlan(markdown, {path, project}={}) and scanPlans({root, repoMap}={}). scanPlans resolves {projects: [{id, name, plans: [{id, title, path, tasks, epics, warnings}]}], warnings, scannedAt}. Tasks have id, title, status (complete|active|blocked|pending), stage, epicId, epicTitle, owner, acceptance, dependencies (array), line, source. Epics have id and title. Default scan root is ~/Code; bound recursion and skip dependencies/build trees. No source plan writes. Local data is served from /api/plans on loopback only and never bundled or committed. Browser Markdown import is an alternative.

## As-built design notes — 2026-10-03

The implementation at `8c5486d` uses a hybrid renderer: live Three.js world/camera/curves underneath HTML task buttons projected into screen coordinates. The five visual lanes map exact stage values to Discover, Build, Verify, Review and Land; missing or unknown stages fall back to Build. Merge and landed verification share Land. This is a stage view, not a critical-path or topological-layout algorithm.

The initial view is a visibly labeled illustrative plan. Local projects and imported plans receive separate source labels. Selecting a task reveals its recorded status, owner, epic, source line, acceptance text, dependencies and same-plan dependents. Status controls dim nonmatching tasks rather than removing them. The dependency control hides/shows the curve geometry. Map view disables rotation; it still uses the same Three.js perspective camera. Explicit camera focus centers on the selected visible task, and reset returns to the home view.

At most four tasks per lane appear in the canvas. Selecting another task includes it by replacing the last item in that lane. The complete epic-filtered task list remains the reading/navigation path, especially on narrow screens. Narrow layout uses a workspace drawer and a task-inspector overlay; the canvas becomes a miniature overview. Physical-device touch and no-WebGL behavior remain verification gaps, as recorded in the original visual QA.

Architecture and API/record details are maintained in [architecture.md](architecture.md) and [data-flow.md](data-flow.md). [RFC 0001](rfc/0001-local-plan-observatory.md) describes the implemented choices and tradeoffs; its future ideas are proposals, not accepted commitments. The initial ownership/contract text above is preserved as historical design context. The implementation additionally returns `sourceId` and `project` fields and currently uses path-based hashed IDs before the browser normalizes tasks to source IDs within a selected plan.

## Initial product-extension proposal — historical peer handoff

The peer session relayed a direction toward plan/code/evidence traceability and eventual optional Serenity context, with Go preferred for a new CLI/service. [RFC 0002](rfc/0002-plan-code-evidence-traceability.md) records the proposal separately from the existing design: explicit qualified task-to-file/test bindings first; structural source extraction and contextual memory later. Code presence, declared plan status, recorded check evidence and accepted judgments must remain separate. No runtime extension or new service is implemented by this docs pass, and no Serenity endpoint is assumed qualified.

## Settled observatory design — not implemented

The user's direct choices supersede the earlier declared-links-first/later-memory proposal. [RFC 0002](rfc/0002-plan-code-evidence-traceability.md) and [ADR 0001](adr/0001-go-observatory-and-context.md) record the settled Go core serving the preserved Three.js interface, local plan discovery with code analysis only for selected projects, automatic explained candidate links requiring confirmation, click-triggered OpenRouter `openai/gpt-6-luna` with exact persistent result reuse, and read-only shared-brain Serenity context scoped to the selected project. The selected visual direction and complete task-list navigation remain the baseline.

The inspector separates plan state, authored links, inferred suggestions, code/check observations and remembered context. Backend credentials stay in Go; memory bodies stay outside traceability exports. The Go core with existing Three.js browser interface is explicitly confirmed. The first Serenity panel includes project facts, decisions, constraints, intents and open questions, with truthful unavailable sections and a required project-scoped owner read-contract gate. The user explicitly permits deeper AI to use selected plan/code plus the displayed project-scoped Serenity context; durable reuse requires complete memory-lineage eligibility/expiry validation and invalidation of derived answers. All material choices are resolved. Peer concurrence and independent review passed exact semantic candidate `53f830d`; ADR 0001 records the accepted design. Implementation/brain/provider qualification remains future work. This is design work only; no backend, provider or context integration is implemented or qualified.

## Portable plan-contract discussion — closed at recommendation level

At the user's request, the Wazi session introduced its lane, contributed source-verified consumer requirements, reviewed the shared synthesis and watched through `DISCUSSION-CLOSED-01`. The shared skills record is `docs/tasks/portable-plan-contract-discussion-20261004.md`, content SHA256 `6e17965153a8e8967372c52c64f935872737543e7227282facb6a86e62e2b69c`, locally retained in commit `2564160bd7b8a51ea8801d70f0ad57aeab6ec1a5`. That reference identifies discussion evidence, not an approved schema or remote-delivery proof.

Participants recommend a portable headless experimental interchange separating PlanDefinition, ExecutionSnapshot and evidence/requirement evaluations. Preserve one authored authority per plan revision: ordinary Markdown input, native service authority where enrolled, and version-pinned normalized JSON interchange. Full-SDLC logical gates, policy-conditioned check acceptance and revision-bound evidence remain distinct from presentation lanes. Compound units retain one canonical admission/execution authority; Wazi must not duplicate their scheduler. Unknown/stale/unsupported evidence cannot become accepted completion, and audit-only late facts do not reopen canceled/expired execution.

Wazi stewardship of the shared schema/semantic validator/conformance fixtures remains OPEN with the user and maintainer as decision owners; the future-ownership question was raised explicitly. Wazi accepted consumer/design participation only. Exact vocabulary/versioning/migration, contributor-lineage versus authority-attestation proof and actual consumer qualification have named follow-up owners in the synthesis. No shared-contract implementation, provider effect or deployment is assigned or qualified. This closed recommendation-level discussion is separate from accepted observatory ADR 0001.

## Collapsible observatory panels

The browser starts with a 66 px icon-only project rail and a 42 px task-inspector rail on desktop. Each panel expands independently; the project panel restores names, counts and search. Project and utility icons retain accessible labels, native hover titles and hover/focus label treatment. Closing the inspector preserves the selected task, and selecting a task opens its details. The inspector rail is hidden while details are expanded, so it reserves no extra width. Panel state is local to the current page session.

At 1278 px viewport width, the compact scene occupies 1170 px (about 92%); expanding both panels leaves 771 px. The existing ResizeObserver adjusts the canvas and camera aspect. Memoized task arrays keep scene data stable across panel toggles. Resize initializes the camera home once, then preserves its orbit and zoom while updating aspect; explicit reset and project navigation still restore the home view. Below the existing breakpoints, navigation remains a drawer and details an overlay controlled from the top bar. Closed mobile navigation is hidden from focus and accessibility navigation. The open workspace drawer moves and traps keyboard focus, makes the scene/details inert and restores focus to its opening control on close.

## Experimental portable contract implementation

[ADR 0002](adr/0002-experimental-portable-plan-contract.md) records the newly
assigned bounded stewardship. `contracts/plan/v0` contains separate structural
projections and normative semantics for definition, execution and evidence/
requirement judgments; `manifest.json` pins the frozen schema/fixture bytes. The
Go library and CLI validate shape/coherence offline and never authenticate receipt
claims or authorize dispatch. Browser import is a read-only consumer: preserve
metadata, raw source and stage tokens while showing producer-reported judgments
as unverified. Markdown checkboxes remain authored completion assertions.

Unsupported stages have an explicit Other presentation lane, rather than an
implementation fallback. Core typed dependency predicates are never inferred
from the visual lane. Local Markdown discovery remains the existing Node/Vite
prototype; this package is not the future Go-host migration or an implementation
of ADR 0001's provider/context/cache gates.

## Go observatory implementation boundaries

The local host coordinates plan discovery through the existing JavaScript parser seam, keeping authored metadata and unsupported stages intact. Discovery reads plans; source analysis starts only for explicitly selected projects. Repository identity is application-owned and distinct from filesystem locators. Confirmed task links live in a versioned sidecar with source digests and concurrent-write checks; they establish source presence and freshness, never qualified completion. The inspector integrates proposals, bounded read-only source previews, reverse task links and scoped context inside the existing collapsible layout.

The host constructs analysis inputs from current source snapshots and a leased copy of the exact context displayed by the browser. Browser payloads carry selection identifiers and explicit intent, not authoritative source bodies or credentials. Context uses explicit repository/brain/audience/project mappings and a qualified owner reader; an injected test reader does not qualify the actual owner API. All typed sections remain visibly unavailable when that reader is absent.

Private result persistence separates plan/code-only input from memory-derived input. Memory-derived display and reuse requires current lineage eligibility; unreachable validation hides bodies, while negative eligibility and expiry invalidate them. Persistent dispatch receipts prevent automatic resend after uncertain outcomes. Regeneration requires fresh host-qualified inputs and an explicit cost acknowledgement. These boundaries are independent of the frozen portable plan contract.

Candidate delivery decision: the user's explicit headless-review-and-merge instruction authorizes landing the observatory with honest unavailable Serenity context. Full rich-context qualification remains an open E3 requirement. Retain CI-built frontend artifacts temporarily for reproducible offline headless and landed acceptance while respecting shared local build load limits.

## Accepted desktop direction — planning only

[ADR0003](adr/0003-platform-desktop-shells.md) selects a Swift/AppKit + WKWebView Mac shell for phase1 and Wails for later Windows/Linux. Preserve the shared Go host and React/Three.js view. The native shell owns only window/lifecycle, packages the Go host and pinned Node/parser/assets, and loads the owned loopback origin with existing session protections. No native title bar/browser chrome; dedicated drag and accessible custom controls preserve canvas interaction. App-private persistence stays Go-owned. Runtime/native qualification remains the checkable E8 plan; no application implementation or release was performed by this planning pass.

## Native Mac shell implementation (E8)

The desktop surface reuses the React/Three.js app inside a nonpersistent WKWebView. AppKit owns its single window, menus, read-only file chooser and host lifetime. The public full-size-content window hides its title, titlebar separator and standard buttons; web controls retain the dedicated 34px strip. Native messages accept only exact named window actions from the owned main frame at the pinned host origin. A fresh nonce/readiness protocol identifies the bundled Go process; closing the app closes its parent-watch pipe. Startup and content failure replace the web view with an explicit recovery panel.

A pure borderless window rendered correctly, but native pointer automation could not locate its window; the public hidden-titlebar window is the native proof fallback. Actual runtime matrix and remaining acceptance gates are recorded in the E8 plan. The portable contract and configured Three.js source dependency are unchanged. Confirmed link removal uses the existing private versioned sidecar CAS seam and leaves authored source files unchanged.

### Native application identity

The Mac app and Dock icon reuse the interface's Phosphor Planet duotone mark, lavender `#a2afff` over `#0d111d`. Versioned SVG and ICNS live under `apps/macos/`; the bundle carries the icon's MIT notice. This preserves Wazi's current identity without introducing another logo.

### Owner-selected aperture identity

The owner selected the first generated aperture design to replace the planet logo in both the interface and native app. `src/assets/wazi-logo.png` is the exact selected source; the header imports it and `apps/macos/Wazi.icns` contains resized representations. Preserve the chosen composition rather than tracing a different vector approximation. The app icon notice records generated-image provenance; Phosphor notices remain for the other interface icons.

### Planned explicit repair command

The owner requested the top-level `wazi repair` command. E11 plans a separate, explicit source-repair CLI: deterministic Markdown preview first, owner-applied exact candidate with conflict checks and recoverable backup, and optional EXPLABS proposals. The browser/desktop observatory stays read-only. Existing Markdown authoring guidance and the frozen portable JSON contract are separate authorities; preflight must qualify the chosen profile without inventing task meaning or evidence. This is planned work, with no command implementation or backend call included in the planning artifact.

## Explicit deterministic plan repair

E11 adds a top-level CLI operation before legacy host flag parsing. Its frozen ordinary Markdown profile is separate from portable JSON and completion evidence. Preview performs no writes; explicitly saved source-bound candidates live outside the selected repository, and apply uses a private exact backup plus guarded atomic replacement. Missing semantics remain owner-authored diagnostics. The browser and Mac app remain read-only. Optional EXPLABS and interactive proposal modes are explicitly unavailable in this first local delivery; no dotenv is loaded. See [profile](repair-profile.md) and [usage](repair.md).

## Explicit AI repair extension

`wazi repair --ai` is a distinct EXPLABS proposal operation, using only one selected plan and fixed syntax instructions. It does not share E3 analysis caches or invoke a provider from host startup, deterministic preview or apply. All proposals pass a byte-preserving checkbox policy and existing semantic diagnostics; missing meaning is never invented. The candidate store remains the single explicit source-write transaction with digest CAS and exact backup.

Requests use the documented HTTPS endpoint, bounded nonstreaming output and one generation attempt without fallback. Operation/policy/prompt, complete source/path, model/endpoint, credential-scope fingerprint and generation settings qualify private reuse. A durable non-content receipt precedes dispatch; uncertainty and definitive failure both refuse implicit resend. Completed-body eligibility is 24 hours with sixteen worst-case 4-MiB capacity reservations. This eligibility is separate from owner-explicit generic candidate/backup retention. Only the request cache gets best-effort Time Machine exclusion; other backups or sync are not qualified. Provider retention depends on account settings and is disclosed without changing those settings.

Mac discovery uses existing correct bundle metadata and scoped registration/import. A read-only Spotlight volume is an external system gate, not evidence that changing the logo or app signature would repair search.

2026-10-06 column-header repair: stage labels belong to the same spatial columns as task cards. Use world-space anchors above each column and the current camera projection/scale so zoom, pan, orbit and focus move the label with its column. Keep labels noninteractive and preserve the existing six lanes, selected-task behavior and visibility cap. No Delivery-view or Serenity change is included.

2026-10-06 E15 supersedes the prototype four-per-lane cap: selecting a project opens All plans; selecting one plan narrows the view. Canvas/list/counts use all recognized tasks in the current plan scope. UI identity binds source plan and authored task ID; host inspection keeps the original plan/task identity and dependency edges stay within their authored plan. Repeated epic/task IDs across plans must not merge records. Plan discovery remains bounded to120 direct split Markdown files, with a clear incomplete-discovery warning; All plans means all indexed plans, not an assertion of exhaustive disk discovery.
