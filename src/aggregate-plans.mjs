const planScope = plan => encodeURIComponent(String(plan.path || plan.id));

export function qualifiedTaskId(plan, task) {
  const authoredId = task.sourceId || task.id;
  return `plan:${planScope(plan)}::${encodeURIComponent(String(authoredId))}`;
}

export function qualifiedEpicId(plan, epicId) {
  return `plan:${planScope(plan)}::epic:${encodeURIComponent(String(epicId))}`;
}

/** Aggregate tasks without losing their authored records or crossing plan edges. */
export function aggregatePlanTasks(plans, selectedPlanId = 'all') {
  const selectedPlans = selectedPlanId === 'all'
    ? plans
    : plans.filter(plan => plan.id === selectedPlanId);
  return selectedPlans.flatMap(plan => {
    const byAuthoredId = new Map(plan.tasks.map(task => [
      task.sourceId || task.id,
      qualifiedTaskId(plan, task),
    ]));
    return plan.tasks.map(task => ({
      ...task,
      id: qualifiedTaskId(plan, task),
      dependencies: task.dependencies.map(id => byAuthoredId.get(id) || id),
      originalDependencies: [...task.dependencies],
      authoredId: task.sourceId || task.id,
      authoredEpicId: task.epicId,
      epicScopeId: task.epicId ? qualifiedEpicId(plan, task.epicId) : '',
      planId: plan.id,
      planPath: plan.path,
      planTitle: plan.title,
    planLabel: plan.path ? plan.title + ' · ' + plan.path.split('/').slice(-2).join('/') : plan.title,
      planLabel: plan.path ? plan.title + ' · ' + plan.path.split('/').slice(-2).join('/') : plan.title,
      originalTask: task,
    }));
  });
}

export function aggregatePlanEpics(plans, selectedPlanId = 'all') {
  const selectedPlans = selectedPlanId === 'all'
    ? plans
    : plans.filter(plan => plan.id === selectedPlanId);
  return selectedPlans.flatMap(plan => (plan.epics || []).map(epic => ({
    ...epic,
    id: selectedPlanId === 'all' ? qualifiedEpicId(plan, epic.id) : epic.id,
    authoredId: epic.id,
    planId: plan.id,
    planTitle: plan.title,
    planLabel: plan.path ? plan.title + ' · ' + plan.path.split('/').slice(-2).join('/') : plan.title,
  })));
}
