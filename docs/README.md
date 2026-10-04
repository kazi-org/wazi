# Wazi documentation

These documents reverse-engineer the local prototype at revision `8c5486d`. Runtime code was independently reviewed at `f60775f`; this is an implementation description, not a hosted-system or production-readiness claim.

| Document | Read it for |
| --- | --- |
| [RFC 0001 — Local plan observatory](rfc/0001-local-plan-observatory.md) | Problem, implemented approach, tradeoffs and explicitly proposed future work |
| [RFC 0002 — Plan/code/evidence traceability](rfc/0002-plan-code-evidence-traceability.md) | Settled Go/Three.js design, confirmed links, requested AI/cache and rich first-version Serenity; not implemented |
| [ADR 0001 — Go observatory and context](adr/0001-go-observatory-and-context.md) | Durable design decision and qualification boundaries |
| [Architecture](architecture.md) | Components, runtime modes, dependency/rendering boundaries and lifecycle |
| [Data flow](data-flow.md) | File discovery, parsing, JSON records, status derivation, browser state and scene projection |
| [Design](design.md) | Visual direction, delivery lanes, task inspection and narrow-screen behavior |
| [Local operation](operations.md) | Setup, configuration, limits, troubleshooting and data handling |
| [Delivery plan](plan.md) | Checkable task/delivery records and current documentation lane |
| [Devlog](devlog.md) | Implementation events, decisions, verification evidence and caveats |
| [Implementation review](review.md) | Independent review of the original runtime candidate |
| [Visual QA](../design-qa.md) | Original browser comparison/checks and unverified surfaces |

Treat source checkboxes as recorded plan state. Wazi does not execute tasks or validate their acceptance criteria. The local discovery API is present in `npm run dev`; static output supports the sample and browser imports. Existing authored records are preserved; new design notes are additive.

Cross-session conversation lives in the ignored project-root `ajent.social` file. Architectural explanations belong here; decisions require a durable decision record rather than a channel message alone. RFC 0002 and ADR 0001 record the future product design separately from the as-built baseline. The current lane settles and records that design; it assigns no runtime implementation.
