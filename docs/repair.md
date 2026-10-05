# Repair one local plan

Build the `wazi` executable from `cmd/wazi`. The repair command is independent of the browser and native app; both remain read-only. This deterministic version does not call AI or load `.env`.

```sh
go build -o /path/to/bin/wazi ./cmd/wazi
/path/to/bin/wazi repair /path/to/project/docs/plan.md
```

Preview shows source/candidate SHA256 digests, the exact syntax diff and line/field diagnostics. It writes no files. Repair only normalizes unequivocal checkbox syntax. Missing IDs, owner, stage or acceptance must be authored explicitly in the source; repair does not invent intent. Unsupported stages remain visible. Ordinary Markdown validity is separate from portable JSON conformance and task completion.

To save a reviewed candidate, choose a private data directory outside the selected repository:

```sh
wazi repair --data /private/wazi-repair --save-candidate /path/to/project/docs/plan.md
wazi repair --data /private/wazi-repair --apply-candidate PRINTED_CANDIDATE_ID
```

Flags go before the source path. Save retains the immutable original/candidate and manifest privately; candidate IDs bind the exact source path and both byte digests. Apply refuses a changed source or damaged candidate, creates a durable exact-original backup, and replaces only that file. Source permissions are preserved. Symlink components, non-Markdown files, special permissions and inputs over 4 MiB are refused. A backup path is printed after success; retain it until satisfied with the result. Inspect it and restore its exact bytes explicitly if rollback is needed.

Private candidates/backups contain plan text. They are outside Git, but this version does not qualify operating-system backup exclusion or cloud-sync exclusion: choose a local private location with the required backup policy. Do not put it in a synced directory. Cooperating repair commands serialize; unrelated editors do not take the repair lock. Close/save the source editor before applying; identity/digest are rechecked immediately before replacement, but no portable filesystem primitive provides atomic comparison-and-swap against every unrelated editor.

`--ai` fails visibly disabled because EXPLABS API/authentication/privacy/model behavior is unqualified. No provider request or dotenv read occurs. `--interactive` likewise explains that owner-value proposals are unavailable: edit the reported fields and preview again. These boundaries do not prevent deterministic preview/save/apply.
