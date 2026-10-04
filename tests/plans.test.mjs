import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import { parsePlan, scanPlans } from '../scripts/plans.mjs';

test('parsePlan extracts skill checkbox fields, epics and stable source identity', () => {
  const markdown = `# E2 — Build the observatory

## Tasks
- [x] T2.0 Qualify contracts  Owner: coordinator  kind: agent stage: preflight  acc: [reference inspected; parser contract agreed]
- [ ] T2.1 Implement scanner  Owner: plan-reader  kind: agent stage: implement  blocked-by: [T2.0]  acc: [Plans parse; source stays unchanged]
`;
  const first = parsePlan(markdown, { path: 'docs/plans/E2.md', project: 'wazi' });
  const second = parsePlan(markdown, { path: 'docs/plans/E2.md', project: 'wazi' });
  assert.equal(first.title, 'E2 — Build the observatory');
  assert.deepEqual(first.epics, [{ id: 'E2', title: 'Build the observatory' }]);
  assert.equal(first.tasks.length, 2);
  assert.deepEqual(first.tasks.map((task) => task.status), ['complete', 'pending']);
  assert.equal(first.tasks[1].stage, 'implement');
  assert.equal(first.tasks[1].owner, 'plan-reader');
  assert.equal(first.tasks[1].acceptance, 'Plans parse; source stays unchanged');
  assert.deepEqual(first.tasks[1].dependencies, ['T2.0']);
  assert.equal(first.tasks[1].source, 'docs/plans/E2.md:5');
  assert.equal(first.tasks[1].id, second.tasks[1].id);
});

test('same task IDs in separate files keep separate identities', () => {
  const body = '# E1 — Epic\n- [ ] T1.1 Same visible ID  Owner: team';
  const master = parsePlan(body, { path: 'docs/plan.md', project: 'repo' });
  const split = parsePlan(body, { path: 'docs/plans/E1.md', project: 'repo' });
  assert.equal(master.tasks[0].sourceId, split.tasks[0].sourceId);
  assert.notEqual(master.id, split.id);
  assert.notEqual(master.tasks[0].id, split.tasks[0].id);
});

test('ignores checkbox examples in fenced code blocks', () => {
  const markdown = `# Example plan

~~~markdown
- [ ] T9.9 This is only an example  Owner: nobody
~~~

- [ ] T1.1 Real work  Owner: team`;
  const parsed = parsePlan(markdown, { path: 'plan.md' });
  assert.equal(parsed.tasks.length, 1);
  assert.equal(parsed.tasks[0].sourceId, 'T1.1');
});

test('keeps pending status for unresolved dependency and emits a warning', async (t) => {
  const root = await fs.mkdtemp(path.join(os.tmpdir(), 'wazi-plans-'));
  t.after(() => fs.rm(root, { recursive: true, force: true }));
  const repo = path.join(root, 'repo');
  await fs.mkdir(path.join(repo, 'docs'), { recursive: true });
  await fs.writeFile(path.join(repo, 'docs', 'plan.md'), '# Project\n- [ ] T1.1 Needs unknown  blocked-by: [T8.8]');
  const result = await scanPlans({ root });
  const project = result.projects.find((entry) => entry.name === 'repo');
  const task = project.plans[0].tasks[0];
  assert.equal(task.status, 'pending');
  assert.match(project.plans[0].warnings.join('\n'), /Unresolved dependency T8\.8/);
});

test('indexes split plans and resolves cross-file dependency status', async (t) => {
  const root = await fs.mkdtemp(path.join(os.tmpdir(), 'wazi-plans-'));
  t.after(() => fs.rm(root, { recursive: true, force: true }));
  const repo = path.join(root, 'repo');
  await fs.mkdir(path.join(repo, 'docs', 'plans'), { recursive: true });
  await fs.writeFile(path.join(repo, 'docs', 'plan.md'), '# Project plan\n- [ ] T1.1 Master task');
  await fs.writeFile(path.join(repo, 'docs', 'plans', 'E1.md'), '# E1 — Foundation\n- [x] T1.1 Repeated split ID\n- [ ] T1.2 Follow-up  blocked-by: [T1.1]');
  await fs.writeFile(path.join(repo, 'docs', 'plans', 'readme.txt'), '- [ ] T4.1 Ignored');
  const result = await scanPlans({ root });
  const project = result.projects.find((entry) => entry.name === 'repo');
  assert.equal(project.plans.length, 2);
  const [master, split] = project.plans;
  assert.notEqual(master.tasks[0].id, split.tasks[0].id);
  assert.equal(split.tasks[1].status, 'blocked');
  assert.deepEqual(project.plans.flatMap((plan) => plan.tasks).filter((task) => task.sourceId === 'T1.1').map((task) => task.status), ['pending', 'complete']);
});

