import fs from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import { createHash } from 'node:crypto';
import { parsePlan } from '../src/plan-parser.mjs';

export { parsePlan } from '../src/plan-parser.mjs';

const MAX_DEPTH = 3;
const MAX_ENTRIES = 12000;
const MAX_SPLIT_PLANS = 120;
const MAX_FILE_BYTES = 1_000_000;
const SKIP_DIRS = new Set([
  '.git', '.hg', '.svn', 'node_modules', 'bower_components', 'vendor',
  'build', 'dist', 'out', 'coverage', 'tmp', 'temp', '.next', '.venv',
  'venv', 'Pods', 'DerivedData',
]);
const SKIP_NAME_PATTERNS = [/^wt[-_]/i, /worktrees/i, /-wt(?:[-_]|$)/i, /-t\d+(?:\.\d+)?-wt/i, /-task-t\d/i];

function stableId(prefix, value) {
  return `${prefix}-${createHash('sha256').update(String(value)).digest('hex').slice(0, 16)}`;
}

function relativePath(root, file) {
  return path.relative(root, file).split(path.sep).join('/');
}

async function isFile(file) {
  try { return (await fs.stat(file)).isFile(); } catch { return false; }
}

async function findPlanFiles(repoPath) {
  const out = [];
  for (const candidate of [path.join(repoPath, 'plan.md'), path.join(repoPath, 'docs', 'plan.md')]) {
    if (await isFile(candidate)) out.push(candidate);
  }
  let splitCapped = false;
  try {
    const splitPath = path.join(repoPath, 'docs', 'plans');
    const entries = (await fs.readdir(splitPath, { withFileTypes: true })).sort((a, b) => a.name.localeCompare(b.name));
    for (const entry of entries) {
      if (!entry.isFile() || !entry.name.toLowerCase().endsWith('.md')) continue;
      if (out.filter((file) => file.startsWith(splitPath + path.sep)).length >= MAX_SPLIT_PLANS) {
        splitCapped = true;
        break;
      }
      out.push(path.join(splitPath, entry.name));
    }
  } catch { /* No split plans in this repository. */ }
  return { files: [...new Set(out)].sort(), splitCapped };
}

async function discoverRepositories(root) {
  const repos = [];
  const visited = new Set();
  let entriesSeen = 0;
  async function walk(directory, depth) {
    if (depth > MAX_DEPTH || entriesSeen >= MAX_ENTRIES) return;
    let real;
    try { real = await fs.realpath(directory); } catch { return; }
    if (visited.has(real)) return;
    visited.add(real);
    let entries;
    try { entries = await fs.readdir(directory, { withFileTypes: true }); } catch { return; }
    const gitMarker = path.join(directory, '.git');
    let isGitRoot = false;
    try { isGitRoot = (await fs.stat(gitMarker)).isDirectory() || (await fs.stat(gitMarker)).isFile(); } catch { /* Not a Git repository root. */ }
    const { files: plans, splitCapped } = await findPlanFiles(directory);
    if (plans.length) repos.push({ path: directory, plans, splitCapped });
    // A Git root owns this subtree. Never rediscover docs/plan.md as a project
    // or enumerate the repository's source tree looking for nested plans.
    if (isGitRoot || plans.length) return;
    for (const entry of entries.sort((a, b) => a.name.localeCompare(b.name))) {
      if (!entry.isDirectory() || entry.name.startsWith('.') || SKIP_DIRS.has(entry.name) || SKIP_NAME_PATTERNS.some((pattern) => pattern.test(entry.name))) continue;
      entriesSeen += 1;
      if (entriesSeen >= MAX_ENTRIES) break;
      await walk(path.join(directory, entry.name), depth + 1);
    }
  }
  await walk(root, 0);
  return { repos, capped: entriesSeen >= MAX_ENTRIES };
}

function repoMapNames(repoMap) {
  if (!repoMap) return new Map();
  let value = repoMap;
  if (typeof repoMap === 'string') {
    try { value = JSON.parse(repoMap); } catch { return new Map(); }
  }
  const entries = value instanceof Map ? [...value.entries()] : Object.entries(value);
  return new Map(entries.flatMap(([name, repoPath]) => typeof repoPath === 'string' ? [[path.resolve(repoPath), name]] : []));
}

function applyDependencyStatus(plans) {
  const bySourceId = new Map();
  for (const plan of plans) for (const task of plan.tasks) {
    const matches = bySourceId.get(task.sourceId) || [];
    matches.push(task);
    bySourceId.set(task.sourceId, matches);
  }
  for (const plan of plans) for (const task of plan.tasks) {
    for (const dependency of task.dependencies) {
      const matches = bySourceId.get(dependency) || [];
      if (!matches.length) {
        plan.warnings.push(`Unresolved dependency ${dependency} at ${task.source}.`);
      } else if (task.status !== 'complete' && matches.some((candidate) => candidate.status !== 'complete')) {
        task.status = 'blocked';
      }
    }
  }
  for (const plan of plans) plan.warnings = [...new Set(plan.warnings)];
}

/** Read plans under root (default ~/Code), returning bounded task records. */
export async function scanPlans({ root = path.join(os.homedir(), 'Code'), repoMap } = {}) {
  const absoluteRoot = path.resolve(root);
  const warnings = [];
  const projects = [];
  const names = repoMapNames(repoMap);
  let discovery;
  try { discovery = await discoverRepositories(absoluteRoot); } catch (error) {
    return { projects, warnings: [`Could not scan plan root: ${error.message}`], scannedAt: new Date().toISOString() };
  }
  if (discovery.capped) warnings.push(`Repository discovery stopped at its ${MAX_ENTRIES} entry limit.`);

  for (const repo of discovery.repos.sort((a, b) => a.path.localeCompare(b.path))) {
    const name = names.get(path.resolve(repo.path)) || path.basename(repo.path) || repo.path;
    const plans = [];
    if (repo.splitCapped) warnings.push(`Split plan indexing for ${name} reached its ${MAX_SPLIT_PLANS} file limit.`);
    for (const file of repo.plans) {
      const relative = relativePath(repo.path, file);
      try {
        const stat = await fs.stat(file);
        if (stat.size > MAX_FILE_BYTES) {
          warnings.push(`Skipped oversized plan ${name}/${relative} (limit ${MAX_FILE_BYTES} bytes).`);
          continue;
        }
        const sourceBytes = await fs.readFile(file);
        const markdown = sourceBytes.toString('utf8');
        // Pass a project-qualified path into the shared parser to keep IDs unique across repos.
        const parsed = parsePlan(markdown, { path: `${path.resolve(repo.path)}/${relative}`, project: name });
        parsed.sourceDigest = `sha256:${createHash('sha256').update(sourceBytes).digest('hex')}`;
        plans.push(parsed);
        plans.at(-1).path = relative;
        plans.at(-1).project = name;
      } catch (error) {
        warnings.push(`Could not read plan ${name}/${relative}: ${error.message}`);
      }
    }
    if (!plans.length) continue;
    applyDependencyStatus(plans);
    projects.push({ id: stableId('project', path.resolve(repo.path)), name, plans });
  }
  projects.sort((a, b) => a.name.localeCompare(b.name) || a.id.localeCompare(b.id));
  return { projects, warnings, scannedAt: new Date().toISOString() };
}
