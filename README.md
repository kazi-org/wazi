# Wazi

A local, read-only 3D observatory for Markdown project plans. Inspired by the column-and-dependency structure in `tmp/idea.png`, with floating task cards, a spatial grid, five delivery lanes and an acceptance inspector.

## Run

```sh
npm install
npm run dev -- --port 4193 --strictPort
```

Open http://127.0.0.1:4193/. The development server reads local plans. The static build contains the sample workflow and supports Markdown import; it does not include your local project data.

The default Three.js source is `~/Code/three.js`, using its existing `build/three.module.js` and `examples/jsm/controls/OrbitControls.js`. Override with `THREE_JS_SOURCE=/path/to/three.js`. This checkout is consumed without modification. Fonts and [Phosphor](https://phosphoricons.com/) icons are installed locally; the browser makes no external asset requests.

## Explore

- Select a local project in the sidebar, then choose its plan and epic.
- Drag the canvas to orbit; right-drag to pan; scroll or pinch to zoom. Map view disables rotation.
- Select a task to see its acceptance criteria, source line, dependencies and the work it unlocks. Use Focus in space to move closer; R resets the camera.
- Status controls emphasize matching tasks. The dependency toggle controls the connection paths.
- Import a Markdown file or paste its text. Imports remain in memory for this browser session.
- Use the task list to inspect every task, especially on a small screen. The scene shows up to four tasks per lane (20 total); selecting another task brings it into the scene. Cross-file dependencies outside the selected plan are recorded but are not drawn as phantom nodes.

## Plan discovery

The server scans repository roots under `~/Code` for `plan.md`, `docs/plan.md` and `docs/plans/*.md`. Override the root with `WAZI_PLAN_ROOT=/path/to/workspace`. Discovery stops at Git roots, skips common build/dependency/worktree directories and bounds file sizes, depth and counts. Repositories are sorted. A visible notice reports scan limits or read failures; unsupported metadata is not fabricated.

Tasks support checkboxes (`[ ]`, `[x]`, `[~]`, `[-]`), stable IDs (`T1.1`, `S1.2.1`), `Owner:`, `stage:`, `status:`, `acc:`, `deps:` and `blocked-by:`. An incomplete known dependency derives a blocked status; unresolved dependencies produce warnings. Checkboxes remain the source for completed status. Import does not rewrite source plans.

The API is read only and loopback-only, rejects cross-site browser requests and does not cache or bundle plan data. A local development prototype is not a hosted service.

## Check

```sh
npm test
npm run build
```

For this session, dependencies, caches, build output and screenshots are on the external SSD. The implementation and parser tests are tracked in this repository; private plan snapshots are not tracked. Verification details live in `design-qa.md` and `docs/devlog.md`.
