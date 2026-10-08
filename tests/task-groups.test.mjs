import test from 'node:test';
import assert from 'node:assert/strict';
import { groupTasks } from '../src/task-groups.mjs';
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
