# Wazi

- This is a local, read-only prototype. Preserve source plans; do not add plan writes, remote ingestion or deployment without a request.
- Three.js is consumed from the configured local checkout. Do not edit that dependency checkout.
- Use `npm test` for the plan reader and `npm run build` for the single-package frontend. Verify changed interactions in a real browser.
- Keep dependencies, generated build output, temporary fixtures and browser evidence on the external SSD during agent work.
- Update docs/design.md for design/architecture choices, docs/devlog.md for verification and docs/plan.md for delivery status.
- Do not commit private plan snapshots or generated screenshots. The example plan must remain visibly labeled as sample data.
