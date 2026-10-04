# ADR 0001 — Go observatory with confirmed links and project context

- Status: accepted design; not implemented.
- Date: 2026-10-03 (peer/user clarification continued on 2026-10-04 UTC).
- Design digest: `WAZI-DESIGN-DIGEST-v1`.
- Contract: [RFC 0002](../rfc/0002-plan-code-evidence-traceability.md).

## Context

The existing prototype visualizes local plans through React/Three.js. The user directly selected local discovery with code analysis only for selected projects, confirmed candidate links, rich read-only Serenity context from the first version, a Go core and click-triggered GPT Luna assistance with persisted reuse. The user asked both sessions to settle and record the design, then stop runtime work; the later shared plan-skill discussion request adds discussion-only work.

## Decision

Use an on-demand Go local host/library/CLI serving the preserved Three.js browser UI. Go owns discovery, selected-project analysis, bounded credentialed operations, versioned snapshots, authored binding writes and private application-result persistence. Plans/code stay read-only. Markdown/parser behavior is preserved through an explicit compatibility seam rather than a prerequisite rewrite.

Automatic local analysis emits explained file/test candidates. Only explicit user confirmation creates an authored versioned sidecar binding. Qualified identities, plan/target digests and dirty-tree/mixed-read detection keep provenance and freshness distinct from execution, review, landing and accepted completion.

Use the shared Serenity brain through explicit project/entity/reference mappings. The first panel includes facts, decisions, constraints, intents and open questions, with truthful unavailable sections. Supported authenticated provider-free fact reads and the richer scoped authored-context owner seam have separate qualification gates. Global Brief relevance ranking is not project isolation. No canonical-file/private fallback, writer provisioning, memory write or complete graph claim is permitted.

On an explicit Dig deeper click, use OpenRouter `openai/gpt-6-luna` with selected plan/code plus the project-scoped Serenity context shown in the panel. Persist and reuse completed answers by exact input/model/prompt/configuration identity. Context-derived outputs additionally bind memory lineage and require current eligibility/content/expiry validation before reuse. Forgotten/inaccessible/expired inputs invalidate/remove derived bodies. Offline/unreachable validation keeps persisted memory-derived bodies hidden and unavailable for reuse/export, including historical views; plan/code-only cache is separate. If lineage cannot be qualified, durable reuse is unavailable and output stays ephemeral; this does not pass the complete v1 persistence gate.

Use private owner-only local app data, separate credentials and no body logs/automatic exports. Capacity checks precede paid calls, cross-instance requests deduplicate, and unknown/uncached completion never causes an automatic resend. Provider/runtime/account cost qualification remains future work.

## Rationale and consequences

Go puts filesystem and credentials behind one local boundary while Three.js preserves the chosen 3D interface. A CLI-only viewer cannot directly support browser confirmation and credentialed contextual requests without a local bridge; the on-demand Go host makes that boundary explicit. A native rewrite or immediate parser translation would discard working behavior without improving the agreed authority model.

A declared-links-only release or facts-only/later-memory scope would omit direct first-version choices. Broader context and memory-inclusive AI are therefore required v1 capabilities with explicit owner-contract and privacy acceptance gates, rather than assumed existing APIs. No graph database, autonomous executor, model-on-refresh or remote deployment is required.

All direct user choices are resolved. Peer concurrence `SERENITY-WAZI-FINAL-CONCURRENCE-v1-REFINE-20261004` and independent review accepted exact semantic candidate `53f830d9ae31beb9651a8876e8f7d285eca7eb5a`; no material user choice remains. This records design intent; it does not claim backend integration, actual brain connectivity, provider execution or full shared-plan consumer compatibility. Portable contract maintainer/schema choices are a separate discussion.
