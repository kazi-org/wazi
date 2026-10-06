# Local operation and troubleshooting

This guide describes the implementation at `8c5486d`. It is a local development prototype; static output supports the sample and imports rather than local filesystem discovery.

## Start the local app

From the repository root, with Node/npm available and a Three.js checkout containing the built modules and examples:

```sh
npm install
THREE_JS_SOURCE=/path/to/three.js WAZI_PLAN_ROOT=/path/to/workspace npm run dev -- --port 4193 --strictPort
```

Open the local URL printed by Vite. Both environment variables are optional in the owner's current environment. `THREE_JS_SOURCE` defaults to the owner's existing checkout; `WAZI_PLAN_ROOT` defaults to the `Code` directory under the current user's home. Set them explicitly for portability. The `server.fs.allow` configuration permits the application root and Three.js source root for Vite module serving; it is distinct from the plan scanner's root.

The current agent-run preview consumes repository source from the primary checkout. Its ignored `node_modules` symlink points to the task's external-SSD dependencies, keeping npm/Vite caches outside the primary checkout. That is a session setup detail, not a tracked project requirement. Do not delete or relocate the target while the preview is using it. Follow the global external-SSD and shared-build/load rules when installing or building during agent work.

## Commands and modes

| Command | Behavior |
| --- | --- |
| `npm run dev` | Vite app and read-only plan discovery API; binds to loopback by default |
| `npm test` | Ten focused Node parser/discovery tests; temporary fixtures use the operating system temp directory |
| `npm run build` | Static application in `dist/client`; reads/bundles local Three.js and npm fonts/icons, without bundling scanned project plans |
| `npm run preview` | Serves the existing static build; does not install the development API |

For agent-run tests set `TMPDIR` to an owned external-SSD artifact directory. Do not interpret a successful build as browser verification. This documentation pass does not rerun tests or a build; the existing records are [review](review.md), [devlog](devlog.md) and [visual QA](../design-qa.md).

## Expected data behavior

The initial view is a visibly labeled sample. Pick a local project, plan and epic to explore real records. The project picker favors the first nonempty plan. Refresh starts a fresh scan and keeps imported projects in memory, although other selection state may be reset when changing project. Reload clears imports and returns to the sample. There is no continuous file watcher for plan records, database or write-back.

The scanner looks for root `plan.md`, `docs/plan.md` and immediate Markdown files in `docs/plans/`; it does not follow links from a table of contents or recursively expand arbitrary Markdown includes. A directory with plan files can be included even without a Git marker. A Git marker stops descent into the repository whether or not it has a plan. Repository directory symlinks are not walked, but candidate plan paths are checked/read with normal filesystem calls rather than a realpath containment policy.

| Bound | Value | Behavior |
| --- | --- | --- |
| Repository depth | 3 edges below the scan root | Deeper projects are not visited; no dedicated depth-limit warning |
| Eligible directory entries | 12,000 | Discovery stops and returns a top-level warning |
| Split files per project | 120 | Further split files are skipped; top-level warning |
| Discovered plan size | 1,000,000 bytes | File skipped; top-level warning |
| Browser file import size | 2,000,000 bytes | File skipped with a toast |
| Paste length | 2,000,000 text characters | Textarea input limit; this is not the file byte limit |
| Scene tasks | Four per lane, at most 20 | Full list remains available; selected task replaces a lane's last visible task |

Unreadable discovery directories can be skipped silently. Unresolved task dependencies are recorded in per-plan `warnings`, which the UI currently does not render. The toast shows only the first top-level scanner warning, or an import/error notice; it is not a complete warning report. A record count is the discovered subset rather than a guarantee of exhaustive workspace coverage.

## Troubleshooting

| Symptom | What to check |
| --- | --- |
| No local projects | Confirm the chosen scan root and supported file locations. The scan only looks three levels deep and skips dependency/build/common worktree names. Import Markdown as an alternative. |
| Static preview says local plans unavailable | Expected: `/api/plans` exists in the development server only. Use `npm run dev` for discovery or import a plan. |
| Tasks have the wrong-looking status | Completed state comes from the checkbox. Known incomplete dependencies can derive blocked state. Local scan and browser import have different derivation rules; neither executes acceptance criteria. |
| A dependency has no curve | Both endpoints must be in the visible set. The selected plan may not contain the target, or the target may be beyond the per-lane limit. Use the inspector and task list. |
| A task appears in Build unexpectedly | Missing or unknown stage values fall back to Build; stage matching is exact. |
| Acceptance text is absent | Check supported `acc:` or `Acceptance:` syntax and task block structure; the parser is a targeted reader rather than a general Markdown compiler. |
| New plan edits are not reflected | Use Refresh or reload. After changing server-side scanner/parser source during development, restart Vite to clear Node module import caching. |
| File chooser cannot be automated | Existing verification encountered an extension restriction; paste the Markdown instead. This does not establish that native file selection is broken for a human user. |
| 3D rendering unavailable | Use the task list. The WebGL failure message and list path exist, but no-WebGL behavior has not been browser-qualified. |
| Unexpected results after rapid refreshes | Requests are not cancelled or sequence-guarded; an earlier scan can resolve after a later one. Retry after scans settle. |

## Data handling and scope

The server endpoint is GET-only, checks loopback Host values, rejects cross-site fetch metadata or mismatched Origin and uses `Cache-Control: no-store`. There are no outbound data requests in the application code. The browser receives task text and source locations, so the JSON response can contain private workspace material. The local guard is not an authentication system or a reason to expose the Vite server publicly. Do not override its bind address for sharing without a separate hosting/access design.

Do not commit scan responses, copied private plan bodies or generated browser screenshots. The project `ajent.social` channel remains ignored; it must contain no secrets, home paths, hostnames, private IPs or customer names. Message posting is explicit coordination, not proof the recipient has read a message.

## AI repair CLI operation (candidate)

Deterministic preview: `wazi repair FILE.md`. To explicitly request an EXPLABS proposal: `wazi repair --ai --env-file /path/to/wazi/.env FILE.md`. Place flags before the selected file. From the Wazi checkout, `--env-file` may be omitted; unrelated working directories do not discover project credentials. Only EXPLABS_API_KEY, EXPLABS_BASE_URL and EXPLABS_MODEL are used, with process values taking precedence. The dotenv file must be an owned regular owner-only file.

Add `--save-candidate` to retain an applicable proposal. Review the printed diff, then use the printed separate `--apply-candidate` command. AI never applies a response automatically. Credential-looking source text, semantic invention, ambiguous changes and truncated responses are refused. Source identity/digest changes prevent apply; an exact-original private backup precedes atomic replacement.

AI preview persists private request state, unlike zero-write deterministic preview. Eligible exact completed inputs reuse for 24 hours. Unknown/failed requests never automatically resend, and missing/corrupt cached bodies fail closed. Capacity exhaustion refuses dispatch. Provider charges and account-dependent retention may apply; Wazi does not alter provider privacy or billing settings. Request storage is owner-only with best-effort Time Machine exclusion; other backups/sync and owner-explicit candidate/backup retention are not covered.

For Mac search, the usual Spotlight shortcut is Command-Space. The installed application uses its Wazi logo. If registration/import succeed but Spotlight metadata remains absent and mdutil reports a read-only index, do not rewrite the application bundle or bypass its signature: qualify the system index and actual search separately.
