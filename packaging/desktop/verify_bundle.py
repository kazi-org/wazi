#!/usr/bin/env python3
"""Offline resource hash and executable architecture check; not runtime acceptance."""
import hashlib
import json
from pathlib import Path
import subprocess
import sys

bundle = Path(sys.argv[1]).resolve()
manifest = json.loads((bundle / 'Contents/Resources/manifest.json').read_text())
if manifest.get('format') != 'wazi-mac-bundle/1' or manifest.get('architecture') != 'arm64':
    raise SystemExit('unsupported bundle manifest')
for relative, expected in manifest['files'].items():
    logical = bundle / relative
    if logical.is_symlink():
        raise SystemExit('symbolic resource not allowed')
    path = logical.resolve()
    if not path.is_relative_to(bundle) or not path.is_file() or path.is_symlink():
        raise SystemExit('missing or unsafe bundle resource')
    if hashlib.sha256(path.read_bytes()).hexdigest() != expected:
        raise SystemExit('resource hash mismatch: ' + relative)
for relative in ['Contents/MacOS/Wazi', 'Contents/Resources/host/wazi', 'Contents/Resources/runtime/node']:
    info = subprocess.check_output(['file', str(bundle / relative)], text=True)
    if 'arm64' not in info:
        raise SystemExit('incorrect executable architecture')
print('PASS: bundle resource hashes and arm64 executables; runtime acceptance separate')

if manifest.get('signing') == 'ad-hoc-development':
    subprocess.run(['codesign', '--verify', '--deep', '--strict', str(bundle)], check=True, capture_output=True)
