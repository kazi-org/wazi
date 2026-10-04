function stableId(prefix, value) {
  // Four FNV-style lanes keep IDs deterministic in both browsers and Node.
  const input = String(value);
  const hashes = [2166136261, 2166136261 ^ 0x9e3779b9, 2166136261 ^ 0x85ebca6b, 2166136261 ^ 0xc2b2ae35];
  for (let i = 0; i < input.length; i += 1) {
    const code = input.charCodeAt(i);
    for (let lane = 0; lane < hashes.length; lane += 1) {
      hashes[lane] = Math.imul(hashes[lane] ^ (code + lane * 31), 16777619) >>> 0;
    }
  }
  return `${prefix}-${hashes.map((hash) => hash.toString(16).padStart(8, '0')).join('')}`;
}

function withoutFences(markdown) {
  let inFence = false;
  let fenceChar = '';
  let fenceSize = 0;
  return String(markdown).split(/\r?\n/).map((line) => {
    const match = line.match(/^\s*(`{3,}|~{3,})/);
    if (match) {
      const marker = match[1];
      if (!inFence) {
        inFence = true;
        fenceChar = marker[0];
        fenceSize = marker.length;
      } else if (marker[0] === fenceChar && marker.length >= fenceSize) {
        inFence = false;
      }
      return '';
    }
    return inFence ? '' : line;
  });
}

const METADATA_FIELDS = 'Owner|Est|kind|verifies|delivers|lane|stage|blocked-by|deps|dependencies|acc|Acceptance|Status|fidelity|Wave|Scope|Contract|External gate|lifecycle|provider|canonical-id|canonical_id|canonical-task-id|canonical_task_id|canonical id|canonicalId|task-id|task_id|task id|taskId|uuid|policy|policy-revision|policy_revision|policy revision|attempt|revision|source|source-revision|source_revision|source revision|base|head|artifact|environment|producer|verifier|result|time|provenance|evidence|requirement|execution|service|claim-id|claim_id|claim id|dispatch-id|dispatch_id|dispatch id|owner-id|owner_id|owner id|adapter|schema|schema-version|schema_version|schema version|authority|status-source|status_source|status source';

function metadataFields(text) {
  const fields = [];
  const known = new Set(METADATA_FIELDS.split('|').map((name) => name.toLowerCase()));
  const pattern = new RegExp(`(?:^|\\s)(${METADATA_FIELDS}|[A-Za-z][A-Za-z0-9]*(?:[-_][A-Za-z0-9]+)+)\\s*:`, 'ig');
  let brackets = 0;
  let inlineCode = false;
  let cursor = 0;
  let match;
  while ((match = pattern.exec(text))) {
    for (; cursor < match.index; cursor += 1) {
      if (text[cursor] === '`') inlineCode = !inlineCode;
      else if (!inlineCode && text[cursor] === '[') brackets += 1;
      else if (!inlineCode && text[cursor] === ']') brackets = Math.max(0, brackets - 1);
    }
    const name = match[1].toLowerCase();
    const start = match.index + match[0].search(/\S/);
    const isFirstKnownField = known.has(name) && !fields.some((item) => known.has(item.name));
    if (brackets === 0 && !inlineCode && (known.has(name) || fields.some((item) => known.has(item.name)))) {
      fields.push({ name, start, valueStart: pattern.lastIndex });
    } else if (isFirstKnownField && brackets === 0 && !inlineCode) {
      fields.push({ name, start, valueStart: pattern.lastIndex });
    }
    cursor = pattern.lastIndex;
  }
  return fields;
}

function bracketValue(text, start) {
  let depth = 0;
  let quote = '';
  for (let i = start; i < text.length; i += 1) {
    const char = text[i];
    if (quote) {
      if (char === quote && text[i - 1] !== '\\') quote = '';
      continue;
    }
    const startsSingleQuoted = char === "'" && (i === start + 1 || /[\s,[;]/.test(text[i - 1]));
    if (char === '"' || startsSingleQuoted) quote = char;
    else if (char === '[') depth += 1;
    else if (char === ']') {
      depth -= 1;
      if (depth === 0) return { value: text.slice(start + 1, i).trim(), end: i };
    }
  }
  return { value: text.slice(start + 1).trim(), end: text.length };
}

function field(text, name) {
  const fields = metadataFields(text);
  const target = fields.find((item) => item.name === name.toLowerCase());
  if (!target) return undefined;
  const next = fields.find((item) => item.start > target.start);
  const end = next?.start ?? text.length;
  let start = target.valueStart;
  while (start < end && /\s/.test(text[start])) start += 1;
  if (text[start] === '[') {
    return bracketValue(text.slice(0, end), start).value;
  }
  const value = text.slice(start, end).trim();
  return ['acc', 'acceptance'].includes(name.toLowerCase()) ? value : value.replace(/[.;,]+\s*$/, '').trim();
}

function titleText(firstLine, hasId) {
  const fields = metadataFields(firstLine);
  const metadataStart = fields[0]?.start ?? firstLine.length;
  const text = firstLine.slice(0, metadataStart).trim();
  if (!hasId) return text;
  return text.replace(/^\S+\s*/, '').trim();
}

function listField(text, name) {
  const raw = field(text, name);
  if (!raw) return [];
  return raw.split(/[,\n]/).map((item) => item.trim().replace(/^['"`]|['"`]$/g, ''))
    .filter((item) => item && !/^none$/i.test(item));
}

function acceptanceField(text) {
  const compact = field(text, 'acc');
  const detailed = field(text, 'acceptance');
  return [compact, detailed].filter(Boolean).join('\n');
}

function taskEpic(lines, lineIndex) {
  for (let i = lineIndex - 1; i >= 0; i -= 1) {
    const heading = lines[i].match(/^\s{0,3}#{1,6}\s+(E\d+(?:[.-]\d+)*)\s*(?:[-–—:|]+\s*)?(.+)?$/i);
    if (heading) return { id: heading[1].toUpperCase(), title: (heading[2] || heading[1]).trim() };
  }
}

function parseHeading(markdown, sourcePath) {
  const lines = withoutFences(markdown);
  const heading = lines.find((line) => /^\s{0,3}#\s+/.test(line))?.match(/^\s{0,3}#\s+(.+?)\s*#*\s*$/);
  const basename = String(sourcePath).split(/[\\/]/).pop()?.replace(/\.[^.]*$/, '') || 'plan';
  const title = heading?.[1]?.trim() || basename;
  const epicMatch = title.match(/^(E\d+(?:[.-]\d+)*)\s*(?:[-–—:|]+\s*)?(.+)$/i);
  return epicMatch ? { title, epic: { id: epicMatch[1].toUpperCase(), title: epicMatch[2].trim() || epicMatch[1] } } : { title };
}

/** Browser-safe parser for skill-created plan Markdown. */
export function parsePlan(markdown, { path: sourcePath = 'plan.md', project = 'Project' } = {}) {
  const lines = withoutFences(markdown);
  const { title, epic: headingEpic } = parseHeading(markdown, sourcePath);
  const tasks = [];
  const epicsById = new Map();
  if (headingEpic) epicsById.set(headingEpic.id, headingEpic);

  for (let index = 0; index < lines.length; index += 1) {
    const match = lines[index].match(/^\s*[-*+]\s+\[([ xX~-])\]\s+(.+)$/);
    if (!match) continue;
    const lineNumber = index + 1;
    const marker = match[1].toLowerCase();
    const checked = marker === 'x';
    const firstLine = match[2].trim();
    const idMatch = firstLine.match(/^((?:[A-Z][A-Z0-9]*(?:[.-][A-Z0-9]+)+)|(?:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}))\b/i);
    const isUuidId = idMatch?.[1] && /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(idMatch[1]);
    const sourceId = idMatch?.[1] ? (isUuidId ? idMatch[1] : idMatch[1].toUpperCase()) : `line-${lineNumber}`;
    let end = index + 1;
    while (end < lines.length && !/^\s*[-*+]\s+\[[ xX~-]\]\s+/.test(lines[end]) && !/^\s{0,3}#{1,6}\s+/.test(lines[end])) {
      const line = lines[end];
      const continuation = /^\s{2,}/.test(line) || /^\s*(?:Acceptance|Stage|Status)\s*:/i.test(line) || /^\s*[A-Za-z][A-Za-z0-9_-]*\s*:/.test(line);
      if (line.trim() && !continuation) break;
      end += 1;
    }
    const sourceBlock = lines.slice(index, end).join('\n');
    const block = [firstLine, ...lines.slice(index + 1, end).map((line) => line.trim())].join('\n');
    const epic = taskEpic(lines, index) || headingEpic;
    if (epic) epicsById.set(epic.id, epic);
    const explicitStatus = field(block, 'status')?.toLowerCase();
    const dependencies = [...new Set([...listField(block, 'blocked-by'), ...listField(block, 'deps'), ...listField(block, 'dependencies')].map((item) => item.toUpperCase()))];
    const stage = field(block, 'stage') || block.match(/(?:^|[;\n])\s*Stage\s*:\s*([^;\n.]+)/i)?.[1]?.trim() || '';
    const owner = field(block, 'owner') || '';
    const metadata = Object.fromEntries(metadataFields(block).map(({ name }) => [name, field(block, name)]).filter(([, value]) => value !== undefined));
    const status = checked ? 'complete' : marker === '~' ? 'active' : marker === '-' ? 'blocked' : explicitStatus === 'blocked' ? 'blocked' : explicitStatus === 'active' || explicitStatus === 'in-progress' ? 'active' : 'pending';
    const source = `${sourcePath}:${lineNumber}`;
    tasks.push({
      id: stableId('task', `${project}\0${sourcePath}\0${sourceId}`),
      sourceId,
      canonicalId: metadata['canonical-id'] || metadata.canonical_id || metadata['canonical-task-id'] || metadata.canonical_task_id || metadata['canonical id'] || metadata.canonicalid || metadata['task-id'] || metadata.task_id || metadata['task id'] || metadata.taskid || metadata.uuid || (isUuidId ? idMatch[1] : ''),
      title: titleText(firstLine, Boolean(idMatch)) || sourceId,
      status,
      authoredStatus: marker === 'x' ? 'checked' : marker === '~' ? 'in-progress-marker' : marker === '-' ? 'blocked-marker' : 'unchecked',
      stage,
      epicId: epic?.id || '',
      epicTitle: epic?.title || '',
      owner,
      acceptance: acceptanceField(block),
      metadata,
      sourceBlock,
      dependencies,
      line: lineNumber,
      source,
    });
    index = end - 1;
  }
  const normalizedPath = String(sourcePath).replace(/\\/g, '/');
  return {
    id: stableId('plan', `${project}\0${normalizedPath}`),
    title,
    path: normalizedPath,
    project,
    tasks,
    epics: [...epicsById.values()],
    warnings: [],
  };
}
