import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import { parsePlan, scanPlans } from '../scripts/plans.mjs';
import { laneFor } from '../src/demo.mjs';
import { looksLikePortableJson, portablePlanFromBundle, PORTABLE_CONTRACT_DIGEST, PORTABLE_CONTRACT_VERSION } from '../src/portable-plan.mjs';

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

test('preserves contract metadata, unsupported stages, UUID identity and authored status separately', () => {
  const uuid = '3f6f6e8a-46b0-4f05-9854-4d7e8f4d16cc';
  const authoredId = uuid.toUpperCase();
  const sourceBlock = `- [x] ${authoredId} Inspect portable evidence  kind: agent stage: rereview provider: aprl canonical-id: ${uuid} policy-revision: review-v2 custom-hint: keep this\n  Acceptance: First criterion.\n    Second criterion stays on its own line.`;
  const [task] = parsePlan(sourceBlock, { path: 'docs/plan.md', project: 'repo' }).tasks;

  assert.equal(task.sourceId, authoredId);
  assert.equal(task.canonicalId, uuid);
  assert.equal(task.authoredStatus, 'checked');
  assert.equal(task.status, 'complete');
  assert.equal(task.stage, 'rereview');
  assert.equal(task.metadata.provider, 'aprl');
  assert.equal(task.metadata['policy-revision'], 'review-v2');
  assert.match(task.sourceBlock, /custom-hint: keep this/);
  assert.equal(task.acceptance, 'First criterion.\nSecond criterion stays on its own line.');
  assert.equal(laneFor(task), 'other');
});

test('keeps punctuation and opaque multiline acceptance intact beside metadata', () => {
  const [task] = parsePlan(`# Opaque acceptance
- [ ] T7.3 Preserve criteria  stage: implement
  Acceptance: A sentence with punctuation.
    A second line with: authored wording.
  provider: local`, { path: 'docs/plan.md' }).tasks;
  assert.equal(task.acceptance, 'A sentence with punctuation.\nA second line with: authored wording.');
});

test('projects frozen portable v0.0.1 without trusting reported qualification or losing source data', async () => {
  const bundle = JSON.parse(await fs.readFile(new URL('./fixtures/portable-display.json', import.meta.url), 'utf8'));
  const manifest = JSON.parse(await fs.readFile(new URL('../contracts/plan/v0/manifest.json', import.meta.url), 'utf8'));
  const project = portablePlanFromBundle(bundle, 'portable-display.json');
  const [plan] = project.plans;
  const [task] = plan.tasks;

  assert.equal(PORTABLE_CONTRACT_VERSION, '0.0.1');
  assert.equal(PORTABLE_CONTRACT_DIGEST, 'sha256:7582512f122d2f2a9c4461facc7541c9887053f137260d6ebe9c6dea611d039d');
  assert.equal(manifest.contractDigest, PORTABLE_CONTRACT_DIGEST);
  assert.equal(plan.id, bundle.definition.id);
  assert.equal(task.id, bundle.definition.tasks[0].id);
  assert.equal(task.stage, 'rereview');
  assert.equal(laneFor(task), 'other');
  assert.equal(task.authoredStatus, 'pending');
  assert.equal(task.authoredStatusLabel, 'pending');
  assert.equal(task.status, 'pending');
  assert.equal(task.acceptance, bundle.definition.tasks[0].acceptance);
  assert.deepEqual(task.source, bundle.definition.tasks[0].source);
  assert.deepEqual(task.metadata, bundle.definition.tasks[0].metadata);
  assert.equal(project.portableBundle, bundle);
  assert.equal(task.reportedEvidence[0].trust, 'verified');
  assert.equal(task.reportedEvaluations[0].qualification, 'verified');
  assert.match(plan.warnings.join(' '), /unverified by Wazi/i);
  const authoredComplete = structuredClone(bundle);
  authoredComplete.definition.tasks[0].authoredStatus = 'complete';
  const [completeTask] = portablePlanFromBundle(authoredComplete).plans[0].tasks;
  assert.equal(completeTask.status, 'complete');
  assert.equal(completeTask.authoredStatusLabel, 'Marked done');
  assert.equal(looksLikePortableJson('  {"contractVersion":"0.0.1"}'), true);
  assert.equal(looksLikePortableJson('# markdown'), false);
});

test('accepts every frozen valid contract fixture for display projection', async () => {
  const fixtureDirectory = new URL('../contracts/plan/v0/fixtures/valid/', import.meta.url);
  const names = await fs.readdir(fixtureDirectory);
  for (const name of names.filter((entry) => entry.endsWith('.json'))) {
    const bundle = JSON.parse(await fs.readFile(new URL(name, fixtureDirectory), 'utf8'));
    assert.doesNotThrow(() => portablePlanFromBundle(bundle, name), name);
  }
});

test('rejects unsupported versions and malformed portable core shapes', async () => {
  const fixture = JSON.parse(await fs.readFile(new URL('./fixtures/portable-display.json', import.meta.url), 'utf8'));
  assert.throws(() => portablePlanFromBundle({ ...fixture, contractVersion: '0.0.2' }), /Unsupported portable plan contract version/);
  assert.throws(() => portablePlanFromBundle({ ...fixture, extra: true }), /not defined by contract/);
  const missingAcceptance = structuredClone(fixture);
  delete missingAcceptance.definition.tasks[0].acceptance;
  assert.throws(() => portablePlanFromBundle(missingAcceptance), /acceptance is required/);
  const missingMapping = structuredClone(fixture);
  delete missingMapping.definition.tasks[0].source.line;
  assert.throws(() => portablePlanFromBundle(missingMapping), /line is required for Markdown/);
  const wrongRef = structuredClone(fixture);
  wrongRef.definition.tasks[0].source.ref = 'other.md';
  assert.throws(() => portablePlanFromBundle(wrongRef), /must match the plan source ref/);
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
