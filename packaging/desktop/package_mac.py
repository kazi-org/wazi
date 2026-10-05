#!/usr/bin/env python3
"""Assemble a local-development bundle from qualified build outputs. No publication."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import tarfile
import urllib.request

NODE_VERSION = '22.23.3'
NODE_ARCHIVE = f'node-v{NODE_VERSION}-darwin-arm64.tar.gz'
NODE_SHA256 = '23b25245dcfb9af7262f8ff142e9e2e0af025368117329e7a7458a51e5922f53'


def digest(path):
    h = hashlib.sha256()
    with path.open('rb') as source:
        for block in iter(lambda: source.read(1 << 20), b''):
            h.update(block)
    return h.hexdigest()


def assemble(repo, output, host, shell, assets, cache, sign=True):
    repo, output, cache = repo.resolve(), output.resolve(), cache.resolve()
    if any(p.is_symlink() for p in assets.rglob('*')):
        raise ValueError('frontend assets must not contain symbolic links')
    if output.exists():
        raise ValueError('output must be a new directory; preserve existing bundles')
    for p in [host, shell, assets / 'index.html']:
        if not p.is_file():
            raise ValueError('missing qualified build resource')
    cache.mkdir(parents=True, exist_ok=True)
    archive = cache / NODE_ARCHIVE
    if not archive.exists():
        part = cache / (NODE_ARCHIVE + '.part')
        with urllib.request.urlopen(f'https://nodejs.org/dist/v{NODE_VERSION}/{NODE_ARCHIVE}', timeout=60) as response, part.open('wb') as out:
            shutil.copyfileobj(response, out)
        if digest(part) != NODE_SHA256:
            raise ValueError('downloaded Node checksum mismatch')
        part.replace(archive)
    if digest(archive) != NODE_SHA256:
        raise ValueError('cached Node checksum mismatch')
    # Extract only the two explicitly named regular files, never paths from the archive.
    prefix = f'node-v{NODE_VERSION}-darwin-arm64/'
    contents = output / 'Contents'
    resources = contents / 'Resources'
    for relative in ['MacOS', 'Resources/host', 'Resources/runtime', 'Resources/parser/scripts', 'Resources/parser/src', 'Resources/notices']:
        (contents / relative).mkdir(parents=True, exist_ok=True)
    shutil.copy2(shell, contents / 'MacOS/Wazi')
    shutil.copy2(host, resources / 'host/wazi')
    shutil.copy2(repo / 'apps/macos/Info.plist', contents / 'Info.plist')
    shutil.copy2(repo / 'apps/macos/Wazi.icns', resources / 'Wazi.icns')
    shutil.copy2(repo / 'apps/macos/PHOSPHOR-LICENSE.txt', resources / 'notices/WaziIcon-LICENSE.txt')
    with tarfile.open(archive, 'r:gz') as tf:
        for name, dest in [('bin/node', resources / 'runtime/node'), ('LICENSE', resources / 'notices/Node-LICENSE')]:
            member = tf.getmember(prefix + name)
            if not member.isfile() or member.size > 128 << 20:
                raise ValueError('Node resource is not bounded regular content')
            source = tf.extractfile(member)
            if source is None:
                raise ValueError('Node resource unavailable')
            with source, dest.open('wb') as out:
                shutil.copyfileobj(source, out)
    for relative in ['scripts/host-bridge.mjs', 'scripts/plans.mjs', 'src/plan-parser.mjs']:
        shutil.copy2(repo / relative, resources / 'parser' / relative)
    shutil.copytree(assets, resources / 'web', symlinks=False)
    for package in ['react', 'react-dom', 'scheduler', '@phosphor-icons/react', '@fontsource-variable/manrope', '@fontsource/ibm-plex-mono']:
        license_file = repo / 'node_modules' / package / 'LICENSE'
        if not license_file.is_file():
            raise ValueError('runtime dependency license missing: ' + package)
        shutil.copy2(license_file, resources / 'notices' / (package.replace('/', '-') + '-LICENSE'))
    three_source = Path(os.environ.get('THREE_JS_SOURCE', str(repo / '.artifacts/three')))
    if not (three_source / 'LICENSE').is_file():
        raise ValueError('Three.js license missing; configure THREE_JS_SOURCE')
    shutil.copy2(three_source / 'LICENSE', resources / 'notices/Three-LICENSE')
    for name in ['LICENSE', 'THIRD_PARTY_NOTICES.md']:
        if (repo / name).is_file():
            shutil.copy2(repo / name, resources / 'notices' / name)
    for p in [contents / 'MacOS/Wazi', resources / 'host/wazi', resources / 'runtime/node']:
        p.chmod(0o755)
        if sign:
            subprocess.run(['codesign', '--force', '--sign', '-', str(p)], check=True, capture_output=True)
    manifest = {'format': 'wazi-mac-bundle/1', 'architecture': 'arm64', 'deploymentTarget': '14.0', 'signing': 'ad-hoc-development' if sign else 'unsigned-development', 'node': {'version': NODE_VERSION, 'archiveSHA256': NODE_SHA256}, 'sourceRevision': subprocess.check_output(['git', '-C', str(repo), 'rev-parse', 'HEAD'], text=True).strip(), 'rootExecutableIntegrity': 'codesign verifies the root executable; its signature is finalized after manifest creation', 'files': {str(p.relative_to(output)): digest(p) for p in sorted(output.rglob('*')) if p.is_file() and p != contents / 'MacOS/Wazi' and '_CodeSignature' not in p.relative_to(output).parts}}
    (resources / 'manifest.json').write_text(json.dumps(manifest, indent=2) + '\n')
    if sign:
        subprocess.run(['codesign', '--force', '--sign', '-', str(output)], check=True, capture_output=True)
        subprocess.run(['codesign', '--verify', '--deep', '--strict', str(output)], check=True, capture_output=True)
    return output


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--repo', type=Path, default=Path(__file__).resolve().parents[2])
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--host', type=Path, required=True)
    parser.add_argument('--shell', type=Path, required=True)
    parser.add_argument('--assets', type=Path, required=True)
    parser.add_argument('--cache', type=Path, required=True)
    parser.add_argument('--unsigned', action='store_true')
    args = parser.parse_args()
    print(assemble(args.repo, args.output, args.host, args.shell, args.assets, args.cache, not args.unsigned))


if __name__ == '__main__':
    main()
