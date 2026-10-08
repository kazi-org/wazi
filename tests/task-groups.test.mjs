import test from 'node:test';
import assert from 'node:assert/strict';
import { groupTasks, limitTaskGroups, matchesStatusFilter } from '../src/task-groups.mjs';
test('milestone-staged tasks remain visible under reported statuses without mutation', () => {
  const tasks = ['pending','active','blocked','complete','future-status'].map((status,i)=>({id:String(i),status,stage:'S0'}));
  const before = JSON.stringify(tasks);
  const groups = groupTasks(tasks);
  assert.deepEqual(groups.map(g=>g.tasks.map(t=>t.id)), [['0'],['1'],['2'],['3'],['4']]);
  assert.equal(JSON.stringify(tasks),before);
  assert.equal(groups.find(g=>g.id==='complete').title,'Marked done');
  assert.equal(groupTasks(tasks,'stage').find(g=>g.id==='other').tasks.length,5);
});
test('delivery stages retain merge and landed verification mapping', () => {
  const tasks = ['implement','merge','verify-landed'].map(stage=>({stage,status:'active'}));
  const groups=groupTasks(tasks,'stage');
  assert.equal(groups.find(g=>g.id==='implement').tasks.length,1);
  assert.equal(groups.find(g=>g.id==='land').tasks.length,2);
  assert.equal(groupTasks([]).length,4);
});

test('column limits report the full count and keep a selected matching task reachable', () => {
  const tasks=Array.from({length:9},(_,i)=>({id:'t'+i,status:'active'}));
  const group=groupTasks(tasks).find(item=>item.id==="active");
  const capped=limitTaskGroups([group],'5','t8')[0];
  assert.equal(capped.total,9);
  assert.equal(capped.showing,5);
  assert.deepEqual(capped.tasks.map(task=>task.id),['t0','t1','t2','t3','t8']);
  assert.equal(limitTaskGroups([group],'all')[0].showing,9);
});

test('status filtering hides only nonmatching statuses while preserving unknown access', () => {
  assert.equal(matchesStatusFilter({status:'active'},'active'),true);
  assert.equal(matchesStatusFilter({status:'pending'},'active'),false);
  assert.equal(matchesStatusFilter({status:'vendor-state'},'unknown'),true);
  assert.equal(matchesStatusFilter({status:'blocked'},'unknown'),false);
  assert.equal(matchesStatusFilter({status:'vendor-state'},'all'),true);
});
