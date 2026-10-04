const definitions = [
  ['T1.1', 'Map the opportunity', 'preflight', 'complete', [], 'A clear product brief with a focused first user journey.'],
  ['T1.2', 'Define the experience', 'preflight', 'complete', ['T1.1'], 'Design principles and the core interactions are documented.'],
  ['T1.3', 'Qualify the stack', 'preflight', 'complete', ['T1.1'], 'Local dependencies and preview tools are available.'],
  ['T2.1', 'Build the plan reader', 'implement', 'complete', ['T1.2', 'T1.3'], 'Markdown tasks retain their IDs, dependencies and acceptance criteria.'],
  ['T2.2', 'Create the spatial canvas', 'implement', 'active', ['T1.2'], 'Tasks occupy distinct lanes in an interactive 3D scene.'],
  ['T2.3', 'Connect the task inspector', 'implement', 'pending', ['T2.1'], 'Selecting a task reveals its source and acceptance criteria.'],
  ['T2.4', 'Add workspace navigation', 'implement', 'pending', ['T2.1'], 'Switch projects without losing the plan context.'],
  ['T3.1', 'Verify plan fidelity', 'verify', 'complete', ['T2.1'], 'Parser checks cover stages, dependencies and source locations.'],
  ['T3.2', 'Test the journey', 'verify', 'blocked', ['T2.2', 'T2.3'], 'The core journey works with mouse, keyboard and touch.'],
  ['T3.3', 'Check responsive layouts', 'verify', 'blocked', ['T2.2'], 'Persistent controls stay accessible at narrow viewports.'],
  ['T3.4', 'Measure render performance', 'verify', 'pending', ['T2.2'], 'The scene remains responsive during navigation.'],
  ['T4.1', 'Review the architecture', 'review', 'blocked', ['T3.1', 'T3.2'], 'An independent reviewer inspects the implementation.'],
  ['T4.2', 'Review the visual details', 'review', 'pending', ['T3.3'], 'Typography, layout and all primary states pass visual QA.'],
  ['T4.3', 'Resolve review findings', 'review', 'pending', ['T4.1'], 'Accepted blocking findings have been resolved and rechecked.'],
  ['T5.1', 'Integrate the candidate', 'merge', 'blocked', ['T4.3'], 'The reviewed revision is integrated into the target branch.'],
  ['T5.2', 'Verify the landed plan', 'verify-landed', 'pending', ['T5.1'], 'The integrated tree matches the accepted candidate.'],
  ['T5.3', 'Open the observatory', 'verify-landed', 'pending', ['T5.2'], 'A running local preview is available for exploration.'],
];
export const sampleProject = {
  id: 'sample', name: 'Wazi', sample: true,
  plans: [{id: 'sample-plan', title: 'Plan observatory', path: 'Example plan · illustrative statuses', epics: [{id: 'E1', title: 'From idea to orbit'}], warnings: [],
    tasks: definitions.map(([id,title,stage,status,dependencies,acceptance], i)=>({id,title,stage,status,dependencies,acceptance,epicId:'E1',epicTitle:'From idea to orbit',owner: ['Product','Engineering','Design'][i%3],line:i+1,source:''})),
  }],
};
export const lanes = [
  { id:'preflight', title:'Discover', subtitle:'Frame the work', color:'#70dabd' },
  { id:'implement', title:'Build', subtitle:'Make it real', color:'#929dff' },
  { id:'verify', title:'Verify', subtitle:'Prove it works', color:'#69cfe5' },
  { id:'review', title:'Review', subtitle:'A second perspective', color:'#d4a0de' },
  { id:'land', title:'Land', subtitle:'Bring it together', color:'#ecbc85' },
  { id:'other', title:'Other', subtitle:'Unsupported or unstaged', color:'#9aa6b2' },
];
export const laneFor = task => ['merge','verify-landed'].includes(task.stage) ? 'land' : ['preflight','implement','verify','review'].includes(task.stage) ? task.stage : 'other';
export const statusLabel = {complete:'Complete',active:'In progress',blocked:'Blocked',pending:'Planned'};
