import { useState } from 'react';
import { getDesktopBridge, postWindowAction } from './desktop.mjs';
import './desktop.css';

function currentWindow() {
  return typeof window === 'undefined' ? null : window;
}

export default function DesktopChrome() {
  const nativeWindow = currentWindow();
  const qualified = Boolean(getDesktopBridge(nativeWindow));
  const [message, setMessage] = useState('');
  if (!qualified) return null;

  const invoke = action => {
    const result = postWindowAction(nativeWindow, action);
    setMessage(result.ok ? '' : 'Window control could not be sent to the app.');
  };

  return <div className="desktop-chrome">
    <div className="desktop-window-controls" aria-label="Window controls">
      <button className="desktop-control desktop-close" type="button" aria-label="Close window" title="Close window" onClick={() => invoke('close')}><span aria-hidden="true"/></button>
      <button className="desktop-control desktop-minimize" type="button" aria-label="Minimize window" title="Minimize window" onClick={() => invoke('minimize')}><span aria-hidden="true"/></button>
      <button className="desktop-control desktop-fullscreen" type="button" aria-label="Toggle full screen" title="Toggle full screen" onClick={() => invoke('fullscreen')}><span aria-hidden="true"/></button>
    </div>
    <div className="desktop-drag-region" aria-hidden="true" onPointerDown={event => {
      if (event.button === 0 && event.target === event.currentTarget) invoke('drag');
    }}/>
    <span className="desktop-chrome-status" role="status" aria-live="polite">{message}</span>
  </div>;
}
