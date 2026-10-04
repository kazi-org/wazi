// Only opaque discovered identities cross the browser boundary. Sources are captured by Go.
export async function hostRequest(path, body, signal) {
  const headers = {};
  if (body !== undefined) {
    const session = await fetch('/api/session', {signal, cache:'no-store'});
    if (!session.ok) throw new Error('The local Go host is unavailable.');
    headers['X-Wazi-Session'] = (await session.json()).token;
    headers['Content-Type'] = 'application/json';
  }
  const response = await fetch(path, {method:body === undefined?'GET':'POST', headers, body:body === undefined?undefined:JSON.stringify(body), signal, cache:'no-store'});
  const raw = await response.text();
  let data;
  try { data=JSON.parse(raw); } catch { throw new Error(response.ok?'The local host returned an invalid response.':raw.trim().slice(0,500)||`Local host returned ${response.status}.`); }
  if (!response.ok) throw new Error(data.error || data.message || `Local host returned ${response.status}.`);
  return data;
}
export function sameSelection(a, b) {
  return a.projectId === b.projectId && a.planPath === b.planPath && a.taskId === b.taskId;
}
