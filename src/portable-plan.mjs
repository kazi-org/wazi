export const PORTABLE_CONTRACT_VERSION = '0.0.1';
export const PORTABLE_CONTRACT_DIGEST = 'sha256:7582512f122d2f2a9c4461facc7541c9887053f137260d6ebe9c6dea611d039d';

const ID = /^[A-Za-z0-9][A-Za-z0-9._:/@+-]*$/;
const DIGEST = /^sha256:[0-9a-f]{64}$/;
const predicates = new Set(['execution-complete', 'checks-satisfied', 'independent-approved', 'landing-verified', 'deployment-verified', 'domain-accepted']);
const authoredStatuses = new Set(['pending', 'active', 'blocked', 'complete']);
const executionStates = new Set(['waiting', 'active', 'complete', 'canceled', 'expired', 'unknown']);
const evidenceKinds = new Set(['execution', 'check', 'review', 'landing', 'deployment', 'domain', 'context', 'policy-approval']);
const evidenceResults = new Set(['passed', 'failed', 'missing', 'unverified', 'stale', 'rejected', 'unknown', 'canceled', 'expired', 'unsupported', 'unavailable']);
const evaluationOutcomes = new Set(['satisfied', 'unsatisfied', 'unknown', 'unavailable', 'stale', 'unsupported']);

function object(value, path) {
  if (!value || typeof value !== 'object' || Array.isArray(value)) throw new Error(`${path} must be an object.`);
  return value;
}

function string(value, path, pattern = null) {
  if (typeof value !== 'string' || !value.trim() || (pattern && !pattern.test(value))) throw new Error(`${path} must be a non-empty valid string.`);
  return value;
}

function array(value, path) {
  if (!Array.isArray(value)) throw new Error(`${path} must be an array.`);
  return value;
}

function exactKeys(value, allowed, path) {
  const extra = Object.keys(value).find((key) => !allowed.has(key));
  if (extra) throw new Error(`${path}.${extra} is not defined by contract ${PORTABLE_CONTRACT_VERSION}.`);
}

function requireKeys(value, names, path) {
  const missing = names.find((name) => !(name in value));
  if (missing) throw new Error(`${path}.${missing} is required.`);
}

function validateSource(source, ref, authority, path) {
  object(source, path);
  exactKeys(source, new Set(['ref', 'line', 'canonicalId', 'raw']), path);
  string(source.ref, `${path}.ref`);
  if (source.ref !== ref) throw new Error(`${path}.ref must match the plan source ref.`);
  if (source.line !== undefined && (!Number.isInteger(source.line) || source.line < 1)) throw new Error(`${path}.line must be a positive integer.`);
  if (source.canonicalId !== undefined) string(source.canonicalId, `${path}.canonicalId`, ID);
  if (source.raw !== undefined && (typeof source.raw !== 'string' || !source.raw)) throw new Error(`${path}.raw must be a non-empty string when present.`);
  if (authority === 'markdown' && source.line === undefined) throw new Error(`${path}.line is required for Markdown source mapping.`);
  if (authority === 'native' && source.canonicalId === undefined) throw new Error(`${path}.canonicalId is required for native source mapping.`);
}

function validateReportedEvidence(evidence, taskIds) {
  const evidenceIds = new Set();
  array(evidence, 'evidence').forEach((item, index) => {
    const path = `evidence[${index}]`;
    object(item, path);
    const allowed = new Set(['id', 'taskId', 'attemptId', 'planRevision', 'planDigest', 'kind', 'subject', 'producer', 'verifier', 'contributors', 'result', 'observedAt', 'provenance', 'trust', 'auditOnly', 'metadata', 'policyRevision', 'independence']);
    exactKeys(item, allowed, path);
    requireKeys(item, ['id', 'taskId', 'attemptId', 'planRevision', 'planDigest', 'kind', 'subject', 'producer', 'verifier', 'contributors', 'result', 'observedAt', 'provenance', 'trust', 'auditOnly', 'metadata', 'policyRevision'], path);
    const id = string(item.id, `${path}.id`, ID);
    if (evidenceIds.has(id)) throw new Error(`${path}.id is duplicated.`);
    evidenceIds.add(id);
    if (!taskIds.has(string(item.taskId, `${path}.taskId`, ID))) throw new Error(`${path}.taskId does not reference a task in the definition.`);
    string(item.attemptId, `${path}.attemptId`, ID);
    string(item.planRevision, `${path}.planRevision`);
    string(item.planDigest, `${path}.planDigest`, DIGEST);
    if (!evidenceKinds.has(item.kind)) throw new Error(`${path}.kind is unsupported by contract ${PORTABLE_CONTRACT_VERSION}.`);
    if (!evidenceResults.has(item.result)) throw new Error(`${path}.result is unsupported by contract ${PORTABLE_CONTRACT_VERSION}.`);
    string(item.producer, `${path}.producer`, ID);
    string(item.verifier, `${path}.verifier`, ID);
    if (!Array.isArray(item.contributors) || item.contributors.some((person) => typeof person !== 'string')) throw new Error(`${path}.contributors must be an array of strings.`);
    if (!['verified', 'unverified'].includes(item.trust)) throw new Error(`${path}.trust must be verified or unverified.`);
    if (typeof item.auditOnly !== 'boolean') throw new Error(`${path}.auditOnly must be boolean.`);
    object(item.subject, `${path}.subject`);
    object(item.metadata, `${path}.metadata`);
    string(item.policyRevision, `${path}.policyRevision`);
    string(item.observedAt, `${path}.observedAt`);
    string(item.provenance, `${path}.provenance`);
  });
  return evidenceIds;
}

