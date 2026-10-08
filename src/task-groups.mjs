import { lanes, laneFor } from './demo.mjs';

const statusLanes = [
  { id:'pending', title:'Planned', subtitle:'Not marked started', color:'#9aa6b2' },
  { id:'active', title:'In progress', subtitle:'Work reported active', color:'#929dff' },
  { id:'blocked', title:'Blocked', subtitle:'Waiting on dependencies', color:'#ecbc85' },
  { id:'complete', title:'Marked done', subtitle:'Reported by the source plan', color:'#70dabd' },
  { id:'unknown', title:'Unknown status', subtitle:'Unrecognized source status', color:'#d4a0de' },
];
export function groupTasks(tasks, grouping = 'status') {
  const byStage = grouping === 'stage';
  const definitions = byStage ? lanes : statusLanes;
  const idFor = byStage ? laneFor : task =>
    statusLanes.some(lane => lane.id === task.status) ? task.status : 'unknown';
  return definitions.map(lane => ({...lane, tasks:tasks.filter(task => idFor(task) === lane.id)}))
    .filter(lane => byStage || lane.id !== 'unknown' || lane.tasks.length);
}
