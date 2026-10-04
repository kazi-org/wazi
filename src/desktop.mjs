export const DESKTOP_PROTOCOL = 'wazi-desktop/1';
export const WINDOW_ACTIONS = Object.freeze(['close', 'minimize', 'fullscreen', 'drag']);

function hasExactKeys(value, expected) {
  if (value === null || typeof value !== 'object' || Array.isArray(value)) return false;
  const keys = Object.keys(value).sort();
  return keys.length === expected.length && keys.every((key, index) => key === expected[index]);
}

/** Return the native message handler only when the shell's pinned marker is exact. */
export function getDesktopBridge(windowLike) {
  if (!windowLike || !hasExactKeys(windowLike.__WAZI_DESKTOP__, ['platform', 'protocol'])) return null;
  const marker = windowLike.__WAZI_DESKTOP__;
  if (marker.platform !== 'macos' || marker.protocol !== DESKTOP_PROTOCOL) return null;
  const handler = windowLike.webkit?.messageHandlers?.waziWindow;
  return handler && typeof handler.postMessage === 'function' ? handler : null;
}

/** Dispatch the contract's one-field message. Never fall back to browser APIs. */
export function postWindowAction(windowLike, action) {
  if (!WINDOW_ACTIONS.includes(action)) return { ok: false, reason: 'invalid_action' };
  const bridge = getDesktopBridge(windowLike);
  if (!bridge) return { ok: false, reason: 'unavailable' };
  try {
    bridge.postMessage({ action });
    return { ok: true };
  } catch {
    return { ok: false, reason: 'dispatch_failed' };
  }
}