function validateReportedEvaluations(evaluations, taskIds, requirementIds, evidenceIds) {
  const evaluationIds = new Set();
  array(evaluations, 'evaluations').forEach((item, index) => {
    const path = `evaluations[${index}]`;
    object(item, path);
    exactKeys(item, new Set(['id', 'requirementId', 'taskId', 'attemptId', 'planRevision', 'planDigest', 'evidenceIds', 'policyRevision', 'outcome', 'evaluatedAt', 'authority', 'qualification', 'alternative', 'metadata', 'approvalEvidenceId']), path);
    requireKeys(item, ['id', 'requirementId', 'taskId', 'attemptId', 'planRevision', 'planDigest', 'evidenceIds', 'policyRevision', 'outcome', 'evaluatedAt', 'authority', 'qualification', 'metadata'], path);
    const id = string(item.id, `${path}.id`, ID);
    if (evaluationIds.has(id)) throw new Error(`${path}.id is duplicated.`);
    evaluationIds.add(id);
    if (!requirementIds.has(string(item.requirementId, `${path}.requirementId`, ID))) throw new Error(`${path}.requirementId does not reference a definition requirement.`);
    if (!taskIds.has(string(item.taskId, `${path}.taskId`, ID))) throw new Error(`${path}.taskId does not reference a task in the definition.`);
    string(item.attemptId, `${path}.attemptId`, ID);
    string(item.planRevision, `${path}.planRevision`);
    string(item.planDigest, `${path}.planDigest`, DIGEST);
    if (!Array.isArray(item.evidenceIds) || item.evidenceIds.some((evidenceId) => !evidenceIds.has(evidenceId))) throw new Error(`${path}.evidenceIds must reference reported evidence.`);
    string(item.policyRevision, `${path}.policyRevision`);
    if (!evaluationOutcomes.has(item.outcome)) throw new Error(`${path}.outcome is unsupported by contract ${PORTABLE_CONTRACT_VERSION}.`);
    if (!['verified', 'unverified'].includes(item.qualification)) throw new Error(`${path}.qualification must be verified or unverified.`);
    string(item.authority, `${path}.authority`, ID);
    string(item.evaluatedAt, `${path}.evaluatedAt`);
    object(item.metadata, `${path}.metadata`);
  });
}

