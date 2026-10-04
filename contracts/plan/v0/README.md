# Portable plan contract 0.0.1

Experimental, headless, read-only interchange maintained here for consumer
adapters. [SEMANTICS.md](SEMANTICS.md) is normative for mapping/coherence;
[contract.schema.json](contract.schema.json) is JSON Schema Draft 2020-12.
The three projection schemas separate authored definition, current execution and
evidence/evaluations. Stages remain open tokens. The browser is one consumer.

## Pin and consume offline

Pin the repository revision and `manifest.json` contract digest:

`sha256:7582512f122d2f2a9c4461facc7541c9887053f137260d6ebe9c6dea611d039d`

The manifest hashes schemas, normative semantics and neutral fixtures, in lexical
path order: SHA256 of UTF-8 `(path + NUL + lowercase file SHA256 + LF)` records.
`manifest.json` excludes itself; Go implementation and operational README are not
part of that specification digest. Consumers must verify listed bytes, exact path
coverage and digest before using the pin. Schema `$id` URNs are identifiers,
not network fetch locations. All schema resources are embedded; external schema
loading is disabled. Fetch dependencies once when building, then the built CLI
requires no network, provider credentials, browser or running Wazi service.

```sh
go build -o /path/to/artifacts/wazi-contract ./cmd/wazi-contract
/path/to/artifacts/wazi-contract version
/path/to/artifacts/wazi-contract fixtures --contract-digest sha256:7582512f122d2f2a9c4461facc7541c9887053f137260d6ebe9c6dea611d039d
/path/to/artifacts/wazi-contract validate --contract-digest sha256:7582512f122d2f2a9c4461facc7541c9887053f137260d6ebe9c6dea611d039d /path/to/bundle.json
```

The library is `github.com/kazi-org/wazi/contracts/plan/v0`. A valid result means
shape/coherence only, with authority authentication false. Consumers must still
verify source-byte digests, native receipt provenance, grants/policy, contributor
lineage and environment before accepting readiness or success. Use named unknown
sentinels only for missing identities/policy in nonsatisfying assertions; never
pretend those are authenticated issuers or approvals.

`fixtures/catalog.json` names 60 neutral valid/invalid cases, including parallel
checks/review, bounded correction, stable delivery gates, terminal audit-only
facts, native narrative status, compound units, stale bindings, independent-review
limits and unavailable CI with/without scoped authorization. Their receipt URNs
and success assertions are illustrative; no live result is claimed.

## Versioning and compatibility

0.0.1 is frozen at public specification revision
`f04497a3fcde3c1b78d09b683405d4d9f7645efc` on
[`contract/experimental-0.0.1`](https://github.com/kazi-org/wazi/tree/f04497a3fcde3c1b78d09b683405d4d9f7645efc/contracts/plan/v0).
Its normative bytes match the original local freeze `16b66e5`. Consumers pin the
exact revision and manifest digest; this specification branch contains the
schemas, semantics and fixtures, while the Go implementation lives on the
reviewed delivery branch/main. Repository delivery and Go conformance receipts
are recorded separately in the project devlog. Do not
mutate a frozen schema/semantic/fixture file under the same version. Any change to
those bytes requires a new experimental version and digest; consumers opt in
explicitly. There is no stable-v1 compatibility promise or release publication.

Markdown normalization supports one authoritative file per definition; missing
acceptance/stage or split/include authority produces out-of-band diagnostics.
Native sources retain their own authority; normalization never creates a second
writable master or scheduler. Namespaced metadata preserves native array
acceptance, request/sequence/lifecycle and other opaque source fields. A snapshot
has one homogeneous subject; heterogeneous candidates remain unqualified or use
separate authoritative slices. Core references reject local absolute/private or
credential-bearing URLs; arbitrary metadata still requires consumer export review.
