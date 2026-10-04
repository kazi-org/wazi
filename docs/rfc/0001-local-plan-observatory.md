# RFC 0001 — Local, read-only plan observatory

- Status: informational, reverse-engineered from the implemented prototype.
- Date: 2026-10-03.
- Source baseline: local revision `8c5486d`; runtime implementation independently reviewed at `f60775f`.
- Scope: document existing behavior and its tradeoffs. Proposed improvements below are not implemented or accepted decisions.

## Problem and intended outcome

Markdown delivery plans hold task IDs, stages, acceptance criteria and dependency references, but a long document makes relationships hard to see. Wazi presents those records as an interactive spatial view while retaining the original plan files as the source of truth. The supplied inspiration diagram contributed colored columns and dependency paths; the prototype reinterprets them as five delivery lanes in a dark 3D observatory.

## Implemented approach

The local Vite development server discovers plan files, parses them using a browser-safe shared parser, derives dependency blocking across the project's discovered files and returns JSON. React keeps a snapshot of that response and the user's selections in memory. Three.js renders the spatial environment and dependency geometry; HTML buttons form the camera-projected task cards and inspector. Markdown import runs the shared parser entirely in the browser.

The development server reads files; it does not execute plan tasks, invoke the plan skill, verify acceptance predicates, modify checkboxes or talk to an agent scheduler. A completed checkbox is a recorded assertion, not independent evidence that a capability works. This RFC does not change the original implementation's readiness boundary.

See [architecture](../architecture.md), [data flow](../data-flow.md) and [design](../design.md) for the detailed contracts.

## Requirements represented by the prototype

| Requirement | Existing behavior | Source |
| --- | --- | --- |
| Explore local project plans | Repository/plan/epic selection and refresh | [App](../../src/App.jsx), [scanner](../../scripts/plans.mjs) |
| Preserve plan meaning | Retain IDs, owner, stage, criteria, dependencies and source location | [parser](../../src/plan-parser.mjs) |
| Inspect relationships spatially | Stage lanes, orbit/pan/zoom, curves and arrows | [Space](../../src/Space.jsx), [lane mapping](../../src/demo.mjs) |
| Access every task | Complete task list; selected task enters the bounded scene | [App](../../src/App.jsx) |
| Preserve authored plans | Scanner uses file reads; imports remain in memory | [scanner](../../scripts/plans.mjs), [App](../../src/App.jsx) |
| Support a static prototype | Sample and browser imports work without the local API | [Vite configuration](../../vite.config.mjs) |

## Architecture choices and tradeoffs

### Shared Markdown parser

The same pure parser serves Node scanning and browser import. This avoids maintaining two syntax readers. It supports the subset used by this prototype, not a general Markdown AST or a full compiler for plan-skill execution semantics. Missing or unsupported fields are not synthesized into verification evidence.

Dependency status derivation is currently separate: scanning resolves references across all discovered plans in one project and may override any non-complete task to blocked; import resolves only the imported document and changes pending tasks to blocked. These are implementation differences to preserve in documentation, not a claim of equivalent behavior.

### Local development API

`GET /api/plans` is a Vite `configureServer` middleware. It keeps filesystem access out of browser code and lets the owner choose the scan root. It is absent from the built static site and Vite preview server. It performs a fresh scan on each request; there is no persistence service, authentication system, request cache, refresh daemon or streaming protocol.

The server binds to loopback by default and guards Host, Origin and cross-site browser fetches. These are local-prototype boundaries, not qualification for exposing the server publicly. Plan records include extracted task text and source locations; treat responses as potentially private.

### Hybrid DOM/WebGL rendering

Three.js owns the camera, controls, background grid, point field and connection geometry. HTML task buttons remain readable, focusable and available to browser accessibility tooling. Each animation frame projects world positions into the card layer. Cards face the screen rather than becoming textured 3D meshes.

The five lanes are fixed UI groupings, not an inferred topological sort or critical-path calculation. Unknown stages go into Build. Merge and landed verification share Land. Status controls emphasize matching tasks instead of deleting other nodes.

### Bounded spatial overview

Up to four tasks per lane keep the default view legible. All filtered tasks remain in the list, and selecting a task beyond the initial four replaces the last visible member of its lane. Curves are drawn only when both endpoints are visible. The inspector searches the selected plan; a cross-file reference can affect scanner status without gaining a drawn node or a navigable target.

### Locally supplied Three.js

Vite aliases `three` and `three/addons/` to an existing checkout configured by `THREE_JS_SOURCE`, with a user-specific default. The source checkout is consumed without modification, and its JavaScript becomes part of the build. This supports the requested local dependency but means a fresh checkout is not self-contained: the caller must supply the library checkout as well as npm dependencies.

## Alternatives and limitations

These are explanatory tradeoffs inferred from the implementation; no claim is made that the alternatives were historically evaluated:

- A database-backed service could persist settings and snapshots, at the cost of another source of truth and hosting/access controls.
- A full WebGL card renderer could make cards physically rotate, at the cost of additional text-rendering and accessibility work.
- A unified project-wide graph could expose cross-file links, but needs unambiguous identities, duplicate-ID handling and a larger-scene strategy.
- A streaming filesystem watcher could update automatically, but the current explicit refresh path is simpler and bounded.

Current limitations include bounded discovery, no runtime schema enforcement, duplicate source-ID ambiguity, separate status derivation paths, no plan-warning surface in the UI, no request cancellation/versioning and no durable import storage. These are described in [data flow](../data-flow.md) and [operations](../operations.md). They are not implemented follow-up commitments.

## Verification evidence

The existing [visual QA](../../design-qa.md), [independent implementation review](../review.md) and [devlog](../devlog.md) record the original local verification: ten parser/discovery tests, a production build and browser interaction checks. Native file-picker automation, physical-device touch gestures and no-WebGL behavior were not fully exercised. This documentation pass does not promote those records into CI, deployment or production acceptance evidence.

## Potential next RFCs

The next material changes would benefit from their own explicit proposals: project-wide graph identity and dependency resolution; a shared status-derivation contract; root allowlists/symlink boundaries; automatic refresh with request ordering; or a hosted service. None is authorized by this informational RFC alone.

The subsequent [RFC 0002](0002-plan-code-evidence-traceability.md) records peer-relayed plan/code/evidence direction as a proposal separate from this implemented baseline.
