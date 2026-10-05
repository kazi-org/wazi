import test from 'node:test';
import assert from 'node:assert/strict';
import { parsePlan } from '../src/plan-parser.mjs';

function authoredProjection(markdown) {
  const plan = parsePlan(markdown, { path: 'docs/plans/E11.md', project: 'wazi-repair-test' });
  return plan.tasks.map((task) => ({
    sourceId: task.sourceId,
    title: task.title,
    status: task.status,
    authoredStatus: task.authoredStatus,
    owner: task.owner,
    stage: task.stage,
    dependencies: task.dependencies,
    acceptance: task.acceptance,
    metadata: task.metadata,
  }));
}

test('recognized list marker and checkbox case normalization preserves the authored plan projection', () => {
  const source = `# E11 — Deterministic repair

<!-- Keep comments and fenced examples outside the task projection. -->

* [X] T11.2 Preserve reviewed source  Owner: coordinator  stage: rereview  blocked-by: [T11.0]  acc: [First criterion.
  Second criterion remains exact, with punctuation.]
* [ ] T11.3 Report unsupported profile  Owner: repair-worker  stage: future-stage  deps: [T11.2]
+ [~] T11.4 Preserve active checkbox
* [-] T11.5 Preserve blocked checkbox

\`\`\`markdown
- [ ] T99.9 Example only
\`\`\`
`;
  const candidate = source
    .replace('* [X] T11.2', '- [x] T11.2')
    .replace('* [ ] T11.3', '- [ ] T11.3')
    .replace('+ [~] T11.4', '- [~] T11.4')
    .replace('* [-] T11.5', '- [-] T11.5');

  const before = authoredProjection(source);
  const after = authoredProjection(candidate);

  assert.deepEqual(after, before);
  assert.deepEqual(after.map((task) => task.sourceId), ['T11.2', 'T11.3', 'T11.4', 'T11.5']);
  assert.deepEqual(after.map((task) => task.authoredStatus), [
    'checked',
    'unchecked',
    'in-progress-marker',
    'blocked-marker',
  ]);
  assert.deepEqual(after.map((task) => task.status), ['complete', 'pending', 'active', 'blocked']);
  assert.equal(after[0].owner, 'coordinator');
  assert.equal(after[0].stage, 'rereview');
  assert.deepEqual(after[0].dependencies, ['T11.0']);
  assert.equal(after[0].acceptance, 'First criterion.\nSecond criterion remains exact, with punctuation.');
  assert.equal(after[1].stage, 'future-stage');
  assert.deepEqual(after[1].dependencies, ['T11.2']);
  assert.equal(after.length, 4, 'the fenced task example must not enter the projection');
});