export function portablePlanFromBundle(input, name = 'portable-plan.json', projectId = 'portable-import') {
  const bundle = object(input, 'bundle');
  exactKeys(bundle, new Set(['contractVersion', 'definition', 'snapshot', 'evidence', 'evaluations']), 'bundle');
  if (bundle.contractVersion !== PORTABLE_CONTRACT_VERSION) throw new Error(`Unsupported portable plan contract version: ${String(bundle.contractVersion)}.`);
  requireKeys(bundle, ['definition', 'evidence', 'evaluations'], 'bundle');

  const definition = object(bundle.definition, 'definition');
  exactKeys(definition, new Set(['id', 'revision', 'digest', 'title', 'source', 'tasks', 'requirements', 'executionUnits', 'metadata']), 'definition');
  requireKeys(definition, ['id', 'revision', 'digest', 'title', 'source', 'tasks', 'requirements', 'executionUnits', 'metadata'], 'definition');
  const planId = string(definition.id, 'definition.id', ID);
  if (!planId.includes(':')) throw new Error('definition.id must be authority-qualified.');
  const revision = string(definition.revision, 'definition.revision');
  const digest = string(definition.digest, 'definition.digest', DIGEST);
  const title = string(definition.title, 'definition.title');
  const source = object(definition.source, 'definition.source');
  exactKeys(source, new Set(['authority', 'authorityId', 'ref', 'revision', 'digest', 'adapterVersion']), 'definition.source');
  requireKeys(source, ['authority', 'authorityId', 'ref', 'revision', 'digest', 'adapterVersion'], 'definition.source');
  if (!['markdown', 'native'].includes(source.authority)) throw new Error('definition.source.authority must be markdown or native.');
  string(source.authorityId, 'definition.source.authorityId', ID);
  const sourceRef = string(source.ref, 'definition.source.ref');
  string(source.revision, 'definition.source.revision');
  string(source.digest, 'definition.source.digest', DIGEST);
  string(source.adapterVersion, 'definition.source.adapterVersion');
  if (source.revision !== revision || source.digest !== digest) throw new Error('Definition and authoritative source revisions or digests disagree.');
  object(definition.metadata, 'definition.metadata');

  const taskIds = new Set();
  const definitionTasks = array(definition.tasks, 'definition.tasks');
  const rawTasks = definitionTasks.map((task, index) => {
    const path = `definition.tasks[${index}]`;
    object(task, path);
    exactKeys(task, new Set(['id', 'title', 'stage', 'authoredStatus', 'acceptance', 'source', 'dependencies', 'metadata', 'executionUnitId']), path);
    requireKeys(task, ['id', 'title', 'stage', 'authoredStatus', 'acceptance', 'source', 'dependencies', 'metadata'], path);
    const id = string(task.id, `${path}.id`, ID);
    if (!id.includes(':')) throw new Error(`${path}.id must be authority-qualified.`);
    if (taskIds.has(id)) throw new Error(`${path}.id is duplicated.`);
    taskIds.add(id);
    string(task.title, `${path}.title`);
    string(task.stage, `${path}.stage`);
    if (!authoredStatuses.has(task.authoredStatus)) throw new Error(`${path}.authoredStatus is unsupported.`);
    string(task.acceptance, `${path}.acceptance`);
    validateSource(task.source, sourceRef, source.authority, `${path}.source`);
    object(task.metadata, `${path}.metadata`);
    array(task.dependencies, `${path}.dependencies`).forEach((dep, depIndex) => {
      const depPath = `${path}.dependencies[${depIndex}]`;
      object(dep, depPath);
      exactKeys(dep, new Set(['taskId', 'predicate', 'requirementId']), depPath);
      requireKeys(dep, ['taskId', 'predicate'], depPath);
      string(dep.taskId, `${depPath}.taskId`, ID);
      if (!predicates.has(dep.predicate)) throw new Error(`${depPath}.predicate is unsupported.`);
      if (dep.requirementId !== undefined) string(dep.requirementId, `${depPath}.requirementId`, ID);
    });
    if (task.executionUnitId !== undefined) string(task.executionUnitId, `${path}.executionUnitId`, ID);
    return task;
  });
  for (const [index, task] of rawTasks.entries()) {
    for (const dep of task.dependencies) {
      if (!taskIds.has(dep.taskId)) throw new Error(`definition.tasks[${index}] dependency ${dep.taskId} is outside this plan.`);
      if (dep.taskId === task.id) throw new Error(`definition.tasks[${index}] cannot depend on itself.`);
    }
  }

  const requirements = array(definition.requirements, 'definition.requirements');
  const requirementIds = new Set();
  requirements.forEach((requirement, index) => {
    const path = `definition.requirements[${index}]`;
    object(requirement, path);
    exactKeys(requirement, new Set(['id', 'taskId', 'predicate', 'policyRevision', 'subject', 'domain', 'allowLocalAlternative']), path);
    requireKeys(requirement, ['id', 'taskId', 'predicate', 'policyRevision', 'subject'], path);
    const id = string(requirement.id, `${path}.id`, ID);
    if (requirementIds.has(id)) throw new Error(`${path}.id is duplicated.`);
    requirementIds.add(id);
    if (!taskIds.has(string(requirement.taskId, `${path}.taskId`, ID))) throw new Error(`${path}.taskId does not reference a task in the definition.`);
    if (!predicates.has(requirement.predicate)) throw new Error(`${path}.predicate is unsupported.`);
    string(requirement.policyRevision, `${path}.policyRevision`);
    object(requirement.subject, `${path}.subject`);
    if (requirement.domain !== undefined) string(requirement.domain, `${path}.domain`);
    if (requirement.allowLocalAlternative !== undefined && typeof requirement.allowLocalAlternative !== 'boolean') throw new Error(`${path}.allowLocalAlternative must be boolean.`);
  });
  for (const [index, task] of rawTasks.entries()) {
    for (const dep of task.dependencies) {
      if (dep.requirementId && !requirementIds.has(dep.requirementId)) throw new Error(`definition.tasks[${index}] dependency references an unknown requirement.`);
    }
  }
  array(definition.executionUnits, 'definition.executionUnits').forEach((unit, index) => {
    const path = `definition.executionUnits[${index}]`;
    object(unit, path);
    exactKeys(unit, new Set(['id', 'authority', 'canonicalId', 'taskIds', 'stages']), path);
    requireKeys(unit, ['id', 'authority', 'canonicalId', 'taskIds', 'stages'], path);
    string(unit.id, `${path}.id`, ID);
    string(unit.authority, `${path}.authority`, ID);
    string(unit.canonicalId, `${path}.canonicalId`, ID);
    if (!Array.isArray(unit.taskIds) || !unit.taskIds.every((id) => taskIds.has(id))) throw new Error(`${path}.taskIds must reference tasks in the definition.`);
    if (!Array.isArray(unit.stages) || !unit.stages.every((stage) => typeof stage === 'string' && stage.trim())) throw new Error(`${path}.stages must contain stage names.`);
  });

  if (bundle.snapshot !== undefined) {
    const snapshot = object(bundle.snapshot, 'snapshot');
    exactKeys(snapshot, new Set(['planId', 'planRevision', 'planDigest', 'observedAt', 'executions', 'metadata', 'subject']), 'snapshot');
    requireKeys(snapshot, ['planId', 'planRevision', 'planDigest', 'observedAt', 'executions', 'metadata', 'subject'], 'snapshot');
    string(snapshot.planId, 'snapshot.planId', ID);
    string(snapshot.planRevision, 'snapshot.planRevision');
    string(snapshot.planDigest, 'snapshot.planDigest', DIGEST);
    string(snapshot.observedAt, 'snapshot.observedAt');
    object(snapshot.metadata, 'snapshot.metadata');
    object(snapshot.subject, 'snapshot.subject');
    array(snapshot.executions, 'snapshot.executions').forEach((execution, index) => {
      const path = `snapshot.executions[${index}]`;
      object(execution, path);
      exactKeys(execution, new Set(['taskId', 'attemptId', 'state', 'executionUnitId']), path);
      requireKeys(execution, ['taskId', 'attemptId', 'state'], path);
      if (!taskIds.has(string(execution.taskId, `${path}.taskId`, ID))) throw new Error(`${path}.taskId does not reference a task in the definition.`);
      string(execution.attemptId, `${path}.attemptId`, ID);
      if (!executionStates.has(execution.state)) throw new Error(`${path}.state is unsupported.`);
      if (execution.executionUnitId !== undefined) string(execution.executionUnitId, `${path}.executionUnitId`, ID);
    });
  }

  const evidence = array(bundle.evidence, 'evidence');
  const evidenceIds = validateReportedEvidence(evidence, taskIds);
  validateReportedEvaluations(bundle.evaluations, taskIds, requirementIds, evidenceIds);
  const plan = {
    id: planId,
    title,
    path: sourceRef,
    sourceRef,
    sourceAuthority: source.authority,
    sourceAuthorityId: source.authorityId,
    sourceRevision: revision,
    sourceDigest: digest,
    contractVersion: bundle.contractVersion,
    contractDigest: PORTABLE_CONTRACT_DIGEST,
    portable: true,
    epics: [],
    warnings: ['Reported execution, evidence, and evaluation values are unverified by Wazi.'],
    tasks: rawTasks.map((task) => {
      const taskEvidence = evidence.filter((item) => item.taskId === task.id);
      const taskEvaluations = bundle.evaluations.filter((item) => item.taskId === task.id);
      const taskExecutions = bundle.snapshot?.executions.filter((item) => item.taskId === task.id) || [];
      return {
        id: task.id,
        sourceId: task.id,
        canonicalId: task.source.canonicalId || '',
        title: task.title,
        stage: task.stage,
        authoredStatus: task.authoredStatus,
        authoredStatusLabel: task.authoredStatus === 'complete' ? 'Marked done' : task.authoredStatus,
        status: task.authoredStatus,
        dependencies: task.dependencies.map((dependency) => dependency.taskId),
        reportedDependencies: task.dependencies,
        requirements: requirements.filter((item) => item.taskId === task.id),
        metadata: task.metadata,
        acceptance: task.acceptance,
        source: task.source,
        sourceBlock: task.source.raw || JSON.stringify(task, null, 2),
        line: task.source.line,
        owner: typeof task.metadata.owner === 'string' ? task.metadata.owner : '',
        reportedExecutions: taskExecutions,
        reportedEvidence: taskEvidence,
        reportedEvaluations: taskEvaluations,
        executionUnitId: task.executionUnitId || ''
      };
    })
  };
  return {
    id: projectId,
    name: name.replace(/\.json$/i, ''),
    imported: true,
    portable: true,
    portableBundle: bundle,
    plans: [plan]
  };
}

export function looksLikePortableJson(text) {
  const first = String(text).replace(/^\uFEFF/, '').trimStart()[0];
  return first === '{' || first === '[';
}
