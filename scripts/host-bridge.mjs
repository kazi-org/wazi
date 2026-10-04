// Fixed compatibility bridge: preserves the shared parser and reports private roots.
import { scanPlans } from './plans.mjs';
import fs from 'node:fs/promises';
import path from 'node:path';
const root = process.argv[2];
if (!root || process.argv.length !== 3) process.exit(2);
const scan = await scanPlans({ root });
const roots = {};
const queue = [{ path: path.resolve(root), depth: 0 }];
let seen = 0;
while (queue.length && seen < 12000) {
  const current = queue.shift(); seen++;
  let entries; try { entries = await fs.readdir(current.path, { withFileTypes: true }); } catch { continue; }
  for (const p of scan.projects) if (!roots[p.id] && p.plans.length && await Promise.all(p.plans.map(async plan => { try { return (await fs.stat(path.join(current.path, plan.path))).isFile(); } catch { return false; } })).then(xs => xs.every(Boolean))) roots[p.id] = current.path;
  const marker = entries.find(e => e.name === '.git');
  if (marker && marker.isDirectory() || marker && marker.isFile()) continue;
  if (current.depth >= 8) continue;
  for (const e of entries) if (e.isDirectory() && !e.name.startsWith('.') && !['node_modules','vendor','dist','build'].includes(e.name)) queue.push({path:path.join(current.path,e.name),depth:current.depth+1});
}
process.stdout.write(JSON.stringify({scan,roots}));
