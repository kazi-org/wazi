import test from 'node:test';
import assert from 'node:assert/strict';
import { DESKTOP_PROTOCOL, WINDOW_ACTIONS, getDesktopBridge, postWindowAction } from '../src/desktop.mjs';

function nativeWindow(postMessage = () => {}) {
  return {
    __WAZI_DESKTOP__: { platform: 'macos', protocol: DESKTOP_PROTOCOL },
    webkit: { messageHandlers: { waziWindow: { postMessage } } },
  };
}

test('browser mode has no qualified bridge and never dispatches native messages', () => {
  let calls = 0;
  const browser = { webkit: { messageHandlers: { waziWindow: { postMessage: () => calls++ } } } };
  assert.equal(getDesktopBridge(browser), null);
  assert.deepEqual(postWindowAction(browser, 'close'), { ok: false, reason: 'unavailable' });
  assert.equal(calls, 0);
});

test('marker and handler must exactly match the pinned native contract', () => {
  for (const marker of [
    { platform: 'ios', protocol: DESKTOP_PROTOCOL },
    { platform: 'macos', protocol: 'wazi-desktop/2' },
    { platform: 'macos', protocol: DESKTOP_PROTOCOL, extra: true },
  ]) assert.equal(getDesktopBridge({ __WAZI_DESKTOP__: marker, webkit: { messageHandlers: { waziWindow: { postMessage() {} } } } }), null);
  assert.equal(getDesktopBridge({ ...nativeWindow(), webkit: { messageHandlers: {} } }), null);
});

test('only the four contract actions send exact one-field messages', () => {
  const messages = [];
  const win = nativeWindow(message => messages.push(message));
  for (const action of WINDOW_ACTIONS) assert.deepEqual(postWindowAction(win, action), { ok: true });
  assert.deepEqual(messages, WINDOW_ACTIONS.map(action => ({ action })));
  assert.ok(messages.every(message => Object.keys(message).length === 1));
});

test('unknown actions are rejected without calling the native handler', () => {
  let calls = 0;
  assert.deepEqual(postWindowAction(nativeWindow(() => calls++), 'quit'), { ok: false, reason: 'invalid_action' });
  assert.deepEqual(postWindowAction(nativeWindow(() => calls++), '__proto__'), { ok: false, reason: 'invalid_action' });
  assert.equal(calls, 0);
});

test('native handler exceptions are returned as dispatch failures', () => {
  const result = postWindowAction(nativeWindow(() => { throw new Error('bridge unavailable'); }), 'minimize');
  assert.deepEqual(result, { ok: false, reason: 'dispatch_failed' });
});
