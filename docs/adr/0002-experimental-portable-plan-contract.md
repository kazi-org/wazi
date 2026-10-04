# ADR 0002: Maintain the experimental portable plan contract in Wazi

Status: Accepted for the bounded PC-WAZI implementation dispatch.

The earlier shared discussion recommended Wazi stewardship while leaving the
user decision open. The later dispatch and direct ship request assign Wazi the
initial experimental-v0 schema, semantic validator and neutral fixture package,
plus its read-only display compatibility. Historical discussion records remain
unchanged; they did not authorize implementation retroactively.

Maintain one frozen version/digest in `contracts/plan/v0`, separately from browser
presentation, executor internals, native wire contracts and provider policy.
Use Go for the headless validator/library/CLI. Structural and semantic validity
means coherent assertions; it cannot authenticate receipts, issue authority,
prove source-byte content or admit native service execution. Consumers pin the
same specification and own their source/receipt/policy qualification.

Keep authored definition, current execution and evidence/requirement judgments
separate. Typed dependency gates can join parallel checks/review, preserve bounded
negative-review correction and represent landing/deployment distinctly. Compound
units retain one named canonical service authority. Terminal late facts remain
audit-only. Authored checkboxes and narrative AMOS/context records do not qualify
completion. Unsupported stages remain visible in Wazi without dispatch meaning.

Experimental 0.0.1 bounds Markdown normalization to one authoritative file;
source gaps are diagnostics, never invented task/source fields. Native request,
sequence, array acceptance and opaque metadata remain consumer-owned mappings.
A single snapshot has a homogeneous subject; heterogeneity stays unqualified or
uses separate authoritative slices. Independent-review lineage must be claimed
explicitly and authenticated by consumers; singular-author external receipts
cannot silently establish all-contributor independence.

This scope does not implement the broader Go-host observatory in ADR 0001,
Serenity connectivity, AI/link suggestions, canonical scheduling, provider
activation, runtime interoperability, release publication or deployment.
