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
  const data = await response.json();
  if (!response.ok) throw new Error(data.error || data.message || `Local host returned ${response.status}.`);
  return data;
}
export function sameSelection(a, b) {
  return a.projectId === b.projectId && a.planPath === b.planPath && a.taskId === b.taskId;
}
