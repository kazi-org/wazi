# Go observatory operations (candidate)

This document describes the E3 delivery candidate. Full E3 acceptance remains gated by the supported Serenity owner APIs and the delivery records in docs/plan.md. Fixture readers do not establish live connectivity.

The Go host serves built frontend assets on loopback, discovers plan files beneath its configured root, and analyzes code only after an explicit selected-project request. The browser imports and sample plan remain display-only. Build with `THREE_JS_SOURCE` pointing to the preserved local Three.js checkout. Keep build output, Go caches, Node dependencies and app-private data on the external SSD during agent work.

The local browser capability is fetched from `/api/session` and sent as `X-Wazi-Session` with JSON mutations. Host and same-origin checks reject cross-site requests. It grants access only through the loopback host and does not grant remote service authority. Treat local processes and the selected filesystem root as trusted access.

Project selection clears prior context and outputs. Proposed links explain their deterministic local basis. Confirming a proposed or manual file/test link writes only the versioned `.wazi/links.json` sidecar. Plans and code remain read-only. Source presence and authored checkbox status do not establish tests passing, review, landing or qualified completion. Stale basis or sidecar revisions reject confirmation and require refresh.

Serenity mappings must explicitly bind repository identity to brain, audience, project and allowed entity/reference identifiers. A missing or unqualified owner reader displays all sections as unavailable. Never substitute global relevance ranking, canonical file reads, or a facts-only response for the required typed rich context. A future qualified reader must prove supported authenticated provider-free reads and all-type lineage revalidation; Wazi's injected fixture contract alone is insufficient.

`Dig deeper` is the only model-dispatch control. The separate plan/code-only operation explicitly excludes memory. Neither startup, project selection nor refresh calls OpenRouter. Credentials belong in host process configuration, never sidecars or browser payloads. No live model call was authorized for agent verification: HTTP tests use local fixtures. Missing credentials must return a typed unavailable response rather than activate a provider.

Completed answers use exact qualified inputs, model, prompt/analyzer version and configuration. Memory-derived bodies must remain hidden while their owner eligibility cannot be revalidated; negative eligibility or expiry invalidates derived bodies. An uncertain dispatch cannot be retried automatically. Explicit regeneration warns that a new provider request may incur cost. Inspect/delete operations do not require a model call. App data is owner-only and is excluded from source control; do not export or log private bodies.
