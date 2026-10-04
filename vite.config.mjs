import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import { homedir } from 'node:os';
import path from 'node:path';
import { pathToFileURL } from 'node:url';

const threeSource = process.env.THREE_JS_SOURCE || path.join(homedir(), 'Code/dndungu/three.js');
const planApi = () => ({
  name: 'local-read-only-plans',
  configureServer(server) {
    server.middlewares.use('/api/plans', async (req, res) => {
      if (req.method !== 'GET') { res.statusCode = 405; res.end(); return; }
      if (!/^localhost(?::\d+)?$|^127\.0\.0\.1(?::\d+)?$|^\[::1\](?::\d+)?$/.test(req.headers.host || '')) { res.statusCode = 403; res.end(); return; }
      if (req.headers['sec-fetch-site'] === 'cross-site' || (req.headers.origin && req.headers.origin !== `http://${req.headers.host}`)) { res.statusCode = 403; res.end(); return; }
      res.setHeader('Content-Type', 'application/json');
      res.setHeader('Cache-Control', 'no-store');
      try {
        const { scanPlans } = await import(pathToFileURL(path.join(process.cwd(), 'scripts/plans.mjs')).href);
        res.end(JSON.stringify(await scanPlans({ root: process.env.WAZI_PLAN_ROOT || path.join(homedir(), 'Code') })));
      } catch (err) { res.statusCode = 500; res.end(JSON.stringify({error: 'Local plans could not be read. Import a Markdown plan instead.'})); console.error(err.message); }
    });
  },
});
export default defineConfig({
  plugins: [react(), planApi()],
  resolve: { alias: [
    { find: /^three$/, replacement: path.join(threeSource, 'build/three.module.js') },
    { find: /^three\/addons\//, replacement: path.join(threeSource, 'examples/jsm/') },
  ] },
  build: { outDir: 'dist/client' },
  server: { host: '127.0.0.1', allowedHosts: ['localhost'], fs: { allow: [process.cwd(), threeSource] } },
});
