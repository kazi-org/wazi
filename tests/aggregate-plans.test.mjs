import test from 'node:test';
import assert from 'node:assert/strict';
import {aggregatePlanEpics, aggregatePlanTasks, qualifiedTaskId} from '../src/aggregate-plans.mjs';

const plan = (id, path, tasks, epics = []) => ({id, path, title:id, tasks, epics});
const task = (id, dependencies = [], epicId = 'E1') => ({id, sourceId:id, title:id, dependencies, epicId});

test('all-plan aggregation qualifies repeated authored task IDs by plan path', () => {
  const plans = [
    plan('a', 'docs/plan.md', [task('T1.0')]),
    plan('b', 'docs/plans/next.md', [task('T1.0')]),
  ];
  const tasks = aggregatePlanTasks(plans);
  assert.equal(tasks.length, 2);
  assert.notEqual(tasks[0].id, tasks[1].id);
  assert.equal(tasks[0].id, qualifiedTaskId(plans[0], plans[0].tasks[0]));
  assert.equal(tasks[0].authoredId, 'T1.0');
  assert.equal(tasks[0].originalTask, plans[0].tasks[0]);
});

test('dependencies resolve only within their source plan', () => {
  const plans = [
    plan('a', 'docs/a.md', [task('T1.0'), task('T1.1', ['T1.0', 'T9.9'])]),
    plan('b', 'docs/b.md', [task('T1.0')]),
  ];
  const tasks = aggregatePlanTasks(plans);
  assert.equal(tasks[1].dependencies[0], tasks[0].id);
  assert.equal(tasks[1].dependencies[1], 'T9.9');
  assert.notEqual(tasks[1].dependencies[0], tasks[2].id);
  assert.deepEqual(tasks[1].originalDependencies, ['T1.0', 'T9.9']);
});

test('plan selection is reversible and same-named epics stay separately scoped', () => {
  const plans = [
    plan('a', 'docs/a.md', [task('T1.0')], [{id:'E1', title:'Build'}]),
    plan('b', 'docs/b.md', [task('T1.0')], [{id:'E1', title:'Build'}]),
  ];
  assert.equal(aggregatePlanTasks(plans, 'b').length, 1);
  const epics = aggregatePlanEpics(plans);
  assert.equal(epics.length, 2);
  assert.notEqual(epics[0].id, epics[1].id);
  assert.equal(aggregatePlanEpics(plans, 'a')[0].id, 'E1');
});

test('all-plan aggregation retains a 144-task, 15-plan workspace', () => {
  const plans = Array.from({length:15}, (_, planIndex) => plan(
    'p'+planIndex,
    'docs/plans/p'+planIndex+'.md',
    Array.from({length:planIndex<9?10:9}, (_, taskIndex) => task('T'+taskIndex+'.1')),
  ));
  const tasks = aggregatePlanTasks(plans);
  assert.equal(tasks.length, 144);
  assert.equal(new Set(tasks.map(item => item.id)).size, 144);
  assert.equal(tasks.filter(item => item.authoredId === 'T0.1').length, 15);
});