test('preserves multiline acceptance and explicit active and blocked task states', () => {
  const markdown = `# E1 — States
- [ ] T1.1 Active work  status: active  acc: [first line; second line]
- [ ] T1.2 Blocked work  status: blocked  Owner: team`;
  const parsed = parsePlan(markdown, { path: 'plan.md', project: 'sample' });
  assert.deepEqual(parsed.tasks.map((task) => task.status), ['active', 'blocked']);
  assert.equal(parsed.tasks[0].acceptance, 'first line; second line');
});

test('parses single-space metadata and strips all trailing metadata from the task title', () => {
  const parsed = parsePlan(`# Metadata
- [ ] T3.4 Wire a capability gate kind: agent stage: implement blocked-by: [T3.3] acc: [acceptance stays in its field]`, { path: 'plan.md' });
  const [task] = parsed.tasks;
  assert.equal(task.title, 'Wire a capability gate');
  assert.equal(task.stage, 'implement');
  assert.deepEqual(task.dependencies, ['T3.3']);
  assert.equal(task.acceptance, 'acceptance stays in its field');
});

test('maps alternate checkbox markers and retains title when there is no task ID', () => {
  const parsed = parsePlan(`# Legacy syntax
- [~] Document owner hand-off stage: review
- [-] Investigate current failure
- [ ] Preserve the first word`, { path: 'plan.md' });
  assert.deepEqual(parsed.tasks.map((task) => task.status), ['active', 'blocked', 'pending']);
  assert.deepEqual(parsed.tasks.map((task) => task.title), ['Document owner hand-off', 'Investigate current failure', 'Preserve the first word']);
  assert.equal(parsed.tasks[0].sourceId, 'line-2');
});

test('acceptance brackets preserve apostrophes in ordinary prose', () => {
  const parsed = parsePlan(`# Acceptance
- [ ] T1.1 Check user's task acc: [The user's task remains visible and its status is clear.]`, { path: 'plan.md' });
  assert.equal(parsed.tasks[0].acceptance, "The user's task remains visible and its status is clear.");
});

test('scanner stops at git roots and does not invent projects for docs directories', async (t) => {
  const root = await fs.mkdtemp(path.join(os.tmpdir(), 'wazi-plan-discovery-'));
  t.after(() => fs.rm(root, { recursive: true, force: true }));
  const repo = path.join(root, 'org', 'actual-repo');
  const worktree = path.join(root, 'org', 'actual-repo-wt-task-T3.4');
  const nested = path.join(repo, 'docs', 'examples', 'nested');
  await fs.mkdir(path.join(repo, '.git'), { recursive: true });
  await fs.mkdir(path.join(repo, 'docs'), { recursive: true });
  await fs.mkdir(path.join(nested, 'docs'), { recursive: true });
  await fs.mkdir(path.join(worktree, 'docs'), { recursive: true });
  await fs.writeFile(path.join(worktree, '.git'), 'gitdir: ../actual-repo/.git/worktrees/task\n');
  await fs.writeFile(path.join(repo, 'docs', 'plan.md'), '# Real project\n- [ ] T1.1 A real task');
  await fs.writeFile(path.join(nested, 'docs', 'plan.md'), '# Not a project\n- [ ] T9.9 An example');
  await fs.writeFile(path.join(nested, 'docs', 'plan.md'), '# Not a project\n- [ ] T9.9 An example');
  await fs.writeFile(path.join(worktree, 'docs', 'plan.md'), '# Duplicate worktree copy\n- [ ] T1.1 Duplicate');
  const result = await scanPlans({ root });
  assert.deepEqual(result.projects.map((project) => project.name), ['actual-repo']);
  assert.equal(result.projects[0].plans.length, 1);
});
