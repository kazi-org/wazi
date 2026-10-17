// Fixed compatibility bridge: preserves the shared parser and reports private roots.
import { scanPlans } from './plans.mjs';
import fs from 'node:fs/promises';
import path from 'node:path';
import { createHash } from 'node:crypto';
const [root, source = 'local', projectId] = process.argv.slice(2);
if (!root || ![3, 5].includes(process.argv.length) || (process.argv.length === 5 && source !== 'origin')) process.exit(2);
const scan = await scanPlans({ root, source, projectId });
const roots = {};
const wanted = new Set(scan.projects.map(p => p.id));
const idForDirectory = dir => `project-${createHash('sha256').update(path.resolve(dir)).digest('hex').slice(0, 16)}`;
const queue = [{ path: path.resolve(root), depth: 0 }];
let seen = 0;
while (queue.length && seen < 12000) {
  const current = queue.shift(); seen++;
  let entries; try { entries = await fs.readdir(current.path, { withFileTypes: true }); } catch { continue; }
  const id = idForDirectory(current.path);
  if (wanted.has(id)) roots[id] = current.path;
  const marker = entries.find(e => e.name === '.git');
  if (marker && marker.isDirectory() || marker && marker.isFile()) continue;
  if (current.depth >= 8) continue;
  for (const e of entries) if (e.isDirectory() && !e.name.startsWith('.') && !['node_modules','vendor','dist','build'].includes(e.name)) queue.push({path:path.join(current.path,e.name),depth:current.depth+1});
}
process.stdout.write(JSON.stringify({scan,roots}));
